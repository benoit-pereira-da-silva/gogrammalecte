package french

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// MorphToken supplies text and optional previously-disambiguated morphologies.
// A nil slice asks the analyser to query the lexicon; an explicit empty slice
// means the token has no remaining analyses.
type MorphToken struct {
	Value        string
	Morphologies []string
}

// IsApposition reports the reference noun/past-participle apposition condition.
func IsApposition(lexicon MorphologyLexicon, first, second string) bool {
	if utf8.RuneCountInString(second) < 2 {
		return true
	}
	a, _ := MorphologyPredicate("mbNomNotAdj", lexicon.Morph(second))
	b, _ := MorphologyPredicate("mbPpasNomNotAdj", lexicon.Morph(first))
	return a && b
}

// TokensAgree checks agreement, optionally requiring an adjectival analysis.
// Tokens without known morphology do not introduce an agreement failure.
func TokensAgree(lexicon MorphologyLexicon, first, second MorphToken, notOnlyNouns bool) bool {
	a, b := first.Morphologies, second.Morphologies
	if a == nil {
		a = lexicon.Morph(first.Value)
	}
	if len(a) == 0 {
		return true
	}
	if b == nil {
		b = lexicon.Morph(second.Value)
	}
	if len(b) == 0 {
		return true
	}
	if notOnlyNouns {
		adjective, _ := MorphologyPredicate("mbAdj", b)
		adjectiveOrNumber, _ := MorphologyPredicate("mbAdjNb", a)
		if !adjective && !adjectiveOrNumber {
			return false
		}
	}
	return Agreement(a, b)
}

// CombinedNounPOS returns the plural noun tag for a pair of words.
func CombinedNounPOS(lexicon MorphologyLexicon, first, second string) string {
	a, b := lexicon.Morph(first), lexicon.Morph(second)
	if len(a) == 0 || len(b) == 0 {
		return ":N:e:p"
	}
	g1, _ := GenderNumber(a)
	g2, _ := GenderNumber(b)
	if g1 == ":m" || g2 == ":m" {
		return ":N:m:p"
	}
	if g1 == ":f" || g2 == ":f" {
		return ":N:f:p"
	}
	return ":N:e:p"
}

// PossiblyUnit implements the reference measurement-unit heuristic.
func PossiblyUnit(value string) bool {
	if strings.ContainsAny(value, "µ/⁰¹²³⁴⁵⁶⁷⁸⁹Ωℓ·") {
		return true
	}
	r := []rune(value)
	if len(r) <= 1 || len(r) >= 16 || !unicode.IsLower(r[0]) {
		return false
	}
	lower := false
	for _, c := range r[1:] {
		if unicode.IsUpper(c) || unicode.IsTitle(c) {
			lower = false
			break
		}
		if unicode.IsLower(c) {
			lower = true
		}
	}
	if !lower {
		return true
	}
	for _, c := range r {
		if c >= '0' && c <= '9' {
			return true
		}
	}
	return false
}
