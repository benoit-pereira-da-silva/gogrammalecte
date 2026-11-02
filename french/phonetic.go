package french

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// PhoneticSimilar returns the registered phonetic group, trying lowercase when
// the first code point is uppercase. The returned slice is independent of the tables.
func PhoneticSimilar(word string) []string {
	loadPhonetic()
	if word == "" {
		return []string{}
	}
	if i, ok := phonetic.Words[word]; ok {
		return copyStrings(phonetic.Sets[i])
	}
	if firstUpper(word) {
		if i, ok := phonetic.Words[strings.ToLower(word)]; ok {
			return copyStrings(phonetic.Sets[i])
		}
	}
	return []string{}
}

// HasPhoneticSimilar reports whether a phonetic group exists and optionally
// contains a morphology matching pattern. Unsupported regexp syntax returns an error.
func HasPhoneticSimilar(word, pattern string) (bool, error) {
	loadPhonetic()
	if word == "" {
		return false, nil
	}
	if pattern == "" {
		_, ok := phonetic.Words[word]
		if !ok && firstUpper(word) {
			_, ok = phonetic.Words[strings.ToLower(word)]
		}
		return ok, nil
	}
	list, err := SelectPhoneticSimilar(word, pattern)
	return len(list) > 0, err
}

// SelectPhoneticSimilar filters a phonetic group by a standard-library regexp.
// One entry is retained per matching morphology; source duplicates are preserved.
func SelectPhoneticSimilar(word, pattern string) ([]string, error) {
	similar := PhoneticSimilar(word)
	if pattern == "" {
		return similar, nil
	}
	r, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	result := []string{}
	for _, candidate := range similar {
		for _, morph := range phonetic.Morphologies[candidate] {
			if r.MatchString(morph) {
				result = append(result, candidate)
			}
		}
	}
	return result, nil
}

// ArePhoneticallySimilar compares group identities using upstream casing rules.
func ArePhoneticallySimilar(word, other string) bool {
	if word == "" || other == "" {
		return false
	}
	a := phoneticSet(word)
	return a >= 0 && a == phoneticSet(other)
}
func phoneticSet(word string) int {
	loadPhonetic()
	if i, ok := phonetic.Words[word]; ok {
		return i
	}
	if firstUpper(word) {
		if i, ok := phonetic.Words[strings.ToLower(word)]; ok {
			return i
		}
		if allUpper(word) {
			if i, ok := phonetic.Words[capitalize(word)]; ok {
				return i
			}
		}
	}
	return -1
}
func firstUpper(s string) bool { r, _ := utf8.DecodeRuneInString(s); return unicode.IsUpper(r) }
func allUpper(s string) bool {
	cased := false
	for _, r := range s {
		if unicode.IsLower(r) || unicode.IsTitle(r) {
			return false
		}
		if unicode.IsUpper(r) {
			cased = true
		}
	}
	return cased
}
func capitalize(s string) string {
	r := []rune(strings.ToLower(s))
	if len(r) > 0 {
		r[0] = unicode.ToTitle(r[0])
	}
	return string(r)
}
