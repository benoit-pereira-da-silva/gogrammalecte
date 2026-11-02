package text

// Paragraph diagnostic rendering follows gc_core/py/text.py by Olivier R.
// in Grammalecte (GPLv3). French display strings remain reference data.

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Diagnostic preserves the original extensible JSON error schema. Rendering
// requires integer nStart/nEnd offsets and the fields documented by each API.
type Diagnostic map[string]any

func diagnosticInt(d Diagnostic, key string) (int, error) {
	switch value := d[key].(type) {
	case int:
		return value, nil
	case int64:
		return int(value), nil
	case float64:
		if !math.IsNaN(value) && !math.IsInf(value, 0) && value == math.Trunc(value) && value >= float64(-int(^uint(0)>>1)-1) && value <= float64(int(^uint(0)>>1)) {
			return int(value), nil
		}
	}
	return 0, fmt.Errorf("diagnostic %s must be an integer", key)
}
func diagnosticString(d Diagnostic, key string) (string, error) {
	value, ok := d[key].(string)
	if !ok {
		return "", fmt.Errorf("diagnostic %s must be a string", key)
	}
	return value, nil
}
func diagnosticSuggestions(d Diagnostic) ([]string, error) {
	if value, ok := d["aSuggestions"]; !ok || value == nil {
		return nil, nil
	}
	switch value := d["aSuggestions"].(type) {
	case []string:
		return value, nil
	case []any:
		out := make([]string, len(value))
		for i, item := range value {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("diagnostic suggestion must be a string")
			}
			out[i] = s
		}
		return out, nil
	}
	return nil, fmt.Errorf("diagnostic aSuggestions must be a string list")
}

// ReadableError renders one indexed diagnostic. Grammar diagnostics require
// sLineId and sRuleId; spelling diagnostics require sValue. Unlike Python's
// fallback dictionary repr, malformed records return an explicit error.
func ReadableError(d Diagnostic, spelling bool) (string, error) {
	index, err := diagnosticInt(d, "iError")
	if err != nil {
		return "", err
	}
	start, err := diagnosticInt(d, "nStart")
	if err != nil {
		return "", err
	}
	end, err := diagnosticInt(d, "nEnd")
	if err != nil {
		return "", err
	}
	result := fmt.Sprintf("* %d [%d:%d]  # ", index, start, end)
	if spelling {
		word, err := diagnosticString(d, "sValue")
		if err != nil {
			return "", err
		}
		result += word + ":"
	} else {
		line, err := diagnosticString(d, "sLineId")
		if err != nil {
			return "", err
		}
		rule, err := diagnosticString(d, "sRuleId")
		if err != nil {
			return "", err
		}
		message := "# error : message not found"
		if _, ok := d["sMessage"]; ok {
			message, err = diagnosticString(d, "sMessage")
			if err != nil {
				return "", err
			}
		}
		result += line + " / " + rule + ":\n  " + message
	}
	suggestions, err := diagnosticSuggestions(d)
	if err != nil {
		return "", err
	}
	if len(suggestions) > 0 {
		result += "\n  > Suggestions : " + strings.Join(suggestions, " | ")
	}
	if url, ok := d["URL"]; ok && url != nil && url != "" {
		value, err := diagnosticString(d, "URL")
		if err != nil {
			return "", err
		}
		result += "\n  > URL: " + value
	}
	return result, nil
}

// ReadableErrors wraps diagnostics with the reference hanging indentation.
// Spelling errors without an aSuggestions field are intentionally omitted.
func ReadableErrors(diagnostics []Diagnostic, width int, spelling bool) (string, error) {
	if width <= 0 {
		return "", fmt.Errorf("invalid wrapping width %d", width)
	}
	var result strings.Builder
	for _, d := range diagnostics {
		if _, ok := d["aSuggestions"]; spelling && !ok {
			continue
		}
		message, err := ReadableError(d, spelling)
		if err != nil {
			return "", err
		}
		for i, part := range strings.Split(message, "\n") {
			indent := "  "
			if i > 0 {
				indent = "    "
			}
			lines, err := wrapMessage(part, width, indent)
			if err != nil {
				return "", err
			}
			result.WriteString(strings.Join(lines, "\n"))
			result.WriteByte('\n')
		}
	}
	if result.Len() > 0 {
		result.WriteByte('\n')
	}
	return result.String(), nil
}
func nonnegativeRepeat(value string, n int) string {
	if n < 0 {
		return ""
	}
	return strings.Repeat(value, n)
}
func clippedIndex(index, length int) int {
	if index < 0 {
		index += length
	}
	if index < 0 {
		return 0
	}
	if index > length {
		return length
	}
	return index
}

// GenerateParagraph creates the underlined report and the ordered error list.
// As in Python, it adds iError numbering to the supplied diagnostic maps.
func GenerateParagraph(paragraph string, grammar, spelling []Diagnostic, width int) (string, []Diagnostic, error) {
	if paragraph == "" {
		return "", []Diagnostic{}, nil
	}
	lines, err := Wrap(paragraph, width)
	if err != nil {
		return "", nil, err
	}
	all := append(append([]Diagnostic{}, grammar...), spelling...)
	for _, d := range all {
		if _, err := diagnosticInt(d, "nStart"); err != nil {
			return "", nil, err
		}
		if _, err := diagnosticInt(d, "nEnd"); err != nil {
			return "", nil, err
		}
	}
	less := func(a, b Diagnostic) bool {
		x, _ := diagnosticInt(a, "nStart")
		y, _ := diagnosticInt(b, "nStart")
		return x < y
	}
	grammars := append([]Diagnostic{}, grammar...)
	spellings := append([]Diagnostic{}, spelling...)
	sort.SliceStable(grammars, func(i, j int) bool { return less(grammars[i], grammars[j]) })
	sort.SliceStable(spellings, func(i, j int) bool { return less(spellings[i], spellings[j]) })
	all = append(append([]Diagnostic{}, grammars...), spellings...)
	sort.SliceStable(all, func(i, j int) bool { return less(all[i], all[j]) })
	for i, d := range all {
		d["iError"] = i + 1
	}
	var result strings.Builder
	offset := 0
	for _, line := range lines {
		result.WriteString(line)
		result.WriteByte('\n')
		length := len([]rune(line))
		markers := ""
		grammarCount, spellingCount := 0, 0
		for _, d := range grammars {
			start, _ := diagnosticInt(d, "nStart")
			end, _ := diagnosticInt(d, "nEnd")
			relative := start - offset
			if relative >= length {
				break
			}
			grammarCount++
			if relative >= len([]rune(markers)) {
				markers += nonnegativeRepeat(" ", relative-len([]rune(markers))) + nonnegativeRepeat("^", end-start)
			}
		}
		for _, d := range spellings {
			start, _ := diagnosticInt(d, "nStart")
			end, _ := diagnosticInt(d, "nEnd")
			start -= offset
			end -= offset
			if start >= length {
				break
			}
			spellingCount++
			if end > len([]rune(markers)) {
				markers += nonnegativeRepeat(" ", end-len([]rune(markers)))
			}
			r := []rune(markers)
			markers = string(r[:clippedIndex(start, len(r))]) + nonnegativeRepeat("°", end-start) + string(r[clippedIndex(end, len(r)):])
		}
		if markers != "" {
			result.WriteString(markers)
			result.WriteByte('\n')
		}
		if grammarCount > 0 {
			rendered, err := ReadableErrors(grammars[:grammarCount], width, false)
			if err != nil {
				return "", nil, err
			}
			result.WriteString(rendered)
			grammars = grammars[grammarCount:]
		}
		if spellingCount > 0 {
			rendered, err := ReadableErrors(spellings[:spellingCount], width, true)
			if err != nil {
				return "", nil, err
			}
			result.WriteString(rendered)
			spellings = spellings[spellingCount:]
		}
		offset += length
	}
	return result.String(), all, nil
}

type NumberedLine struct {
	Number int
	Text   string
}
type LineSpan struct{ Number, Start, End int }

// CreateParagraphWithLines retains caller-supplied line numbers and inserts a
// separating space unless the preceding line ends with a space or dash.
func CreateParagraphWithLines(lines []NumberedLine) (string, []LineSpan) {
	var result strings.Builder
	spans := make([]LineSpan, 0, len(lines))
	offset := 0
	for i, line := range lines {
		value := strings.TrimRight(line.Text, "\r\n")
		r := []rune(value)
		if i+1 < len(lines) && (len(r) == 0 || !strings.ContainsRune("  -–—", r[len(r)-1])) {
			value += " "
		}
		end := offset + len([]rune(value))
		spans = append(spans, LineSpan{line.Number, offset, end})
		offset = end
		result.WriteString(value)
	}
	return result.String(), spans
}

// ConvertToXY changes the supplied maps in place: absolute nStart/nEnd become
// nStartY/nStartX/nEndY/nEndX. Unmapped coordinates follow the source behavior
// and are omitted while the original absolute fields are removed.
func ConvertToXY(grammar, spelling []Diagnostic, lines []LineSpan) error {
	all := append(append([]Diagnostic{}, grammar...), spelling...)
	for _, d := range all {
		if _, err := diagnosticInt(d, "nStart"); err != nil {
			return err
		}
		if _, err := diagnosticInt(d, "nEnd"); err != nil {
			return err
		}
	}
	for _, d := range all {
		start, _ := diagnosticInt(d, "nStart")
		end, _ := diagnosticInt(d, "nEnd")
		count := 0
		for i, line := range lines {
			count = i + 1
			if end <= line.End {
				d["nEndY"] = line.Number
				d["nEndX"] = end - line.Start
				break
			}
		}
		for i := count - 1; i >= 0; i-- {
			line := lines[i]
			if start >= line.Start {
				d["nStartY"] = line.Number
				d["nStartX"] = start - line.Start
				break
			}
		}
		delete(d, "nStart")
		delete(d, "nEnd")
	}
	return nil
}
