package french

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// GeneratePhoneticData rebuilds phonetic groups from the embedded source and lexicon.
func GeneratePhoneticData(lexicon MorphologyLexicon) ([]byte, error) {
	r, e := sourceReader("phonet_simil.txt")
	if e != nil {
		return nil, e
	}
	defer r.Close()
	return CompilePhonetic(r, lexicon)
}

// CompilePhonetic compiles groups, preserving source group order and using sorted
// member iteration for deterministic indices of newly merged groups.
func CompilePhonetic(r io.Reader, lexicon MorphologyLexicon) ([]byte, error) {
	if lexicon == nil {
		return nil, fmt.Errorf("phonetic generation requires a lexicon")
	}
	groups := []map[string]bool{}
	e := readSource(r, func(_ int, line string) error {
		words := strings.Fields(line)
		for i := 0; i < len(words); i++ {
			word := words[i]
			if strings.HasSuffix(word, "er") && IsVerb(word) {
				words = append(words, SimilarInfinitiveForms(word)...)
			}
		}
		group := map[string]bool{}
		for _, word := range words {
			group[word] = true
		}
		groups = append(groups, group)
		return nil
	})
	if e != nil {
		return nil, e
	}
	baseCount := len(groups)
	wordIndex := map[string]int{}
	morphologies := map[string][]string{}
	for i := 0; i < baseCount; i++ {
		for _, word := range sortedSet(groups[i]) {
			if !lexicon.Lookup(word) {
				continue
			}
			if previous, ok := wordIndex[word]; ok {
				merged := map[string]bool{}
				for w := range groups[previous] {
					merged[w] = true
				}
				for w := range groups[i] {
					merged[w] = true
				}
				wordIndex[word] = len(groups)
				groups = append(groups, merged)
			} else {
				wordIndex[word] = i
			}
		}
	}
	lists := make([][]string, len(groups))
	for i, g := range groups {
		lists[i] = sortedSet(g)
	}
	for word := range wordIndex {
		morphologies[word] = copyStrings(lexicon.Morph(word))
	}
	return json.Marshal(map[string]any{"dWord": wordIndex, "lSet": lists, "dMorph": morphologies})
}
func sortedSet(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for word := range set {
		out = append(out, word)
	}
	sort.Strings(out)
	return out
}
