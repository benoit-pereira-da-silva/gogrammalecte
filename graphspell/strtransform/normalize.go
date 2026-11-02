// Portions ported from Grammalecte's graphspell/str_transform.py by Olivier R.
// Distributed under GNU GPL version 3; see the repository LICENSE.
package strtransform

import (
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
)

var spellingReplacer = strings.NewReplacer("ſ", "s", "ﬃ", "ffi", "ﬄ", "ffl", "ﬀ", "ff", "ﬅ", "ft", "ﬁ", "fi", "ﬂ", "fl", "ﬆ", "st", "'", "’")
var simplification = map[rune]string{}

func init() {
	for _, group := range []struct{ letters, value string }{{"àâäáā", "a"}, {"éèêëē", "e"}, {"îïíìīÿyŷýỳȳ", "i"}, {"ôöóòō", "o"}, {"ûùüúū", "u"}, {"ç", "c"}, {"ñ", "n"}, {"k", "q"}, {"œ", "oe"}, {"æ", "ae"}, {"ſ", "s"}, {"ﬃ", "ffi"}, {"ﬄ", "ffl"}, {"ﬀ", "ff"}, {"ﬅ", "ft"}, {"ﬁ", "fi"}, {"ﬂ", "fl"}, {"ﬆ", "st"}, {"’'ʼ‘‛´`′‵՚ꞌꞋ-", ""}} {
		for _, r := range group.letters {
			simplification[r] = group.value
		}
	}
	for i, r := range []rune("⁰¹²³⁴⁵⁶⁷⁸⁹") {
		simplification[r] = string(rune('0' + i))
	}
	for i, r := range []rune("₀₁₂₃₄₅₆₇₈₉") {
		simplification[r] = string(rune('0' + i))
	}
}

// SpellingNormalization expands historical ligatures and normalizes to NFC.
func SpellingNormalization(word string) string {
	return norm.NFC.String(spellingReplacer.Replace(word))
}

// Lower performs full, language-neutral Unicode lowercasing.
func Lower(word string) string { return cases.Lower(language.Und).String(word) }

// Upper performs full Unicode uppercasing, including multi-character mappings.
func Upper(word string) string { return cases.Upper(language.Und).String(word) }

// Capitalize titlecases the first code point and lowercases the remainder.
func Capitalize(word string) string {
	r := []rune(word)
	if len(r) == 0 {
		return ""
	}
	first := string(r[0])
	lower := []rune(Lower(word))
	prefixLen := len([]rune(Lower(first)))
	return cases.Title(language.Und, cases.NoLower).String(first) + string(lower[prefixLen:])
}
func isUpper(r rune) bool {
	return unicode.IsUpper(r) || unicode.Is(unicode.Properties["Other_Uppercase"], r)
}
func isLower(r rune) bool {
	return unicode.IsLower(r) || unicode.Is(unicode.Properties["Other_Lowercase"], r)
}

// IsUpper follows Python's rule: every cased character is uppercase and at least
// one cased character is present. Numbers and punctuation are ignored.
func IsUpper(word string) bool {
	found := false
	for _, r := range word {
		if isLower(r) || unicode.IsTitle(r) {
			return false
		}
		if isUpper(r) {
			found = true
		}
	}
	return found
}

// IsTitle follows Python's cased-word title test, including uncased separators.
func IsTitle(word string) bool {
	previous, found := false, false
	for _, r := range word {
		if isUpper(r) || unicode.IsTitle(r) {
			if previous {
				return false
			}
			previous = true
			found = true
		} else if isLower(r) {
			if !previous {
				return false
			}
			previous = true
			found = true
		} else {
			previous = false
		}
	}
	return found
}

// IsAlpha reports whether a nonempty string contains only Unicode letters.
func IsAlpha(word string) bool {
	if word == "" {
		return false
	}
	for _, r := range word {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// IsDigit includes Python's non-decimal digit characters, such as superscripts.
func IsDigit(word string) bool {
	if word == "" {
		return false
	}
	for _, r := range word {
		if unicode.IsDigit(r) {
			continue
		}
		found := false
		for _, span := range extraDigitRanges {
			if r >= span[0] && r <= span[1] {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// SimplifyWord applies the spelling-distance normalization in the same order
// as the Python reference. In particular, ai becomes ei after ei becomes e.
func SimplifyWord(word string) string {
	var b strings.Builder
	for _, r := range Lower(word) {
		if v, ok := simplification[r]; ok {
			b.WriteString(v)
		} else {
			b.WriteRune(r)
		}
	}
	r := []rune(b.String())
	b.Reset()
	for i, c := range r {
		if i+1 == len(r) || c != r[i+1] || (c == 'e' && (i+2 == len(r) || r[i+2] != 'e')) {
			b.WriteRune(c)
		}
	}
	s := b.String()
	for _, pair := range [][2]string{{"eau", "o"}, {"au", "o"}, {"ei", "e"}, {"ai", "ei"}, {"ph", "f"}} {
		s = strings.ReplaceAll(s, pair[0], pair[1])
	}
	return s
}

// CleanWord reduces identical runs to two characters. Newlines are excluded,
// just as they are by the reference expression's dot without DOTALL.
func CleanWord(word string) string {
	var b strings.Builder
	var previous rune
	count := 0
	for _, r := range word {
		if r == previous {
			count++
		} else {
			previous = r
			count = 1
		}
		if count <= 2 || r == '\n' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// NumbersToExponent maps ASCII decimal digits to their superscript forms.
func NumbersToExponent(word string) string {
	digits := []rune("⁰¹²³⁴⁵⁶⁷⁸⁹")
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return digits[r-'0']
		}
		return r
	}, word)
}

// Ngrams returns slices of n Unicode code points, preserving Python slicing
// for zero and negative n as well as for ordinary positive widths.
func Ngrams(word string, n int) []string {
	r := []rune(word)
	result := []string{}
	for i := 0; i < len(r)-n+1; i++ {
		start := min(i, len(r))
		end := sliceIndex(i+n, len(r))
		if end < start {
			end = start
		}
		result = append(result, string(r[start:end]))
	}
	return result
}

// LongestCommonSubstring preserves the upstream tie-breaking order.
func LongestCommonSubstring(first, second string) string {
	return string(longestCommonSubstring([]rune(first), []rune(second)))
}

// NoStemming returns the supplied stem without transformation.
func NoStemming(_ string, stem string) string { return stem }
