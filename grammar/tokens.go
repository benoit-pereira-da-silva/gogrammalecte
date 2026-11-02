// Package grammar provides Grammalecte's native rule-execution primitives.
package grammar

import (
	"fmt"
	"unicode/utf8"

	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/tokenizer"
	"github.com/benoit-pereira-da-silva/gogrammalecte/text"
)

// Token is a mutable grammar token. All positions count Unicode code points.
// Morphology nil means uncached, while a non-nil empty slice is an explicit
// empty analysis. Pointer tokens share disambiguation state across rule graphs.
type Token struct {
	Index        int             `json:"i"`
	Type         string          `json:"sType"`
	Value        string          `json:"sValue"`
	Start        int             `json:"nStart"`
	End          int             `json:"nEnd"`
	Morphology   []string        `json:"lMorph,omitempty"`
	Tags         map[string]bool `json:"aTags,omitempty"`
	Multi        *MultiToken     `json:"dMultiToken,omitempty"`
	MultiStartTo *int            `json:"nMultiStartTo,omitempty"`
	MultiEndFrom *int            `json:"nMultiEndFrom,omitempty"`
	Immunity     string          `json:"sImmunity,omitempty"`
	ToRemove     bool            `json:"bToRemove,omitempty"`
	Merged       bool            `json:"bMerged,omitempty"`
	MergeUntil   *int            `json:"nMergeUntil,omitempty"`
	MergedValue  *string         `json:"sMergedValue,omitempty"`
	NewValue     *string         `json:"sNewValue,omitempty"`
	RealValue    string          `json:"sRealValue,omitempty"`
}

type MultiToken struct {
	Start      int      `json:"nTokenStart"`
	End        int      `json:"nTokenEnd"`
	Tokens     []*Token `json:"lTokens,omitempty"`
	Morphology []string `json:"lMorph"`
}

// Word represents the (position, text) pair consumed by regular-expression rules.
type Word struct {
	Start int
	Value string
}

func Tokenize(sentence string, withInfo bool) ([]*Token, error) {
	if !utf8.ValidString(sentence) {
		return nil, fmt.Errorf("tokenize: invalid UTF-8 input")
	}
	source := tokenizer.New("fr").Tokens(sentence, withInfo)
	out := make([]*Token, len(source))
	for i, t := range source {
		out[i] = &Token{Index: t.Index, Type: t.Type, Value: t.Value, Start: t.Start, End: t.End, Morphology: t.Morphology}
	}
	return out, nil
}

func SentenceBoundaries(input string) ([][2]int, error) {
	if !utf8.ValidString(input) {
		return nil, fmt.Errorf("sentence boundaries: invalid UTF-8 input")
	}
	source := text.SentenceBoundaries(input)
	out := make([][2]int, len(source))
	for i, span := range source {
		out[i] = [2]int{span.Start, span.End}
	}
	return out, nil
}

// TokenAt returns the closest token when index is outside the slice, or nil for
// an empty slice.
func TokenAt(tokens []*Token, index int) *Token {
	if len(tokens) == 0 {
		return nil
	}
	if index < 0 {
		index = 0
	}
	if index >= len(tokens) {
		index = len(tokens) - 1
	}
	return tokens[index]
}

// SliceText uses Python's code-point slice bounds. A nil left bound means the
// whole word in token helper APIs, including when right is non-nil.
func SliceText(value string, left, right *int) string {
	if left == nil {
		return value
	}
	r := []rune(value)
	start, end := sliceIndex(*left, len(r)), len(r)
	if right != nil {
		end = sliceIndex(*right, len(r))
	}
	if end < start {
		end = start
	}
	return string(r[start:end])
}
func sliceIndex(index, length int) int {
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
