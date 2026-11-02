package grammar

import (
	"context"
	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell"
	"github.com/benoit-pereira-da-silva/gogrammalecte/internal/pyregex"
	"testing"
)

func TestExpressionBindings(t *testing.T) {
	checker, err := graphspell.Default()
	if err != nil {
		t.Fatal(err)
	}
	pattern, err := pyregex.Compile(`(Les) (chats)`)
	if err != nil {
		t.Fatal(err)
	}
	match, err := pattern.Search(context.Background(), "Les chats.")
	if err != nil {
		t.Fatal(err)
	}
	h := NewHelperContext(checker)
	ctx := &EvalContext{Context: context.Background(), Helpers: h, Text: "Les chats.", OriginalText: "Les chats.", Match: match, Country: "FR", AppContext: "Python", Options: map[string]bool{"typo": true}}
	for expression, want := range map[string]any{`morph(\2, ":N.*:p")`: true, `\1.lower()+" "+suggSing(\2)`: "les chat", `after("^[.]")`: true, `option("typo")`: true, `checkDate("31","2","2024")`: false, `re.search("^ch",\2)`: true, `/0`: "Les chats"} {
		got, err := Evaluate(expression, ctx)
		if err != nil || got != want {
			t.Errorf("%s got %#v want %#v err %v", expression, got, want, err)
		}
	}
	if _, err := Evaluate(`morph()`, ctx); err == nil {
		t.Fatal("invalid arguments accepted")
	}
}

func TestGraphNeighborAfterTokenRemoval(t *testing.T) {
	h := NewHelperContext(fixtureLexicon{"chat": {">chat/:N:m:s"}})
	tokens := []*Token{{Index: 0, Type: "INFO"}, {Index: 4, Type: "WORD", Value: "chat", Start: 0, End: 4}, {Index: 7, Type: "WORD", Value: "noir", Start: 5, End: 9}, {Index: 5, Type: "INFO", Start: 9, End: 9}}
	ctx := &EvalContext{Helpers: h, Tokens: tokens, TokenOffset: 0, LastToken: 1}
	value, err := Evaluate(`space(\1,1,1)`, ctx)
	if err != nil || value != true {
		t.Fatalf("current slice neighbor lost: %v %v", value, err)
	}
}
