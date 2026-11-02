package french

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

type emptyLexicon struct{}

func (emptyLexicon) Lookup(string) bool    { return false }
func (emptyLexicon) Morph(string) []string { return nil }
func TestInflectionGeneration(t *testing.T) {
	b, e := GenerateInflectionData()
	if e != nil {
		t.Fatal(e)
	}
	var got, want any
	if e = json.Unmarshal(b, &got); e != nil {
		t.Fatal(e)
	}
	b, e = resources.ReadFile("data/mfsp.json")
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(b, &want)
	if !reflect.DeepEqual(got, want) {
		t.Fatal("native inflection generation differs from pinned tables")
	}
}
func TestConjugationGeneration(t *testing.T) {
	b, e := GenerateConjugationData(emptyLexicon{})
	if e != nil {
		t.Fatal(e)
	}
	var got, want map[string]any
	json.Unmarshal(b, &got)
	b, e = resources.ReadFile("data/conj.json")
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(b, &want)
	delete(got, "dVerbNames")
	delete(want, "dVerbNames")
	if !reflect.DeepEqual(got, want) {
		t.Fatal("native conjugation generation differs from pinned tables")
	}
}
func TestGeneratorErrors(t *testing.T) {
	for _, s := range []string{"word\tZ\n$\n", "_\tnom fem sg\tword\n", "word\tF\n"} {
		if _, e := CompileInflection(strings.NewReader(s)); e == nil {
			t.Errorf("invalid source accepted: %q", s)
		}
	}
	if _, e := GenerateConjugationData(nil); e == nil {
		t.Fatal("missing lexicon accepted")
	}
}

func TestThesaurusGeneration(t *testing.T) {
	b, e := GenerateThesaurusData()
	if e != nil {
		t.Fatal(e)
	}
	var got, want any
	json.Unmarshal(b, &got)
	b, e = resources.ReadFile("data/thesaurus.json")
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(b, &want)
	if !reflect.DeepEqual(got, want) {
		t.Fatal("native thesaurus generation differs from pinned data")
	}
}

type fixtureLexicon struct{}

func (fixtureLexicon) Lookup(w string) bool    { return w == "sang" || w == "sans" || w == "cent" }
func (fixtureLexicon) Morph(w string) []string { return []string{">" + w + "/:N:m:s"} }
func TestPhoneticGeneration(t *testing.T) {
	b, e := CompilePhonetic(strings.NewReader("sang sans\nsans cent\n"), fixtureLexicon{})
	if e != nil {
		t.Fatal(e)
	}
	var data struct {
		Words map[string]int      `json:"dWord"`
		Sets  [][]string          `json:"lSet"`
		Morph map[string][]string `json:"dMorph"`
	}
	json.Unmarshal(b, &data)
	if !reflect.DeepEqual(data.Sets[data.Words["sans"]], []string{"cent", "sang", "sans"}) {
		t.Fatal("overlapping phonetic sets not merged")
	}
	if !reflect.DeepEqual(data.Morph["cent"], []string{">cent/:N:m:s"}) {
		t.Fatal("morphology missing")
	}
}
