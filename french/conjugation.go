package french

import (
	"errors"
	"fmt"
)

// ErrUnknownVerb and ErrUnknownTense distinguish missing reference verbs and invalid tense identifiers.
var (
	ErrUnknownVerb  = errors.New("unknown verb")
	ErrUnknownTense = errors.New("unknown tense")
)
var tenseIndex = map[string]int{":P": 0, ":Q": 1, ":Ip": 2, ":Iq": 3, ":Is": 4, ":If": 5, ":K": 6, ":Sp": 7, ":Sq": 8, ":E": 9}

// IsVerb reports whether verb is an exact infinitive in the embedded table.
func IsVerb(verb string) bool { loadConjugation(); _, ok := conjugation.Verbs[verb]; return ok }

// GetConjugation returns an inflected form. A missing person yields an empty
// string; an unknown verb or tense yields an explicit error.
func GetConjugation(verb, tense, person string) (string, error) {
	tags, ok := verbTags(verb)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrUnknownVerb, verb)
	}
	return conjugateTags(verb, tags, tense, person)
}

// HasConjugation reports whether a nonempty transformation is defined.
func HasConjugation(verb, tense, person string) (bool, error) {
	tags, ok := verbTags(verb)
	if !ok {
		return false, nil
	}
	if tense == ":Y" {
		return true, nil
	}
	i, ok := tenseIndex[tense]
	if !ok {
		return false, fmt.Errorf("%w: %s", ErrUnknownTense, tense)
	}
	return conjugation.Patterns[tense][tags[i]][person] != "", nil
}

// VerbType returns the original compact grammatical classification.
func VerbType(verb string) (string, bool) {
	loadConjugation()
	v, ok := conjugation.Verbs[verb]
	if !ok {
		return "", false
	}
	return conjugation.Types[v[0]], true
}

// DerivedNames returns registered deverbal nouns or the available gendered past participles.
func DerivedNames(verb string) []string {
	loadConjugation()
	if names, ok := conjugation.Names[verb]; ok {
		return copyStrings(names)
	}
	tags, ok := verbTags(verb)
	if !ok {
		return []string{}
	}
	out := []string{mustConjugate(verb, tags, ":Q", ":m:s")}
	for _, person := range []string{":f:s", ":m:p", ":f:p"} {
		if hasTags(tags, ":Q", person) {
			out = append(out, mustConjugate(verb, tags, ":Q", person))
		}
	}
	if len(out) > 1 {
		return out
	}
	return []string{}
}

// SimilarInfinitiveForms returns the conjugations selected by the upstream first-group heuristic.
func SimilarInfinitiveForms(verb string) []string {
	tags, ok := verbTags(verb)
	out := []string{}
	if !ok {
		return out
	}
	for _, p := range [][2]string{{":Iq", ":2s"}, {":Iq", ":3s"}, {":Iq", ":3p"}, {":Is", ":1s"}, {":Ip", ":2p"}, {":Iq", ":2p"}} {
		if hasTags(tags, p[0], p[1]) {
			out = append(out, mustConjugate(verb, tags, p[0], p[1]))
		}
	}
	return out
}
func verbTags(verb string) ([]int, bool) {
	loadConjugation()
	v, ok := conjugation.Verbs[verb]
	if !ok {
		return nil, false
	}
	return conjugation.Tags[v[1]], true
}
func conjugateTags(verb string, tags []int, tense, person string) (string, error) {
	if tense == ":Y" {
		return verb, nil
	}
	i, ok := tenseIndex[tense]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrUnknownTense, tense)
	}
	return modifySuffix(verb, conjugation.Patterns[tense][tags[i]][person]), nil
}
func hasTags(tags []int, tense, person string) bool {
	if tense == ":Y" {
		return true
	}
	i, ok := tenseIndex[tense]
	return ok && conjugation.Patterns[tense][tags[i]][person] != ""
}
func mustConjugate(verb string, tags []int, tense, person string) string {
	value, err := conjugateTags(verb, tags, tense, person)
	if err != nil {
		panic(err)
	}
	return value
}
