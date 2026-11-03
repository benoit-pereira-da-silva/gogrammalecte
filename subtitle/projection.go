// Package subtitle projects timed-text payloads while preserving their source
// byte ranges. Timing, styling and metadata are never submitted as prose.
package subtitle

import (
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"path"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Span maps one projected Unicode code point to an original byte range.
// Exact is false for entities, normalized line breaks and subtitle escapes.
type Span struct {
	Start, End int
	Exact      bool
}

type Cue struct {
	Text  string
	Spans []Span
}

// SourceRange returns the source bytes and whether replacing them preserves
// every omitted tag, line break, entity and subtitle control sequence.
func (c Cue) SourceRange(start, end int) (int, int, bool, bool) {
	if start < 0 || end <= start || end > len(c.Spans) {
		return 0, 0, false, false
	}
	exact := true
	previous := c.Spans[start].Start
	for _, s := range c.Spans[start:end] {
		if !s.Exact || s.Start != previous {
			exact = false
		}
		previous = s.End
	}
	return c.Spans[start].Start, c.Spans[end-1].End, exact, true
}

func Normalize(format string) string {
	switch strings.ToLower(strings.TrimPrefix(format, ".")) {
	case "srt", "subrip":
		return "srt"
	case "vtt", "webvtt":
		return "vtt"
	case "ass", "ssa", "substation":
		return "ass"
	case "sbv", "subviewer":
		return "sbv"
	case "sub":
		return "sub"
	case "microdvd":
		return "microdvd"
	case "lrc":
		return "lrc"
	case "ttml", "dfxp":
		return "ttml"
	case "auto", "subtitles", "subtitle":
		return "auto"
	}
	return ""
}

func FromFilename(filename string) string {
	ext := path.Ext(strings.ToLower(filename))
	switch ext {
	case ".srt", ".vtt", ".webvtt", ".ass", ".ssa", ".sbv", ".sub", ".microdvd", ".lrc", ".ttml", ".dfxp":
		return Normalize(ext)
	}
	return ""
}

const clock = `(?:[0-9]+:[0-5][0-9]:[0-5][0-9]|[0-9]{2,}:[0-5][0-9])[.,][0-9]{2,3}`

var arrowTiming = regexp.MustCompile(`^[ \t]*` + clock + `[ \t]+-->[ \t]+` + clock + `(?:[ \t].*)?$`)
var commaTiming = regexp.MustCompile(`^[ \t]*` + clock + `[ \t]*,[ \t]*` + clock + `[ \t]*$`)
var frames = regexp.MustCompile(`^\{[0-9]+\}\{[0-9]+\}`)
var lyricTime = regexp.MustCompile(`^\[[0-9]+:[0-5][0-9](?:[.:][0-9]{1,3})?\]`)
var inlineTime = regexp.MustCompile(`^(?:` + clock + `|[0-9]+:[0-5][0-9](?:\.[0-9]{1,3})?)$`)
var captionTag = regexp.MustCompile(`(?i)^/?(?:b|i|u|font|c|v|lang|ruby|rt)(?:[. \t/]|$)`)
var drawingTag = regexp.MustCompile(`\\p([0-9]+)`)
var xmlRoot = regexp.MustCompile(`<(?:(?:[A-Za-z_][A-Za-z0-9_.-]*):)?tt(?:[ \t\r\n>])`)
var frameRate = regexp.MustCompile(`^\{[01]\}\{[01]\}[0-9]+(?:\.[0-9]+)?[ \t]*$`)

// Detect uses structural signatures rather than arbitrary numbers in prose.
func Detect(input string) string {
	input = strings.TrimPrefix(input, "\ufeff")
	if strings.HasPrefix(strings.TrimSpace(input), "WEBVTT") {
		return "vtt"
	}
	if xmlRoot.MatchString(input) {
		return "ttml"
	}
	for _, l := range lines(input) {
		text := strings.TrimSpace(input[l.start:l.end])
		if strings.EqualFold(text, "[Events]") {
			return "ass"
		}
		if arrowTiming.MatchString(text) {
			return "srt"
		}
		if commaTiming.MatchString(text) {
			return "sbv"
		}
		if frames.MatchString(text) {
			return "microdvd"
		}
		if lyricTime.MatchString(text) {
			return "lrc"
		}
	}
	return ""
}

// Project accepts UTF-8 SRT, WebVTT, ASS/SSA, SBV/SubViewer, MicroDVD,
// LRC and TTML/DFXP. An incomplete timing block is excluded, not diagnosed.
// Auto detection returns no cues when no subtitle signature is recognized.
func Project(input, format string) ([]Cue, error) {
	if !utf8.ValidString(input) {
		return nil, fmt.Errorf("subtitle input must contain valid UTF-8")
	}
	normalized := Normalize(format)
	if normalized == "auto" {
		normalized = Detect(input)
		if normalized == "" {
			return []Cue{}, nil
		}
	}
	switch normalized {
	case "srt", "vtt", "sbv":
		return timedBlocks(input, normalized), nil
	case "ass":
		return assCues(input), nil
	case "sub":
		if Detect(input) == "microdvd" {
			return microCues(input), nil
		}
		return timedBlocks(input, "sbv"), nil
	case "microdvd":
		return microCues(input), nil
	case "lrc":
		return lyricCues(input), nil
	case "ttml":
		return xmlCues(input), nil
	}
	return nil, fmt.Errorf("unsupported subtitle format %q", format)
}

type line struct{ start, end int }

func lines(input string) []line {
	result := []line{}
	for i := 0; i < len(input); {
		start := i
		for i < len(input) && input[i] != '\n' && input[i] != '\r' {
			i++
		}
		result = append(result, line{start, i})
		if i < len(input) {
			if input[i] == '\r' && i+1 < len(input) && input[i+1] == '\n' {
				i++
			}
			i++
		}
	}
	return result
}

type payloadMode struct{ ass, micro, subviewer, xml, preserve bool }
type cueBuilder struct {
	runes   []rune
	spans   []Span
	drawing bool
}

func (b *cueBuilder) add(r rune, s Span) { b.runes = append(b.runes, r); b.spans = append(b.spans, s) }
func (b *cueBuilder) space(start, end int) {
	if end > start {
		b.add(' ', Span{Start: start, End: end})
	}
}
func (b *cueBuilder) finish() (Cue, bool) {
	a, z := 0, len(b.runes)
	for a < z && unicode.IsSpace(b.runes[a]) {
		a++
	}
	for a < z && unicode.IsSpace(b.runes[z-1]) {
		z--
	}
	if a == z {
		return Cue{}, false
	}
	return Cue{Text: string(b.runes[a:z]), Spans: append([]Span(nil), b.spans[a:z]...)}, true
}
func flush(out *[]Cue, b *cueBuilder) {
	if cue, ok := b.finish(); ok {
		*out = append(*out, cue)
	}
	*b = cueBuilder{}
}

func (b *cueBuilder) write(input string, start, end int, mode payloadMode) {
	for i := start; i < end; {
		if !mode.xml && input[i] == '<' {
			if n := strings.IndexByte(input[i:end], '>'); n >= 0 {
				inside := input[i+1 : i+n]
				if captionTag.MatchString(inside) || inlineTime.MatchString(inside) {
					i += n + 1
					continue
				}
			}
		}
		if !mode.xml && input[i] == '{' && (mode.ass || mode.micro || strings.HasPrefix(input[i:end], `{\`)) {
			n := strings.IndexByte(input[i:end], '}')
			if n < 0 {
				return
			}
			block := input[i : i+n+1]
			if mode.ass {
				for _, m := range drawingTag.FindAllStringSubmatch(block, -1) {
					b.drawing = m[1] != "0"
				}
			}
			i += n + 1
			continue
		}
		if b.drawing {
			_, size := utf8.DecodeRuneInString(input[i:end])
			i += size
			continue
		}
		if mode.ass && input[i] == '\\' && i+1 < end && strings.ContainsRune("Nnh", rune(input[i+1])) {
			b.space(i, i+2)
			i += 2
			continue
		}
		if mode.micro && input[i] == '|' {
			b.space(i, i+1)
			i++
			continue
		}
		if mode.subviewer && strings.HasPrefix(input[i:end], "[br]") {
			b.space(i, i+4)
			i += 4
			continue
		}
		if input[i] == '&' {
			if n := strings.IndexByte(input[i:min(end, i+40)], ';'); n >= 0 {
				original := input[i : i+n+1]
				decoded := html.UnescapeString(original)
				if decoded != original {
					for _, r := range decoded {
						b.add(r, Span{Start: i, End: i + n + 1})
					}
					i += n + 1
					continue
				}
			}
		}
		r, size := utf8.DecodeRuneInString(input[i:end])
		if mode.xml && !mode.preserve && unicode.IsSpace(r) {
			next := i + size
			for next < end {
				r2, s2 := utf8.DecodeRuneInString(input[next:end])
				if !unicode.IsSpace(r2) {
					break
				}
				next += s2
			}
			if len(b.runes) > 0 && b.runes[len(b.runes)-1] == ' ' {
				b.spans[len(b.spans)-1].End = next
				b.spans[len(b.spans)-1].Exact = false
			} else {
				b.add(' ', Span{Start: i, End: next, Exact: next == i+1 && input[i] == ' '})
			}
			i = next
			continue
		}
		b.add(r, Span{Start: i, End: i + size, Exact: true})
		i += size
	}
}

func timedBlocks(input, format string) []Cue {
	out := []Cue{}
	b := cueBuilder{}
	active, skip := false, false
	sourceLines := lines(input)
	for index, l := range sourceLines {
		value := strings.TrimSpace(strings.TrimPrefix(input[l.start:l.end], "\ufeff"))
		if value == "" {
			flush(&out, &b)
			active = false
			skip = false
			continue
		}
		if skip {
			continue
		}
		if format == "vtt" && !active && (value == "NOTE" || strings.HasPrefix(value, "NOTE ") || strings.HasPrefix(value, "NOTE\t") || value == "STYLE" || value == "REGION") {
			skip = true
			continue
		}
		timing := arrowTiming.MatchString(value)
		if format == "sbv" {
			timing = commaTiming.MatchString(value)
		}
		if timing {
			flush(&out, &b)
			active = true
			continue
		}
		if strings.Contains(value, "-->") {
			flush(&out, &b)
			active = false
			continue
		}
		if index+1 < len(sourceLines) {
			next := strings.TrimSpace(input[sourceLines[index+1].start:sourceLines[index+1].end])
			if arrowTiming.MatchString(next) || (format == "sbv" && commaTiming.MatchString(next)) {
				flush(&out, &b)
				active = false
				continue
			}
		}
		if !active {
			continue
		}
		if len(b.spans) > 0 {
			b.space(b.spans[len(b.spans)-1].End, l.start)
		}
		b.write(input, l.start, l.end, payloadMode{subviewer: format == "sbv"})
	}
	flush(&out, &b)
	return out
}

func assCues(input string) []Cue {
	out := []Cue{}
	events := false
	fields := 10
	for _, l := range lines(input) {
		raw := input[l.start:l.end]
		trimmed := strings.TrimSpace(strings.TrimPrefix(raw, "\ufeff"))
		if strings.HasPrefix(trimmed, "[") {
			events = strings.EqualFold(trimmed, "[Events]")
			continue
		}
		if !events {
			continue
		}
		prefix, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		if strings.EqualFold(prefix, "Format") {
			names := strings.Split(value, ",")
			fields = 0
			if strings.EqualFold(strings.TrimSpace(names[len(names)-1]), "Text") {
				fields = len(names)
			}
			continue
		}
		if !strings.EqualFold(prefix, "Dialogue") || fields < 1 {
			continue
		}
		colon := strings.IndexByte(raw, ':')
		start := l.start + colon + 1
		found := true
		for i := 1; i < fields; i++ {
			n := strings.IndexByte(input[start:l.end], ',')
			if n < 0 {
				found = false
				break
			}
			start += n + 1
		}
		if !found {
			continue
		}
		b := cueBuilder{}
		b.write(input, start, l.end, payloadMode{ass: true})
		flush(&out, &b)
	}
	return out
}
func microCues(input string) []Cue {
	out := []Cue{}
	for _, l := range lines(input) {
		start := l.start
		if start == 0 && strings.HasPrefix(input, "\ufeff") {
			start += len("\ufeff")
		}
		raw := input[start:l.end]
		m := frames.FindStringIndex(raw)
		if m == nil || frameRate.MatchString(raw) {
			continue
		}
		b := cueBuilder{}
		b.write(input, start+m[1], l.end, payloadMode{micro: true})
		flush(&out, &b)
	}
	return out
}
func lyricCues(input string) []Cue {
	out := []Cue{}
	for _, l := range lines(input) {
		start := l.start
		if start == 0 && strings.HasPrefix(input, "\ufeff") {
			start += len("\ufeff")
		}
		timed := false
		for {
			m := lyricTime.FindStringIndex(input[start:l.end])
			if m == nil {
				break
			}
			timed = true
			start += m[1]
		}
		if !timed {
			continue
		}
		b := cueBuilder{}
		b.write(input, start, l.end, payloadMode{})
		flush(&out, &b)
	}
	return out
}
func xmlCues(input string) []Cue {
	out := []Cue{}
	decoder := xml.NewDecoder(strings.NewReader(input))
	b := cueBuilder{}
	depth, pDepth, bodyDepth, skip := 0, 0, 0, 0
	spaces := []bool{false}
	for {
		before := int(decoder.InputOffset())
		token, err := decoder.Token()
		after := int(decoder.InputOffset())
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		} // Keep only complete cues while a file is being edited.
		switch t := token.(type) {
		case xml.StartElement:
			depth++
			preserve := spaces[len(spaces)-1]
			for _, attr := range t.Attr {
				if attr.Name.Space == "http://www.w3.org/XML/1998/namespace" && attr.Name.Local == "space" {
					preserve = attr.Value == "preserve"
				}
			}
			spaces = append(spaces, preserve)
			if t.Name.Local == "metadata" && skip == 0 {
				skip = depth
			}
			if t.Name.Local == "body" && bodyDepth == 0 && skip == 0 {
				bodyDepth = depth
			}
			if t.Name.Local == "p" && pDepth == 0 && bodyDepth > 0 && skip == 0 {
				pDepth = depth
				b = cueBuilder{}
			}
			if pDepth > 0 && skip == 0 && t.Name.Local == "br" {
				b.space(before, after)
			}
		case xml.EndElement:
			if depth == pDepth && pDepth > 0 {
				flush(&out, &b)
				pDepth = 0
			}
			if depth == skip {
				skip = 0
			}
			if depth == bodyDepth {
				bodyDepth = 0
			}
			depth--
			spaces = spaces[:len(spaces)-1]
		case xml.CharData:
			if pDepth == 0 || skip > 0 {
				continue
			}
			if strings.HasPrefix(input[before:after], "<![CDATA[") {
				start, end := before+9, after-3
				for i := start; i < end; {
					r, size := utf8.DecodeRuneInString(input[i:end])
					b.add(r, Span{Start: i, End: i + size, Exact: r != '\n' && r != '\r'})
					i += size
				}
			} else {
				b.write(input, before, after, payloadMode{xml: true, preserve: spaces[len(spaces)-1]})
			}
		}
	}
	return out
}
