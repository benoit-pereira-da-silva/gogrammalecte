package french

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/dictionary"
)

func TestEmbeddedLexiconGeneration(t *testing.T) {
	lex, e := dictionary.Default()
	if e != nil {
		t.Fatal(e)
	}
	b, e := GenerateConjugationData(lex)
	if e != nil {
		t.Fatal(e)
	}
	var generated struct {
		Names map[string][]string `json:"dVerbNames"`
	}
	json.Unmarshal(b, &generated)
	loadConjugation()
	if len(generated.Names) != len(conjugation.Names) {
		t.Errorf("derived noun count got %d want %d", len(generated.Names), len(conjugation.Names))
	}
	for word, want := range conjugation.Names {
		got := copyStrings(generated.Names[word])
		want = copyStrings(want)
		slices.Sort(got)
		slices.Sort(want)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("generated nouns for %s: got %v want %v", word, got, want)
		}
	}
	b, e = GeneratePhoneticData(lex)
	if e != nil {
		t.Fatal(e)
	}
	var p struct {
		Words map[string]int      `json:"dWord"`
		Sets  [][]string          `json:"lSet"`
		Morph map[string][]string `json:"dMorph"`
	}
	json.Unmarshal(b, &p)
	loadPhonetic()
	if len(p.Words) != len(phonetic.Words) {
		t.Errorf("phonetic word count got %d want %d", len(p.Words), len(phonetic.Words))
	}
	for word, originalIndex := range phonetic.Words {
		index, ok := p.Words[word]
		if !ok || !reflect.DeepEqual(p.Sets[index], phonetic.Sets[originalIndex]) {
			t.Errorf("generated phonetic group differs for %s", word)
		}
		if !reflect.DeepEqual(p.Morph[word], phonetic.Morphologies[word]) {
			t.Errorf("generated phonetic morphologies differ for %s", word)
		}
	}
}
