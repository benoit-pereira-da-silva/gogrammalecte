// Package text provides paragraph and sentence segmentation compatible with
// Grammalecte's Python text helpers. Inputs must contain valid UTF-8.
//
// Based on gc_core/py/text.py by Olivier R. in Grammalecte (GPLv3).
// See testdata/text.json for the source revision and reference results.
package text

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// Span is a half-open interval [Start, End) in Unicode code points, matching
// Python string indices. These are neither UTF-8 byte nor UTF-16 unit offsets.
type Span struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// Paragraphs normalizes CRLF and CR to LF, then splits at each LF. It preserves
// whitespace and empty paragraphs, including the final one after a newline.
// Empty input produces one empty paragraph, as the Python generator does.
func Paragraphs(input string) []string {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")
	return strings.Split(input, "\n")
}

// The Python expression ends its punctuation branch with a positive lookahead.
// Go's standard regexp package cannot express lookahead, so the consuming
// prefix and the anchored continuation are evaluated separately. The original
// character classes are deliberately explicit: broad Unicode classes or \s
// would change the sentence boundaries.
var (
	sentenceCandidate = regexp.MustCompile(`[.?!…]+[»”’)]?[   ]+[»”’]?|[:;][   ]+`)
	sentenceFollowing = regexp.MustCompile(`^[«"“‘–—   ]*[A-ZÀÂÉÈÊÎÔÇ]`)
)

// sentenceEnds returns UTF-8 byte offsets for internal slicing. A failed
// continuation cannot hide another valid match inside its candidate: the
// consumed punctuation run has the same continuation, and the remaining
// consumed characters cannot start a sentence delimiter. Likewise, reducing
// the greedy whitespace run cannot make a failed continuation succeed.
func sentenceEnds(input string) []int {
	matches := sentenceCandidate.FindAllStringIndex(input, -1)
	ends := make([]int, 0, len(matches)+1)
	for _, match := range matches {
		first := input[match[0]]
		if first == ':' || first == ';' || sentenceFollowing.MatchString(input[match[1]:]) {
			ends = append(ends, match[1])
		}
	}
	// The Python generator always yields a final interval, even when empty.
	return append(ends, len(input))
}

// SentenceBoundaries returns the sentence intervals without normalizing or
// changing the text. Offsets count Unicode code points. The result always
// includes the final interval, which may be empty.
func SentenceBoundaries(input string) []Span {
	ends := sentenceEnds(input)
	spans := make([]Span, 0, len(ends))
	byteStart, pointStart := 0, 0
	for _, byteEnd := range ends {
		pointEnd := pointStart + utf8.RuneCountInString(input[byteStart:byteEnd])
		spans = append(spans, Span{Start: pointStart, End: pointEnd})
		byteStart, pointStart = byteEnd, pointEnd
	}
	return spans
}

// Sentences returns the substrings selected by SentenceBoundaries. It retains
// delimiters and spaces exactly as the Python generator does.
func Sentences(input string) []string {
	ends := sentenceEnds(input)
	sentences := make([]string, 0, len(ends))
	start := 0
	for _, end := range ends {
		sentences = append(sentences, input[start:end])
		start = end
	}
	return sentences
}
