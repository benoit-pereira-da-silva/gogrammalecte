package rules

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestPinnedFrenchSource(t *testing.T) {
	source, err := FrenchSource()
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Tests) != 14160 || len(doc.Graphs) != 10 || len(doc.RegexRules) != 181 {
		t.Fatalf("unexpected corpus: %d tests, %d graphs, %d regex rules", len(doc.Tests), len(doc.Graphs), len(doc.RegexRules))
	}
	data, err := os.ReadFile("testdata/regex_patterns.json")
	if err != nil {
		t.Fatal(err)
	}
	var expected []struct{ ID, Line, Pattern string }
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatal(err)
	}
	for i, want := range expected {
		got := doc.RegexRules[i]
		if got.ID != want.ID || got.Pattern != want.Pattern || got.LineID != want.Line {
			t.Errorf("rule %d: got %s %s %q; want %s %s %q", i, got.ID, got.LineID, got.Pattern, want.ID, want.Line, want.Pattern)
		}
	}
	if !doc.Defaults["Python"]["typo"] || doc.Defaults["Python"]["ocr"] {
		t.Fatal("incorrect Python defaults")
	}
	actions, groups := 0, 0
	for _, rule := range doc.RegexRules {
		actions += len(rule.Actions)
	}
	for _, graph := range doc.Graphs {
		groups += len(graph.Groups)
	}
	if actions != 237 || groups != 1353 {
		t.Fatalf("got %d regex actions and %d graph groups", actions, groups)
	}
}

func TestParseRejectsMalformedSource(t *testing.T) {
	for _, source := range []string{
		"    continuation without a rule", "DEF: missing", "DECL: invalid", "unknown statement",
		"__<s>(missing_action)__ foo", "@@@@END_GRAPH", "@@@@GRAPH: a | A\n@@@@GRAPH: b | B",
		"__<s>(x)__ foo <<- ->> bar && reason\n__<s>(x)__ baz <<- ->> qux && reason",
	} {
		if _, err := Parse(source); err == nil || !strings.Contains(err.Error(), "line") {
			t.Errorf("expected located error for %q, got %v", source, err)
		}
	}
}

func TestSourceMetadataAndActions(t *testing.T) {
	doc, err := Parse("DEF: word café\nOPTSOFTWARE: Python Server\nOPT/typo: True False\nOPTPRIORITY/typo: 7\n__[s]/typo(sample)__ {word} <<- before(\"x\") -1>> thé && Explanation | https://example.org\nTEST: {{café}} ->> thé\n[++]\n__<s>(end)__ fin <<- ->> fin && End")
	if err != nil {
		t.Fatal(err)
	}
	if doc.RegexRules[0].Priority != 7 || doc.RegexRules[0].Actions[0].Condition != `before("x")` || doc.RegexRules[0].Actions[0].URL != "https://example.org" || doc.RegexRules[1].Paragraph {
		t.Fatalf("unexpected parse: %+v", doc)
	}
}

func TestSuggestionCaseFlag(t *testing.T) {
	for _, mode := range []string{"i", "s", "u"} {
		doc, err := Parse("__[" + mode + "](case)__ mot <<- ->> suggestion && Message")
		if err != nil {
			t.Fatal(err)
		}
		if doc.RegexRules[0].UppercaseSuggestions != (mode == "i") {
			t.Fatalf("wrong suggestion case flag for %s", mode)
		}
	}
}

func TestPinnedOptionMetadata(t *testing.T) {
	source, err := FrenchSource()
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("testdata/options.json")
	if err != nil {
		t.Fatal(err)
	}
	var want struct {
		Colors map[string]map[string][]int
		Labels map[string]map[string]any
	}
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	got, _ := json.Marshal(struct {
		Colors map[string]map[string][]int
		Labels map[string]map[string]any
	}{doc.Colors, doc.Labels})
	expected, _ := json.Marshal(want)
	if string(got) != string(expected) {
		t.Fatal("native colors and option labels differ from Python")
	}
}

func TestMessageURLWithoutSpace(t *testing.T) {
	doc, err := Parse(`__[s](link)__ mot <<- ->> suggestion && Message|https://example.org`)
	if err != nil {
		t.Fatal(err)
	}
	action := doc.RegexRules[0].Actions[0]
	if action.Message != "Message" || action.URL != "https://example.org" {
		t.Fatalf("unexpected message: %+v", action)
	}
}
