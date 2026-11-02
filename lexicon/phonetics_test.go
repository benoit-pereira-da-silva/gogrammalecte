package lexicon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestPhoneticsReference(t *testing.T) {
	var fixture struct {
		Cases []struct {
			Word, Morph           string
			Metagraphe, Metaphone [2]string
		}
		FullSHA256 string
		FullCount  int
	}
	data, err := os.ReadFile("testdata/phonetics.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, c := range fixture.Cases {
		if got := Metagraphe(c.Word, c.Morph); got != c.Metagraphe {
			t.Fatalf("Metagraphe(%q,%q) = %q, want %q", c.Word, c.Morph, got, c.Metagraphe)
		}
		if got := DoubleMetaphone(c.Word); got != c.Metaphone {
			t.Fatalf("DoubleMetaphone(%q) = %q, want %q", c.Word, got, c.Metaphone)
		}
	}
	d, err := EmbeddedDictionary()
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Generate(); err != nil {
		t.Fatal(err)
	}
	hash := sha256.New()
	for _, f := range d.Flexions {
		m, p := Metagraphe(f.Text, f.Morphology), DoubleMetaphone(f.Text)
		fmt.Fprintf(hash, "%s\t%s\t%s\t%s\n", m[0], m[1], p[0], p[1])
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != fixture.FullSHA256 {
		t.Fatalf("phonetic SHA-256 = %s, want %s", got, fixture.FullSHA256)
	}
}
