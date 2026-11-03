package subtitle

import (
	"context"
	"sort"
	"strings"
	"unicode"

	"github.com/benoit-pereira-da-silva/gogrammalecte/french"
	"github.com/benoit-pereira-da-silva/gogrammalecte/grammar"
	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell"
	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/tokenizer"
)

// sourceMap converts projected cue offsets to original Unicode code points.
type sourceMap struct {
	bytes []int
	runes []rune
}

func newSourceMap(input string) sourceMap {
	m := sourceMap{runes: []rune(input)}
	for i := range input {
		m.bytes = append(m.bytes, i)
	}
	m.bytes = append(m.bytes, len(input))
	return m
}
func (m sourceMap) position(c Cue, offset int) int {
	if offset >= len(c.Spans) {
		return sort.SearchInts(m.bytes, c.Spans[len(c.Spans)-1].End)
	}
	return sort.SearchInts(m.bytes, c.Spans[max(0, offset)].Start)
}
func (m sourceMap) errors(c Cue, errors []grammar.Error, withContext bool) []grammar.Error {
	out := make([]grammar.Error, 0, len(errors))
	for _, e := range errors {
		start, end, exact, ok := c.SourceRange(e.Start, e.End)
		if !ok {
			continue
		}
		e.Start, e.End = sort.SearchInts(m.bytes, start), sort.SearchInts(m.bytes, end)
		if !exact {
			e.Suggestions = []string{}
		}
		if withContext {
			e.Underlined = string(m.runes[e.Start:e.End])
			e.Before = string(m.runes[max(0, e.Start-80):e.Start])
			e.After = string(m.runes[e.End:min(len(m.runes), e.End+80)])
		}
		out = append(out, e)
	}
	return out
}

// Check checks each cue independently and returns original-file rune offsets.
// Suggestions spanning markup, escapes or physical line breaks are withheld.
func Check(ctx context.Context, input, format string, options grammar.ParseOptions) ([]grammar.Error, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cues, err := Project(input, format)
	if err != nil {
		return nil, err
	}
	engine, err := grammar.Default()
	if err != nil {
		return nil, err
	}
	m := newSourceMap(input)
	out := []grammar.Error{}
	for _, c := range cues {
		errors, err := engine.Check(ctx, c.Text, options)
		if err != nil {
			return nil, err
		}
		out = append(out, m.errors(c, errors, options.Context)...)
	}
	return out, nil
}

// Analyze includes morphology and labeled tokens. Sentence positions and error
// positions refer to the original file; token positions are sentence-relative.
func Analyze(ctx context.Context, input, format string, options grammar.ParseOptions) (*grammar.Analysis, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cues, err := Project(input, format)
	if err != nil {
		return nil, err
	}
	engine, err := grammar.Default()
	if err != nil {
		return nil, err
	}
	m := newSourceMap(input)
	out := &grammar.Analysis{ParagraphErrors: []grammar.Error{}, Sentences: []grammar.SentenceAnalysis{}}
	for _, c := range cues {
		part, err := engine.Analyze(ctx, c.Text, options)
		if err != nil {
			return nil, err
		}
		out.ParagraphErrors = append(out.ParagraphErrors, m.errors(c, part.ParagraphErrors, options.Context)...)
		for _, s := range part.Sentences {
			offset := s.Start
			start, end, _, ok := c.SourceRange(s.Start, s.End)
			if !ok {
				continue
			}
			s.Start, s.End = sort.SearchInts(m.bytes, start), sort.SearchInts(m.bytes, end)
			s.Sentence = input[start:end]
			s.Errors = m.errors(c, s.Errors, options.Context)
			for i := range s.Tokens {
				t := &s.Tokens[i]
				a, z := offset+t.Start, offset+t.End
				t.Start = m.position(c, a) - s.Start
				if z > a && z <= len(c.Spans) {
					t.End = sort.SearchInts(m.bytes, c.Spans[z-1].End) - s.Start
				} else {
					t.End = t.Start
				}
			}
			out.Sentences = append(out.Sentences, s)
		}
	}
	return out, nil
}

// Spell checks only cue text and returns original-file rune offsets.
func Spell(ctx context.Context, input, format string) ([]tokenizer.Token, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cues, err := Project(input, format)
	if err != nil {
		return nil, err
	}
	checker, err := graphspell.Default()
	if err != nil {
		return nil, err
	}
	m := newSourceMap(input)
	out := []tokenizer.Token{}
	for _, c := range cues {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for _, t := range checker.ParseParagraph(c.Text) {
			a, z, _, ok := c.SourceRange(t.Start, t.End)
			if !ok {
				continue
			}
			t.Start, t.End = sort.SearchInts(m.bytes, a), sort.SearchInts(m.bytes, z)
			out = append(out, t)
		}
	}
	return out, nil
}

// Format changes only contiguous literal text runs. Timing, metadata, tags,
// entities, escapes and physical line breaks remain byte-for-byte unchanged.
// Rules requiring context across these boundaries are conservatively skipped.
func Format(ctx context.Context, input, format string, options map[string]bool, rule string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	cues, err := Project(input, format)
	if err != nil {
		return "", err
	}
	formatter, err := french.NewTextFormatter()
	if err != nil {
		return "", err
	}
	type edit struct {
		start, end int
		text       string
	}
	edits := []edit{}
	for _, c := range cues {
		for i := 0; i < len(c.Spans); {
			if !c.Spans[i].Exact {
				i++
				continue
			}
			start, end := c.Spans[i].Start, c.Spans[i].End
			i++
			for i < len(c.Spans) && c.Spans[i].Exact && c.Spans[i].Start == end {
				end = c.Spans[i].End
				i++
			}
			// Preserve edge whitespace: formatter rules normally strip it as
			// paragraph indentation, but a run may border an inline tag.
			raw := input[start:end]
			literal := strings.TrimFunc(raw, unicode.IsSpace)
			if literal == "" {
				continue
			}
			leading := len(raw) - len(strings.TrimLeftFunc(raw, unicode.IsSpace))
			trailing := len(raw) - len(strings.TrimRightFunc(raw, unicode.IsSpace))
			start += leading
			end -= trailing
			var output string
			if rule != "" {
				output, _, err = formatter.Apply(ctx, input[start:end], rule)
			} else {
				output, err = formatter.FormatWithOptions(ctx, input[start:end], options)
			}
			if err != nil {
				return "", err
			}
			edits = append(edits, edit{start, end, output})
		}
	}
	var out strings.Builder
	previous := 0
	for _, e := range edits {
		out.WriteString(input[previous:e.start])
		out.WriteString(e.text)
		previous = e.end
	}
	out.WriteString(input[previous:])
	return out.String(), nil
}
