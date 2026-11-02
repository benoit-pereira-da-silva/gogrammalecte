package grammar

import (
	"sort"

	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell"
)

// TokenInfo is an independent, JSON-safe snapshot. Multi-token membership is
// represented by index bounds and morphologies rather than recursive pointers.
type TokenInfo struct {
	Index int    `json:"i"`
	Type  string `json:"sType"`
	Value string `json:"sValue"`
	Start int    `json:"nStart"`
	End   int    `json:"nEnd"`
	graphspell.TokenLabels
	Tags         []string        `json:"aTags,omitempty"`
	Multi        *MultiTokenInfo `json:"dMultiToken,omitempty"`
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
type MultiTokenInfo struct {
	Start      int      `json:"nTokenStart"`
	End        int      `json:"nTokenEnd"`
	Morphology []string `json:"lMorph"`
}
type labelLexicon struct{ Lexicon }

func (l labelLexicon) IsValidToken(word string) bool {
	if checker, ok := l.Lexicon.(interface{ IsValidToken(string) bool }); ok {
		return checker.IsValidToken(word)
	}
	return l.IsValid(word)
}
func copyPointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

// DescribeToken labels the final token state without mutating caches or
// retaining pointers into the active grammar analysis.
func DescribeToken(token *Token, checker Lexicon) TokenInfo {
	out := TokenInfo{
		Index: token.Index, Type: token.Type, Value: token.Value, Start: token.Start, End: token.End,
		TokenLabels:  graphspell.LabelToken(token.Type, token.Value, token.Morphology, labelLexicon{checker}),
		MultiStartTo: copyPointer(token.MultiStartTo), MultiEndFrom: copyPointer(token.MultiEndFrom),
		Immunity: token.Immunity, ToRemove: token.ToRemove, Merged: token.Merged,
		MergeUntil: copyPointer(token.MergeUntil), MergedValue: copyPointer(token.MergedValue), NewValue: copyPointer(token.NewValue), RealValue: token.RealValue,
	}
	for tag, present := range token.Tags {
		if present {
			out.Tags = append(out.Tags, tag)
		}
	}
	sort.Strings(out.Tags)
	if token.Multi != nil {
		out.Multi = &MultiTokenInfo{Start: token.Multi.Start, End: token.Multi.End, Morphology: cloneMorphologies(token.Multi.Morphology)}
	}
	return out
}
