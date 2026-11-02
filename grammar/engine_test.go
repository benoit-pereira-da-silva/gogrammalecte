package grammar

import (
	"context"
	"github.com/benoit-pereira-da-silva/gogrammalecte/internal/pyregex"
	"github.com/benoit-pereira-da-silva/gogrammalecte/rules"
	"reflect"
	"sync"
	"testing"
)

func engineForSource(t *testing.T, source string) *Engine {
	t.Helper()
	doc, err := rules.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	engine, err := New(doc, nil)
	if err != nil {
		t.Fatal(err)
	}
	return engine
}
func TestRegexEngineOffsetsCasingAndPriority(t *testing.T) {
	doc := &rules.Document{RegexRules: []rules.RegexRule{{ID: "first", LineID: "#1p", Pattern: `(?i)(fote)`, Priority: 4, Paragraph: true, UppercaseSuggestions: true, Actions: []rules.Action{{Kind: "-", Value: "faute", Group: 1, Message: `Fix \1`}}}, {ID: "higher", LineID: "#2p", Pattern: `(?i)(fote)`, Priority: 5, Paragraph: true, Actions: []rules.Action{{Kind: "-", Value: "erreur", Group: 1, Message: "Higher"}}}}, Steps: []rules.Step{{RegexIndex: 0, GraphIndex: -1, Paragraph: true}, {RegexIndex: 1, GraphIndex: -1, Paragraph: true}}}
	engine, err := New(doc, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := engine.Check(context.Background(), "😀 FOTE.", ParseOptions{Context: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatal(result)
	}
	got := result[0]
	if got.Start != 2 || got.End != 6 || got.RuleID != "higher" || got.Underlined != "FOTE" || got.Before != "😀 " || got.After != "." {
		t.Fatalf("unexpected diagnostic %#v", got)
	}
	doc.RegexRules = doc.RegexRules[:1]
	doc.Steps = doc.Steps[:1]
	engine, err = New(doc, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err = engine.Check(context.Background(), "FOTE.", ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result[0].Suggestions[0] != "FAUTE" || result[0].Message != "Fix FOTE" {
		t.Fatal(result)
	}
}
func TestGraphEngineRewritingTagsAndImmunity(t *testing.T) {
	source := `[++]
@@@@GRAPH: rewrite | R
__rewrite__
    chat
        <<- ~>> chien

@@@@END_GRAPH
@@@@GRAPH: diagnose | D
__diagnose__
    chien
        <<- ->> animal && Changed token

@@@@END_GRAPH
`
	engine := engineForSource(t, source)
	result, err := engine.Check(context.Background(), "Le chat.", ParseOptions{Context: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].Start != 3 || result[0].End != 7 || result[0].Underlined != "chat" {
		t.Fatal(result)
	}
	source = `[++]
@@@@GRAPH: immune | I
__tag__
    chat
        <<- />> noun

__diagnose__
    /noun
        <<- ->> animal && Tagged token
        <<- !>>

@@@@END_GRAPH
`
	engine = engineForSource(t, source)
	result, err = engine.Check(context.Background(), "Le chat.", ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatal(result)
	}
}
func TestEngineConditionBreakAndCancellation(t *testing.T) {
	engine := engineForSource(t, `[++]
@@@@GRAPH: conditions | C
__break__
    chat
        <<- False >>>
        <<- ->> chien && Unexpected

@@@@END_GRAPH
`)
	got, err := engine.Check(context.Background(), "Le chat.", ParseOptions{})
	if err != nil || len(got) != 0 {
		t.Fatalf("%#v %v", got, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = engine.Check(ctx, "Le chat.", ParseOptions{}); err != context.Canceled {
		t.Fatal(err)
	}
}

func TestEmptyLiteralSuggestionsAndNullableRewrite(t *testing.T) {
	doc := &rules.Document{RegexRules: []rules.RegexRule{{ID: "empty", Pattern: `(x)`, Paragraph: true, Actions: []rules.Action{{Kind: "-", Value: "", Message: "Delete", Group: 1}}}}, Steps: []rules.Step{{RegexIndex: 0, GraphIndex: -1, Paragraph: true}}}
	engine, err := New(doc, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := engine.Check(context.Background(), "x", ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(got[0].Suggestions) != 1 || got[0].Suggestions[0] != "" {
		t.Fatalf("empty literal suggestion lost: %#v", got)
	}
	engine = engineForSource(t, `[++]
@@@@GRAPH: optional | O
__optional__
    chat ?noir¿
        <<- ~2:0>> *

@@@@END_GRAPH
`)
	if _, err = engine.Check(context.Background(), "Un chat", ParseOptions{}); err != nil {
		t.Fatal(err)
	}
}

func TestEngineCopiesDocument(t *testing.T) {
	document := &rules.Document{RegexRules: []rules.RegexRule{{ID: "original", Pattern: `x`, Paragraph: true, Actions: []rules.Action{{Kind: "-", Value: "y", Message: "message"}}}}, Steps: []rules.Step{{RegexIndex: 0, GraphIndex: -1, Paragraph: true}}}
	engine, err := New(document, nil)
	if err != nil {
		t.Fatal(err)
	}
	document.RegexRules[0].Actions[0].Message = "changed"
	document.Steps = nil
	got, err := engine.Check(context.Background(), "x", ParseOptions{})
	if err != nil || len(got) != 1 || got[0].Message != "message" {
		t.Fatalf("source mutation leaked into engine: %#v %v", got, err)
	}
}

func TestRegexReplacementExpansionPythonParity(t *testing.T) {
	pattern, err := pyregex.Compile(`(?P<word>.*)`)
	if err != nil {
		t.Fatal(err)
	}
	match, err := pattern.Search(context.Background(), `\\n\\1`)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ source, want string }{{`\1`, `\\n\\1`}, {`\\1`, `\1`}, {`\g<0>`, `\\n\\1`}, {`\g<word>`, `\\n\\1`}, {`\0`, "\x00"}, {`\111`, "I"}, {`\n`, "\n"}, {`\t`, "\t"}, {`\v`, "\v"}, {`\&`, `\&`}}
	for _, tc := range cases {
		got, err := expandMatch(tc.source, *match)
		if err != nil || got != tc.want {
			t.Errorf("expand %q got %q, want %q: %v", tc.source, got, tc.want, err)
		}
	}
	for _, source := range []string{`\2`, `\x`, `\g<`, `\g<missing>`, `\`} {
		if _, err := expandMatch(source, *match); err == nil {
			t.Errorf("accepted invalid replacement %q", source)
		}
	}
}

func TestConcurrentEngineStateIsolation(t *testing.T) {
	engine, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	inputs := []string{"Les chat mange.", "Le train de 2 h 47 arriveraient en retard.", "Il pleut comme même…", "Et peut de gens peuvent en dire autant."}
	expected := make([][]Error, len(inputs))
	for i, input := range inputs {
		expected[i], err = engine.Check(context.Background(), input, ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
	}
	var group sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for i, input := range inputs {
				got, err := engine.Check(context.Background(), input, ParseOptions{})
				if err != nil || !reflect.DeepEqual(got, expected[i]) {
					t.Errorf("concurrent parse %q: %#v %v", input, got, err)
				}
			}
		}()
	}
	group.Wait()
}
