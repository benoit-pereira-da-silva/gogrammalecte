package dictionary

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestBundledDictionary(t *testing.T) {
	d, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	if d.Stats().Entries != 515637 {
		t.Fatalf("unexpected entry count: %#v", d.Stats())
	}
	for _, word := range []string{"bonjour", "grammaire", "français", "orthographe", "œil", "chevaux"} {
		if !d.Lookup(word) {
			t.Errorf("missing %q", word)
		}
	}
	if d.Lookup("azertyinexistantxyz") {
		t.Fatal("unexpected word")
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			other, e := Default()
			if e != nil || other != d {
				t.Errorf("default dictionary was not shared: %v", e)
			}
		})
	}
	wg.Wait()
}
func TestBundledLexicon(t *testing.T) {
	r, err := BundledLexicon()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	h := sha256.New()
	var content bytes.Buffer
	if _, err = io.Copy(io.MultiWriter(h, &content), r); err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != BundledLexiconSHA256 {
		t.Fatalf("lexicon checksum %s", got)
	}
	entries, _, err := ReadLexicon(&content)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 515693 {
		t.Fatalf("unexpected lexical rows %d", len(entries))
	}
	d, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !d.Lookup(entry.Word) {
			t.Fatalf("missing word %q", entry.Word)
		}
		want := ">" + entry.Stem + "/" + entry.Tag
		found := false
		for _, m := range d.MorphExact(entry.Word) {
			if m == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing morphology %q for %q", want, entry.Word)
		}
	}
}

func TestBundledRegeneration(t *testing.T) {
	want, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	got, err := BuildBundled(AllVariants)
	if err != nil {
		t.Fatal(err)
	}
	var a, b bytes.Buffer
	if err = want.Write(&a); err != nil {
		t.Fatal(err)
	}
	if err = got.Write(&b); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("native regeneration differs from bundled graph")
	}
}
func TestBundledVariants(t *testing.T) {
	for _, v := range []Variant{Classic, Reform1990} {
		d, e := Bundled(v)
		if e != nil {
			t.Fatal(e)
		}
		if d.Stats().Entries >= 515637 || !d.Lookup("bonjour") {
			t.Fatalf("invalid %s variant stats %#v", v, d.Stats())
		}
		for _, entry := range d.Entries() {
			last := entry.Tag[len(entry.Tag)-1:]
			allowed := "*CMPX"
			if v == Reform1990 {
				allowed = "*RPX"
			}
			if !strings.Contains(allowed, last) {
				t.Fatalf("unexpected variant tag %q", entry.Tag)
			}
		}
	}
	if _, err := Bundled("invalid"); err == nil {
		t.Fatal("invalid variant accepted")
	}
}
