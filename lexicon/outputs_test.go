package lexicon

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestLexicalOutputReference(t *testing.T) {
	var fixture struct {
		Header string
		Rows   []struct {
			Entry, Text, Morph, Dictionary        string
			Index                                 int
			Occurrences                           int64
			Duplicates, Multiple                  int
			Frequency                             float64
			FrequencyIndex, Metagraphe, Metaphone string
			Counts                                []int64
			Row                                   string
		}
		Ngrams []struct {
			Word     string
			N        int
			Expected []string
		}
		Verbs, Flags []struct {
			Input    string
			Expected []string
		}
		Fields []struct{ Name, Value, Expected string }
	}
	data, err := os.ReadFile("testdata/outputs.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	stats := NewStatistics(nil)
	stats.corpora = []CorpusStatistics{{ID: "G", Name: "Google 1-grams"}, {ID: "W", Name: "Wikipédia"}}
	if got := LexiconHeader(stats); got != fixture.Header {
		t.Fatalf("header = %q, want %q", got, fixture.Header)
	}
	for _, c := range fixture.Rows {
		e := ParseEntry(c.Entry)
		e.Stem = e.Fields["st"]
		if e.Stem == "" {
			e.Stem = e.Lemma
		}
		f := &Flexion{Entry: e, Text: c.Text, Morphology: c.Morph, Dictionary: c.Dictionary, Index: c.Index, Occurrences: c.Occurrences, Duplicates: c.Duplicates, OtherEntries: make([]*Entry, c.Multiple), Frequency: c.Frequency, FrequencyIndex: c.FrequencyIndex, MetagrapheCode: c.Metagraphe, MetaphoneCode: c.Metaphone}
		stats.counts[c.Text] = c.Counts
		if got := f.LexiconLine(stats); got != c.Row {
			t.Fatalf("lexicon line = %q, want %q", got, c.Row)
		}
	}
	for _, c := range fixture.Ngrams {
		if got := LexicalNgrams(c.Word, c.N); !reflect.DeepEqual(got, c.Expected) {
			t.Fatalf("LexicalNgrams(%q,%d) = %v, want %v", c.Word, c.N, got, c.Expected)
		}
	}
	for _, c := range fixture.Verbs {
		if got := VerbMultiMorph(c.Input); !reflect.DeepEqual(got, c.Expected) {
			t.Errorf("VerbMultiMorph(%q) = %q, want %q", c.Input, got, c.Expected)
		}
	}
	for _, c := range fixture.Flags {
		if got := LongFlags(c.Input); !reflect.DeepEqual(got, c.Expected) {
			t.Errorf("LongFlags(%q) = %q, want %q", c.Input, got, c.Expected)
		}
	}
	for _, c := range fixture.Fields {
		if got := FieldToHunspell(c.Name, c.Value); got != c.Expected {
			t.Errorf("FieldToHunspell(%q,%q) = %q, want %q", c.Name, c.Value, got, c.Expected)
		}
	}
}

func TestNgramIndex(t *testing.T) {
	d := &Dictionary{Flexions: []*Flexion{{Text: "abcd"}, {Text: "abcd"}, {Text: "abcde"}, {Text: "œuvre"}, {Text: "ab"}}}
	index, err := d.BuildNgrams(2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(index.Words, []string{"abcd", "abcde", "œuvre"}) {
		t.Fatalf("words = %q", index.Words)
	}
	want := []Ngram{{Text: "ab", Count: 3, WordIndices: []int{0, 1}}, {Text: "bc", Count: 1, WordIndices: []int{1}}, {Text: "œu", Count: 1, WordIndices: []int{2}}, {Text: "uv", Count: 1, WordIndices: []int{2}}}
	if !reflect.DeepEqual(index.Grams, want) {
		t.Fatalf("ngrams = %+v, want %+v", index.Grams, want)
	}
	if _, err := d.BuildNgrams(1); err == nil {
		t.Fatal("accepted invalid ngram length")
	}
}

var errOutput = errors.New("output unavailable")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errOutput }

func TestGeneratorWriterErrors(t *testing.T) {
	d, err := ReadDictionary(strings.NewReader("manger\tpo:v1_it____a\nchat/S.\tpo:nom\n"), strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	for name, write := range map[string]func() error{
		"grammar":     func() error { return d.WriteGrammarLexicon(failingWriter{}, "test") },
		"lexicon":     func() error { return d.WriteLexicon(failingWriter{}, "test", NewStatistics(nil)) },
		"conjugation": func() error { return d.WriteConjugations(failingWriter{}) },
		"declension":  func() error { return d.WriteDeclensions(failingWriter{}) },
		"dictionary": func() error {
			return d.WriteHunspell(failingWriter{}, io.Discard, "fr-classique", "test", HunspellOptions{})
		},
		"affixes": func() error {
			return d.WriteHunspell(io.Discard, failingWriter{}, "fr-classique", "test", HunspellOptions{})
		},
		"statistics": func() error { return d.CalculateFrequenciesReport(NewStatistics(nil), failingWriter{}) },
		"aliases":    func() error { return d.WriteAliasReport(failingWriter{}, 1) },
		"counts":     func() error { return NewStatistics([]string{"chat"}).Write(failingWriter{}) },
		"frequency changes": func() error {
			entry := ParseEntry("test\tfq:4")
			entry.OldFrequencyIndex = "3"
			return (&Dictionary{Entries: []*Entry{entry}}).WriteFrequencyChanges(failingWriter{}, io.Discard)
		},
	} {
		if err := write(); !errors.Is(err, errOutput) {
			t.Errorf("%s: got %v, want writer error", name, err)
		}
	}
}
