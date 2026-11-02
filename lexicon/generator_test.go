package lexicon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

type generatorFixture struct {
	Entries []struct {
		Line       string
		Attributes map[string]string
		Morph      []string
		LexMorph   string
		Natural    []string
		Hunspell   []string
		Simplified string
	}
	Rules []struct {
		Line                             string
		Suffix                           bool
		Name, Cut, Add, Flags, Condition string
		Morph                            []string
		LexMorph                         string
		Replication                      bool
		Simplified                       string
	}
	Flags []struct {
		Name        string
		Suffix, Mix bool
		Count       int
	}
	Settings string
	Full     struct {
		Entries, Flexions     int
		SHA256, GrammarSHA256 string
	}
}

func loadGeneratorFixture(t *testing.T) generatorFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/generator.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture generatorFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func TestEntryReference(t *testing.T) {
	for _, c := range loadGeneratorFixture(t).Entries {
		e := ParseEntry(c.Line)
		for name, want := range c.Attributes {
			if name == "iz" {
				name = "is"
			}
			if name == "iD" {
				name = "id"
			}
			if got := e.Attribute(name); got != want {
				t.Fatalf("%q attribute %q = %q, want %q", c.Line, name, got, want)
			}
		}
		for mode, want := range c.Morph {
			if got := e.Morphology(mode); got != want {
				t.Fatalf("%q mode %d = %q, want %q", c.Line, mode, got, want)
			}
			if got := e.HunspellLine(mode, false); got != c.Hunspell[mode] {
				t.Fatalf("Hunspell(%q, %d) = %q, want %q", c.Line, mode, got, c.Hunspell[mode])
			}
		}
		if got := e.HunspellLine(2, true); got != c.Simplified {
			t.Fatalf("simplified(%q) = %q, want %q", c.Line, got, c.Simplified)
		}
		if got := e.LexicalMorphology(); got != c.LexMorph {
			t.Fatalf("lexical morphology(%q) = %q, want %q", c.Line, got, c.LexMorph)
		}
		if got := e.NaturalKey(); !reflect.DeepEqual(got[:], c.Natural) {
			t.Fatalf("natural key(%q) = %q, want %q", c.Line, got, c.Natural)
		}
	}
}

func TestAffixRuleReference(t *testing.T) {
	for _, c := range loadGeneratorFixture(t).Rules {
		r, err := ParseAffixRule(c.Line)
		if err != nil {
			t.Fatal(err)
		}
		if r.Suffix != c.Suffix || r.Name != c.Name || r.Cut != c.Cut || r.Add != c.Add || r.Flags != c.Flags || r.Condition != c.Condition {
			t.Fatalf("rule parse %q: %+v", c.Line, r)
		}
		for mode, want := range c.Morph {
			if got := r.Morphology(mode); got != want {
				t.Fatalf("%q mode %d = %q, want %q", c.Line, mode, got, want)
			}
		}
		if got := r.LexicalMorphology(); got != c.LexMorph {
			t.Fatalf("lexical morphology(%q) = %q, want %q", c.Line, got, c.LexMorph)
		}
		if got := r.IsReplication(); got != c.Replication {
			t.Fatalf("replication(%q) = %t", c.Line, got)
		}
		if got := r.HunspellLine(2, true); got != c.Simplified {
			t.Fatalf("simplified(%q) = %q, want %q", c.Line, got, c.Simplified)
		}
		if got := r.HunspellLine(2, false); got != c.Line {
			t.Fatalf("roundtrip(%q) = %q", c.Line, got)
		}
	}
}

func TestFullNativeInflectionGeneration(t *testing.T) {
	fixture := loadGeneratorFixture(t)
	d, err := EmbeddedDictionary()
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Entries) != fixture.Full.Entries {
		t.Fatalf("entries = %d, want %d", len(d.Entries), fixture.Full.Entries)
	}
	if d.Settings != fixture.Settings {
		t.Fatal("settings differ")
	}
	if len(d.Flags) != len(fixture.Flags) {
		t.Fatalf("flags = %d, want %d", len(d.Flags), len(fixture.Flags))
	}
	for i, want := range fixture.Flags {
		got := d.Flags[i]
		if got.Name != want.Name || got.Suffix != want.Suffix || got.Mix != want.Mix || len(got.Rules) != want.Count {
			t.Fatalf("flag %d = %+v, want %+v", i, got, want)
		}
	}
	if err := d.Generate(); err != nil {
		t.Fatal(err)
	}
	if len(d.Flexions) != fixture.Full.Flexions {
		t.Fatalf("flexions = %d, want %d", len(d.Flexions), fixture.Full.Flexions)
	}
	hash, grammarHash := sha256.New(), sha256.New()
	for _, f := range d.Flexions {
		fmt.Fprintf(hash, "%s\t%s\t%s\t%d\t%d\t%d\n", f.Text, f.Morphology, f.Dictionary, f.Index, f.Duplicates, len(f.OtherEntries))
		fmt.Fprint(grammarHash, f.GrammarCheckerLine())
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != fixture.Full.SHA256 {
		t.Fatalf("inflection SHA-256 = %s, want %s", got, fixture.Full.SHA256)
	}
	if got := hex.EncodeToString(grammarHash.Sum(nil)); got != fixture.Full.GrammarSHA256 {
		t.Fatalf("grammar lexicon SHA-256 = %s, want %s", got, fixture.Full.GrammarSHA256)
	}
}

func TestCorpusStatisticsReference(t *testing.T) {
	var fixture struct {
		Corpora, Words []string
		Counts         map[string][]int64
		Metadata       [][]any
		Info           string
		Totals         map[string]int64
	}
	data, err := os.ReadFile("testdata/statistics.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	stats := NewStatistics(fixture.Words)
	for i, corpus := range fixture.Corpora {
		if err := stats.Add(strings.NewReader(corpus), fmt.Sprint(i), fmt.Sprint("corpus ", i)); err != nil {
			t.Fatal(err)
		}
	}
	for word, want := range fixture.Counts {
		if got := stats.Counts(word); !reflect.DeepEqual(got, want) {
			t.Fatalf("counts(%q) = %v, want %v", word, got, want)
		}
	}
	for word, want := range fixture.Totals {
		if got := stats.Occurrences(word); got != want {
			t.Fatalf("occurrences(%q) = %v, want %v", word, got, want)
		}
	}
	if got := stats.Info(); got != fixture.Info {
		t.Fatalf("info = %q, want %q", got, fixture.Info)
	}
	for i, want := range fixture.Metadata {
		got := stats.Corpora()[i]
		if got.ID != want[0] || got.Name != want[1] || got.Known != int64(want[2].(float64)) || got.Total != int64(want[3].(float64)) {
			t.Fatalf("metadata = %+v, want %v", got, want)
		}
	}
}

func TestInvalidGeneratorInputs(t *testing.T) {
	for _, line := range []string{"", "SFX A 0 a", "SFX A 0 a [", "SFX A 0 a (?=x)"} {
		if _, err := ParseAffixRule(line); err == nil {
			t.Errorf("ParseAffixRule(%q) succeeded", line)
		}
	}
	for _, input := range []string{"onlyonefield\n", "word x\n", "word 1 extra\n"} {
		if err := NewStatistics(nil).Add(strings.NewReader(input), "A", "test"); err == nil {
			t.Errorf("statistics accepted %q", input)
		}
	}
	if err := NewStatistics(nil).Add(strings.NewReader("x 1\n"), "AA", "test"); err == nil {
		t.Fatal("statistics accepted multiple-character ID")
	}
}

func TestSourceUniversalNewlines(t *testing.T) {
	for _, newline := range []string{"\n", "\r\n", "\r"} {
		d, err := ReadDictionary(strings.NewReader(strings.Join([]string{"1", "chat/S.", ""}, newline)), strings.NewReader(strings.Join([]string{"SET UTF-8", "# END", "SFX S. N 1", "SFX S. 0 s . is:pl", ""}, newline)))
		if err != nil {
			t.Fatal(err)
		}
		if len(d.Entries) != 1 || d.Entries[0].Lemma != "chat" || d.Settings != "SET UTF-8\n" || len(d.Flags) != 1 || len(d.Flags[0].Rules) != 1 {
			t.Fatalf("newline %q not normalized: entries=%+v settings=%q flags=%+v", newline, d.Entries, d.Settings, d.Flags)
		}
	}
}
