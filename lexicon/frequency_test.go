package lexicon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestFullCorpusFrequencyReference(t *testing.T) {
	var fixture struct {
		Recognized                                                                           int64
		FlexionSHA256, EntrySHA256, SortedGrammarSHA256, ConjugationSHA256, DeclensionSHA256 string
		FullLexiconSHA256                                                                    string
		ReportSHA256, HunspellMode2DictionarySHA256, HunspellMode2AffixSHA256                string
		Samples                                                                              []struct {
			Word, Lemma    string
			Occur          int64
			Frequency      float64
			FrequencyIndex string
		}
	}
	data, err := os.ReadFile("testdata/frequencies.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	d, err := EmbeddedDictionary()
	if err != nil {
		t.Fatal(err)
	}
	d.SortNatural()
	if err := d.Generate(); err != nil {
		t.Fatal(err)
	}
	stats, err := d.EmbeddedStatistics()
	if err != nil {
		t.Fatal(err)
	}
	report := sha256.New()
	if err := d.CalculateFrequenciesReport(stats, report); err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(report.Sum(nil)); got != fixture.ReportSHA256 {
		t.Fatalf("statistics report SHA-256 = %s, want %s", got, fixture.ReportSHA256)
	}
	if d.RecognizedOccurrences != fixture.Recognized {
		t.Fatalf("recognized occurrences = %d, want %d", d.RecognizedOccurrences, fixture.Recognized)
	}
	flexions, entries := sha256.New(), sha256.New()
	for _, f := range d.Flexions {
		blocked := "False"
		if f.Blocked {
			blocked = "True"
		}
		fmt.Fprintf(flexions, "%s\t%d\t%s\t%s\n", f.Text, f.Occurrences, blocked, f.FrequencyIndex)
	}
	for _, e := range d.Entries {
		fmt.Fprintf(entries, "%s\t%s\t%d\t%d\t%s\t%s\n", e.Lemma, e.Flags, e.Occurrences, e.AverageKnownOccurrences, e.Fields["fq"], e.OldFrequencyIndex)
	}
	if got := hex.EncodeToString(flexions.Sum(nil)); got != fixture.FlexionSHA256 {
		t.Fatalf("frequency SHA-256 = %s, want %s", got, fixture.FlexionSHA256)
	}
	if got := hex.EncodeToString(entries.Sum(nil)); got != fixture.EntrySHA256 {
		t.Fatalf("entry frequency SHA-256 = %s, want %s", got, fixture.EntrySHA256)
	}
	d.SortByFrequency()
	grammar := sha256.New()
	for _, f := range d.Flexions {
		fmt.Fprint(grammar, f.GrammarCheckerLine())
	}
	if got := hex.EncodeToString(grammar.Sum(nil)); got != fixture.SortedGrammarSHA256 {
		t.Fatalf("sorted lexicon SHA-256 = %s, want %s", got, fixture.SortedGrammarSHA256)
	}
	for i, want := range fixture.Samples {
		got := d.Flexions[i]
		if got.Text != want.Word || got.Entry.Lemma != want.Lemma || got.Occurrences != want.Occur || got.Frequency != want.Frequency || got.FrequencyIndex != want.FrequencyIndex {
			t.Fatalf("rank %d = %+v, want %+v", i, got, want)
		}
	}
	conj, decl := sha256.New(), sha256.New()
	if err := d.WriteConjugations(conj); err != nil {
		t.Fatal(err)
	}
	if err := d.WriteDeclensions(decl); err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(conj.Sum(nil)); got != fixture.ConjugationSHA256 {
		t.Fatalf("conjugation SHA-256 = %s, want %s", got, fixture.ConjugationSHA256)
	}
	if got := hex.EncodeToString(decl.Sum(nil)); got != fixture.DeclensionSHA256 {
		t.Fatalf("declension SHA-256 = %s, want %s", got, fixture.DeclensionSHA256)
	}
	dic, aff := sha256.New(), sha256.New()
	if err := d.WriteHunspell(dic, aff, "fr-toutesvariantes", "test", HunspellOptions{Mode: 2, Aliases: true}); err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(dic.Sum(nil)); got != fixture.HunspellMode2DictionarySHA256 {
		t.Fatalf("post-frequency Hunspell dictionary SHA-256 = %s, want %s", got, fixture.HunspellMode2DictionarySHA256)
	}
	if got := hex.EncodeToString(aff.Sum(nil)); got != fixture.HunspellMode2AffixSHA256 {
		t.Fatalf("post-frequency Hunspell affix SHA-256 = %s, want %s", got, fixture.HunspellMode2AffixSHA256)
	}
	d.CalculatePhonetics()
	full := sha256.New()
	if err := d.WriteLexicon(full, "test", stats); err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(full.Sum(nil)); got != fixture.FullLexiconSHA256 {
		t.Fatalf("full lexicon SHA-256 = %s, want %s", got, fixture.FullLexiconSHA256)
	}
}

func TestFrequencyIndexBoundaries(t *testing.T) {
	for _, c := range []struct {
		frequency float64
		want      string
	}{{0, "0"}, {1e-9, "1"}, {1e-8, "2"}, {1e-7, "3"}, {1e-6, "4"}, {1e-5, "5"}, {1e-4, "6"}, {1e-3, "7"}, {1e-2, "8"}, {0.1, "9"}, {100, "9"}, {-1, "1"}} {
		if got := FrequencyIndex(c.frequency); got != c.want {
			t.Errorf("FrequencyIndex(%g) = %s, want %s", c.frequency, got, c.want)
		}
	}
}
