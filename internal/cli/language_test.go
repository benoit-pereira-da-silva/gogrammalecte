package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestEmbeddedLookup(t *testing.T) {
	var out bytes.Buffer
	if err := execute([]string{"lookup", "bonjour"}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Word         string
		Valid        bool
		Morphologies []string
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Valid || got.Word != "bonjour" || len(got.Morphologies) != 1 || got.Morphologies[0] != ">bonjour/:N:m:s/*" {
		t.Fatalf("unexpected lookup: %+v", got)
	}
}

func TestEmbeddedConjugation(t *testing.T) {
	var out bytes.Buffer
	if err := execute([]string{"conjugate", "être", ":Ip", ":1s"}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	var got string
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || got != "suis" {
		t.Fatalf("got %q err %v", got, err)
	}
	out.Reset()
	if err := execute([]string{"conjugate", "inexistantverbe", ":Ip", ":1s"}, strings.NewReader(""), &out); err == nil || out.Len() != 0 {
		t.Fatalf("unknown verb accepted: %v %s", err, out.String())
	}
}

func TestBundledRuleInventory(t *testing.T) {
	var out bytes.Buffer
	if err := execute([]string{"rules"}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	var got struct{ Tests, RegexRules, Graphs int }
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Tests != 14160 || got.RegexRules != 181 || got.Graphs != 10 {
		t.Fatalf("unexpected rules inventory %+v", got)
	}
}

func TestSpellingCommands(t *testing.T) {
	var out bytes.Buffer
	if err := execute([]string{"spell"}, strings.NewReader("😀 Les chevaus."), &out); err != nil {
		t.Fatal(err)
	}
	var errors []struct {
		Value string `json:"sValue"`
		Start int    `json:"nStart"`
		End   int    `json:"nEnd"`
	}
	if err := json.Unmarshal(out.Bytes(), &errors); err != nil || len(errors) != 1 || errors[0].Value != "chevaus" || errors[0].Start != 6 || errors[0].End != 13 {
		t.Fatalf("unexpected errors %s: %v", out.String(), err)
	}
	out.Reset()
	if err := execute([]string{"suggest", "bcp"}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "[[\"beaucoup\"]]\n" {
		t.Fatal(out.String())
	}
	out.Reset()
	if err := execute([]string{"spell"}, strings.NewReader(string([]byte{0xff})), &out); err == nil {
		t.Fatal("invalid UTF-8 accepted")
	}
}

func TestGrammarAndFormatterCommands(t *testing.T) {
	var out bytes.Buffer
	if err := execute([]string{"check"}, strings.NewReader("Il mues."), &out); err != nil {
		t.Fatal(err)
	}
	var errors []struct {
		Start       int      `json:"nStart"`
		End         int      `json:"nEnd"`
		Rule        string   `json:"sRuleId"`
		Suggestions []string `json:"aSuggestions"`
	}
	if err := json.Unmarshal(out.Bytes(), &errors); err != nil || len(errors) != 1 || errors[0].Start != 3 || errors[0].End != 7 {
		t.Fatalf("unexpected grammar response %s err %v", out.String(), err)
	}
	out.Reset()
	if err := execute([]string{"format"}, strings.NewReader("Bonjour  monde."), &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "Bonjour monde." {
		t.Fatal(out.String())
	}
	out.Reset()
	if err := execute([]string{"generate", "inflections"}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	if !json.Valid(out.Bytes()) || out.Len() < 10000 {
		t.Fatal("missing native inflection resource")
	}
}

func TestFullGrammarAnalysis(t *testing.T) {
	var out bytes.Buffer
	if err := execute([]string{"check", "--full", "--include-context"}, strings.NewReader("Les chat sont ici."), &out); err != nil {
		t.Fatal(err)
	}
	var analysis struct {
		Sentences []struct {
			Tokens []json.RawMessage `json:"lTokens"`
		} `json:"lSentences"`
	}
	if err := json.Unmarshal(out.Bytes(), &analysis); err != nil || len(analysis.Sentences) != 1 || len(analysis.Sentences[0].Tokens) == 0 {
		t.Fatalf("missing full analysis: %s err %v", out.String(), err)
	}
}

func TestLexicalValidationCommand(t *testing.T) {
	var out bytes.Buffer
	if err := execute([]string{"generate", "validation"}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	var warnings []string
	if err := json.Unmarshal(out.Bytes(), &warnings); err != nil {
		t.Fatal(err)
	}
	if err := execute([]string{"generate", "unknown-target"}, strings.NewReader(""), &out); err == nil {
		t.Fatal("unknown generation target accepted")
	}
	if err := execute([]string{"generate", "inflections"}, strings.NewReader(""), brokenIO{}); err == nil {
		t.Fatal("output error ignored")
	}
}
