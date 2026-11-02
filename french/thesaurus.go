package french

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SynonymSet contains a source part-of-speech label and its ordered synonyms.
// Its JSON representation preserves the upstream [label, words] tuple.
type SynonymSet struct {
	Label string
	Words []string
}

func (s *SynonymSet) UnmarshalJSON(data []byte) error {
	var fields []json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if len(fields) != 2 {
		return fmt.Errorf("invalid synonym tuple: expected 2 fields, got %d", len(fields))
	}
	if err := json.Unmarshal(fields[0], &s.Label); err != nil {
		return err
	}
	return json.Unmarshal(fields[1], &s.Words)
}
func (s SynonymSet) MarshalJSON() ([]byte, error) { return json.Marshal([]any{s.Label, s.Words}) }

// Synonyms returns the matching synonym sets, using lowercase for an initially
// uppercase word. All returned slices are independent of the embedded tables.
func Synonyms(word string) []SynonymSet {
	if word == "" {
		return []SynonymSet{}
	}
	loadThesaurus()
	sets, ok := thesaurus[word]
	if !ok && firstUpper(word) {
		sets = thesaurus[strings.ToLower(word)]
	}
	out := make([]SynonymSet, len(sets))
	for i, s := range sets {
		out[i] = SynonymSet{s.Label, copyStrings(s.Words)}
	}
	return out
}
