package lexicon

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

func TestAllHunspellExportsReference(t *testing.T) {
	var fixture []struct {
		Variant                       string
		Mode                          int
		Aliases, Simplified           bool
		DictionarySHA256, AffixSHA256 string
	}
	data, err := os.ReadFile("testdata/hunspell.json")
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
	for _, c := range fixture {
		dic, aff := sha256.New(), sha256.New()
		if err := d.WriteHunspell(dic, aff, c.Variant, "test", HunspellOptions{Mode: c.Mode, Aliases: c.Aliases, Simplified: c.Simplified}); err != nil {
			t.Fatal(err)
		}
		if got := hex.EncodeToString(dic.Sum(nil)); got != c.DictionarySHA256 {
			t.Fatalf("%s mode %d aliases %t simplified %t dictionary hash = %s, want %s", c.Variant, c.Mode, c.Aliases, c.Simplified, got, c.DictionarySHA256)
		}
		if got := hex.EncodeToString(aff.Sum(nil)); got != c.AffixSHA256 {
			t.Fatalf("%s mode %d aliases %t simplified %t affix hash = %s, want %s", c.Variant, c.Mode, c.Aliases, c.Simplified, got, c.AffixSHA256)
		}
	}
}

func TestAliasReportReference(t *testing.T) {
	var fixture struct {
		Lines    []string
		Expected string
	}
	data, err := os.ReadFile("testdata/aliases.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	d := &Dictionary{}
	for _, line := range fixture.Lines {
		d.Entries = append(d.Entries, ParseEntry(line))
	}
	var out bytes.Buffer
	if err := d.WriteAliasReport(&out, 2); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != fixture.Expected {
		t.Fatalf("alias report = %q, want %q", got, fixture.Expected)
	}
}
