package grammar

import (
	"context"
	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell"
	"testing"
)

func TestExpressionInvalidArgumentsReturnErrors(t *testing.T) {
	checker, err := graphspell.Default()
	if err != nil {
		t.Fatal(err)
	}
	ctx := &EvalContext{Context: context.Background(), Helpers: NewHelperContext(checker), Text: "chat"}
	for _, source := range []string{`space()`, `space("chat",1)`, `morph2()`, `word(1)`, `before("chat")`} {
		t.Run(source, func(t *testing.T) {
			defer func() {
				if value := recover(); value != nil {
					t.Errorf("invalid expression panicked: %v", value)
				}
			}()
			if _, err := Evaluate(source, ctx); err == nil {
				t.Fatal("invalid expression was accepted")
			}
		})
	}
}
func TestTextareaExpressionNeedsNoTokenSpan(t *testing.T) {
	checker, err := graphspell.Default()
	if err != nil {
		t.Fatal(err)
	}
	ctx := &EvalContext{Context: context.Background(), Helpers: NewHelperContext(checker), Text: "Le chat", OriginalText: "Le chien"}
	for _, source := range []string{`textarea("chat")`, `textarea0("chien")`} {
		t.Run(source, func(t *testing.T) {
			defer func() {
				if value := recover(); value != nil {
					t.Errorf("text-only expression panicked: %v", value)
				}
			}()
			got, err := Evaluate(source, ctx)
			if err != nil || got != true {
				t.Fatalf("got %#v, error %v", got, err)
			}
		})
	}
}
