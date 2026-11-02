package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/benoit-pereira-da-silva/gogrammalecte/grammar"
)

func TestCheckParagraphsPreservesOriginalOffsets(t *testing.T) {
	for _, newline := range []string{"\n", "\r\n", "\r"} {
		t.Run(newline, func(t *testing.T) {
			prefix := "😀 Bonjour." + newline
			input := prefix + "  Il mues."
			var output bytes.Buffer
			if err := execute([]string{"check", "--enable", "esp", "--include-context"}, strings.NewReader(input), &output); err != nil {
				t.Fatal(err)
			}
			var diagnostics []grammar.Error
			if err := json.Unmarshal(output.Bytes(), &diagnostics); err != nil {
				t.Fatal(err)
			}
			foundSpace, foundVerb := false, false
			for _, diagnostic := range diagnostics {
				switch diagnostic.RuleID {
				case "esp_début_ligne":
					foundSpace = true
					if diagnostic.Start != utf8.RuneCountInString(prefix) || diagnostic.End != utf8.RuneCountInString(prefix)+2 || diagnostic.Before != prefix || diagnostic.Underlined != "  " {
						t.Fatalf("wrong paragraph/context offsets: %#v", diagnostic)
					}
				case "g2__conj_il__b1_a1_1":
					foundVerb = true
					if diagnostic.Start != utf8.RuneCountInString(prefix)+5 || diagnostic.End != utf8.RuneCountInString(prefix)+9 {
						t.Fatalf("wrong Unicode offset: %#v", diagnostic)
					}
				}
			}
			if !foundSpace || !foundVerb {
				t.Fatalf("paragraph diagnostics missing: %s", output.String())
			}
		})
	}
}
func TestCheckLongInputChecksEveryParagraph(t *testing.T) {
	prefix := strings.Repeat("Les chats dorment.\n", 180)
	var output bytes.Buffer
	if err := execute([]string{"check"}, strings.NewReader(prefix+"Il mues."), &output); err != nil {
		t.Fatal(err)
	}
	var diagnostics []grammar.Error
	if err := json.Unmarshal(output.Bytes(), &diagnostics); err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 1 || diagnostics[0].Start != utf8.RuneCountInString(prefix)+3 {
		t.Fatalf("last paragraph not checked: %s", output.String())
	}
}
func TestFullCheckParagraphAndTokenOffsets(t *testing.T) {
	prefix := "Bonjour.\r\n"
	var output bytes.Buffer
	if err := execute([]string{"check", "--full"}, strings.NewReader(prefix+"Il mues."), &output); err != nil {
		t.Fatal(err)
	}
	var analysis grammar.Analysis
	if err := json.Unmarshal(output.Bytes(), &analysis); err != nil {
		t.Fatal(err)
	}
	if len(analysis.Sentences) != 2 {
		t.Fatalf("got %d sentences", len(analysis.Sentences))
	}
	sentence := analysis.Sentences[1]
	if sentence.Start != utf8.RuneCountInString(prefix) || sentence.End != utf8.RuneCountInString(prefix)+8 || sentence.Tokens[1].Start != 0 || sentence.Errors[0].Start != utf8.RuneCountInString(prefix)+3 {
		t.Fatalf("wrong document/sentence offsets: %#v", sentence)
	}
}
