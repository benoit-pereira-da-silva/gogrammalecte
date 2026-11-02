package french

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// The names in this table preserve upstream symbolic morphology identifiers.
var morphologyPatterns = map[string]*regexp.Regexp{
	"GenderNumber": regexp.MustCompile(":[mfe]:[spi]"),
	"NA":           regexp.MustCompile(":[NA]"),
	"NAs":          regexp.MustCompile(":[NA].*:s"),
	"NAp":          regexp.MustCompile(":[NA].*:p"),
	"NAi":          regexp.MustCompile(":[NA].*:i"),
	"NAsi":         regexp.MustCompile(":[NA].*:[si]"),
	"NApi":         regexp.MustCompile(":[NA].*:[pi]"),
	"NAm":          regexp.MustCompile(":[NA].*:m"),
	"NAf":          regexp.MustCompile(":[NA].*:f"),
	"NAe":          regexp.MustCompile(":[NA].*:e"),
	"NAme":         regexp.MustCompile(":[NA].*:[me]"),
	"NAfe":         regexp.MustCompile(":[NA].*:[fe]"),
	"NAms":         regexp.MustCompile(":[NA].*:m.*:s"),
	"NAfs":         regexp.MustCompile(":[NA].*:f.*:s"),
	"NAes":         regexp.MustCompile(":[NA].*:e.*:s"),
	"NAmes":        regexp.MustCompile(":[NA].*:[me].*:s"),
	"NAfes":        regexp.MustCompile(":[NA].*:[fe].*:s"),
	"NAmsi":        regexp.MustCompile(":[NA].*:m.*:[si]"),
	"NAfsi":        regexp.MustCompile(":[NA].*:f.*:[si]"),
	"NAesi":        regexp.MustCompile(":[NA].*:e.*:[si]"),
	"NAmesi":       regexp.MustCompile(":[NA].*:[me].*:[si]"),
	"NAfesi":       regexp.MustCompile(":[NA].*:[fe].*:[si]"),
	"NAmp":         regexp.MustCompile(":[NA].*:m.*:p"),
	"NAfp":         regexp.MustCompile(":[NA].*:f.*:p"),
	"NAep":         regexp.MustCompile(":[NA].*:e.*:p"),
	"NAmep":        regexp.MustCompile(":[NA].*:[me].*:p"),
	"NAfep":        regexp.MustCompile(":[NA].*:[me].*:p"),
	"NAmpi":        regexp.MustCompile(":[NA].*:m.*:[pi]"),
	"NAfpi":        regexp.MustCompile(":[NA].*:f.*:[pi]"),
	"NAepi":        regexp.MustCompile(":[NA].*:e.*:[pi]"),
	"NAmepi":       regexp.MustCompile(":[NA].*:[me].*:[pi]"),
	"NAfepi":       regexp.MustCompile(":[NA].*:[fe].*:[pi]"),
	"AD":           regexp.MustCompile(":[AB]"),
	"Vconj":        regexp.MustCompile(":[123][sp]"),
	"Vconj123":     regexp.MustCompile(":V[123].*:[123][sp]"),
	"NVconj":       regexp.MustCompile(":(?:N|[123][sp])"),
	"NAVconj":      regexp.MustCompile(":(?:N|A|[123][sp])"),
	"NP":           regexp.MustCompile(":(?:M[12P]|T)"),
	"NPm":          regexp.MustCompile(":(?:M[12P]|T):m"),
	"NPf":          regexp.MustCompile(":(?:M[12P]|T):f"),
	"NPe":          regexp.MustCompile(":(?:M[12P]|T):e"),
}

// LemmaOfMorphology extracts the initial lemma from a morphology string.
// Python's Unicode word class includes letters, numbers and underscore.
func LemmaOfMorphology(morph string) string {
	if !strings.HasPrefix(morph, ">") {
		return ""
	}
	r := []rune(morph[1:])
	if len(r) == 0 || !wordRune(r[0]) {
		return ""
	}
	end := 1
	for end < len(r) && (wordRune(r[end]) || r[end] == '-') {
		end++
	}
	return string(r[:end])
}
func wordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' }
func nounNotAdjective(morph string) bool {
	for i := 0; i < len(morph); {
		n := strings.Index(morph[i:], ":N")
		if n < 0 {
			return false
		}
		i += n + 2
		if !strings.HasPrefix(morph[i:], ":A") {
			return true
		}
	}
	return false
}

// MatchMorphology evaluates a named upstream pattern. The two negative
// lookahead patterns use explicit suffix checks, leaving regexp standard-only.
func MatchMorphology(name, morph string) (bool, error) {
	switch name {
	case "Lemma":
		return LemmaOfMorphology(morph) != "", nil
	case "NnotA":
		return nounNotAdjective(morph), nil
	case "PNnotA":
		return nounNotAdjective(morph) || strings.Contains(morph, ":Q"), nil
	}
	r, ok := morphologyPatterns[name]
	if !ok {
		return false, fmt.Errorf("unknown morphology pattern: %s", name)
	}
	return r.MatchString(morph), nil
}

// GenderNumber combines gender and number, marking ambiguous values epicene or invariant.
func GenderNumber(morphologies []string) (gender, number string) {
	for _, morph := range morphologies {
		match := morphologyPatterns["GenderNumber"].FindString(morph)
		if match == "" {
			continue
		}
		g, n := match[:2], match[2:]
		if gender == "" {
			gender = g
		} else if gender != g {
			gender = ":e"
		}
		if number == "" {
			number = n
		} else if number != n {
			number = ":i"
		}
	}
	return
}

// Agreement reports whether two analyses can agree in gender and number.
func Agreement(first, second []string) bool {
	g1, n1 := GenderNumber(first)
	g2, n2 := GenderNumber(second)
	return (n1 == ":i" || n2 == ":i" || n1 == n2) && (g1 == ":e" || g2 == ":e" || g1 == g2)
}

// HasConjugationTag searches for the requested conjugation substring in any analysis.
func HasConjugationTag(morphologies []string, tag string) bool {
	for _, m := range morphologies {
		if strings.Contains(m, tag) {
			return true
		}
	}
	return false
}
func every(morphologies []string, p func(string) bool) bool {
	for _, m := range morphologies {
		if !p(m) {
			return false
		}
	}
	return true
}
func some(morphologies []string, p func(string) bool) bool {
	for _, m := range morphologies {
		if p(m) {
			return true
		}
	}
	return false
}
func contains(tag string) func(string) bool {
	return func(m string) bool { return strings.Contains(m, tag) }
}
func matches(name string) func(string) bool {
	return func(m string) bool {
		ok, e := MatchMorphology(name, m)
		if e != nil {
			panic(e)
		}
		return ok
	}
}

// MorphologyPredicate evaluates an upstream predicate identifier. "is" predicates
// require every analysis to match; "mb" predicates require at least one, except
// for the explicitly exclusionary noun/proper-name predicates.
func MorphologyPredicate(name string, morphologies []string) (bool, error) {
	allTags := map[string]string{"isNom": ":N", "isAdj": ":A", "isInv": ":i", "isSg": ":s", "isPl": ":p", "isEpi": ":e", "isMas": ":m", "isFem": ":f"}
	anyTags := map[string]string{"mbNom": ":N", "mbAdj": ":A", "mbMG": ":G", "mbInv": ":i", "mbSg": ":s", "mbPl": ":p", "mbEpi": ":e", "mbMas": ":m", "mbFem": ":f"}
	allPatterns := map[string]string{"isNomNotAdj": "NnotA", "isNomAdj": "NA", "isNomVconj": "NVconj"}
	anyPatterns := map[string]string{"mbAdjNb": "AD", "mbNomAdj": "NA", "mbPpasNomNotAdj": "PNnotA", "mbVconj": "Vconj", "mbVconj123": "Vconj123", "mbNpr": "NP"}
	if tag, ok := allTags[name]; ok {
		return every(morphologies, contains(tag)), nil
	}
	if tag, ok := anyTags[name]; ok {
		return some(morphologies, contains(tag)), nil
	}
	if pattern, ok := allPatterns[name]; ok {
		return every(morphologies, matches(pattern)), nil
	}
	if pattern, ok := anyPatterns[name]; ok {
		return some(morphologies, matches(pattern)), nil
	}
	switch name {
	case "mbNomNotAdj":
		return !some(morphologies, contains(":A")) && some(morphologies, contains(":N")), nil
	case "mbNprMasNotFem":
		return !some(morphologies, matches("NPf")) && some(morphologies, matches("NPm")), nil
	}
	return false, fmt.Errorf("unknown morphology predicate: %s", name)
}
