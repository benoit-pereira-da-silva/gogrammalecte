package graphspell

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/dictionary"
	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/tokenizer"
)

func TestCheckerPythonParity(t *testing.T) {
	raw, err := os.ReadFile("testdata/spellchecker.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Paragraphs []struct {
			Input                   string
			Errors                  []tokenizer.Token
			Counts, Lemmas, Unknown map[string]int
		}
		Suggestions []struct {
			Word   string
			Groups [][]string
		}
	}
	if err = json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	c, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range f.Paragraphs {
		if got := c.ParseParagraph(tc.Input); !reflect.DeepEqual(got, tc.Errors) {
			t.Errorf("errors %q got %#v want %#v", tc.Input, got, tc.Errors)
		}
		for _, mode := range []struct {
			lemma, unknown bool
			want           map[string]int
		}{{false, false, tc.Counts}, {true, false, tc.Lemmas}, {false, true, tc.Unknown}} {
			if got := c.CountWords(tc.Input, mode.lemma, mode.unknown); !reflect.DeepEqual(got, mode.want) {
				t.Errorf("count %q %v/%v got %v want %v", tc.Input, mode.lemma, mode.unknown, got, mode.want)
			}
		}
	}
	for _, tc := range f.Suggestions {
		got, err := c.Suggest(context.Background(), tc.Word, 5)
		if err != nil || !reflect.DeepEqual(got, tc.Groups) {
			t.Errorf("suggest %q got %v want %v err %v", tc.Word, got, tc.Groups, err)
		}
	}
}

func TestSupplementaryDictionaries(t *testing.T) {
	main, err := dictionary.Build([]dictionary.Entry{{Word: "chat", Stem: "chat", Tag: ":N:m:s"}}, dictionary.BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	extra, err := dictionary.Build([]dictionary.Entry{{Word: "codexmot", Stem: "codexmot", Tag: ":N:m:s"}}, dictionary.BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	c, err := New(main, extra, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Lookup("codexmot") || !c.IsValid("Codexmot") || !c.IsValidToken("codexmot-codexmot") || len(c.Morph("Codexmot")) != 1 {
		t.Fatal("supplementary dictionary missing")
	}
	if c.IsValidToken("codexmot-chat") {
		t.Fatal("compound parts must belong to the same dictionary, like Python")
	}
	if _, err := New(nil, nil, nil); err == nil {
		t.Fatal("nil main dictionary accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = c.Suggest(ctx, "test", 3); err == nil {
		t.Fatal("cancellation ignored")
	}
}
