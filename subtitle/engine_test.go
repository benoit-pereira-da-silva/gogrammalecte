package subtitle

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/benoit-pereira-da-silva/gogrammalecte/grammar"
)

func TestCheckSubtitleKeepsGrammarOffsetsAndHidesTimingAndMarkup(t *testing.T) {
	const input = "\ufeff1\r\n00:00:01,000 --> 00:00:03,000\r\nLes <i>chat</i> sont ici. 😀\r\n\r\n2\r\n00:00:04,000 --> 00:00:06,000\r\nBonjour."
	errors, err := Check(context.Background(), input, "srt", grammar.ParseOptions{Context: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(errors) != 1 {
		t.Fatalf("errors: %+v", errors)
	}
	e := errors[0]
	want := utf8.RuneCountInString(input[:strings.Index(input, "chat")])
	if e.Start != want || e.End != want+4 || e.Underlined != "chat" || len(e.Suggestions) == 0 || e.Suggestions[0] != "chats" {
		t.Fatalf("wrong source diagnostic: %+v", e)
	}
	full, err := Analyze(context.Background(), input, "srt", grammar.ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range full.Sentences {
		for _, tkn := range s.Tokens {
			if tkn.Value == "chat" {
				found = true
				if s.Start+tkn.Start != want {
					t.Fatalf("wrong token offsets: %+v", tkn)
				}
			}
		}
	}
	if !found {
		t.Fatal("full token analysis absent")
	}
}

func TestSubtitleSpellingAndFormattingOnlyTouchPayload(t *testing.T) {
	const input = "1\n00:00:01,000 --> 00:00:03,000\n<i>bonjouur</i>\n\n2\n00:00:04,000 --> 00:00:06,000\nC'est beaucoup..."
	errors, err := Spell(context.Background(), input, "srt")
	if err != nil || len(errors) != 1 || errors[0].Value != "bonjouur" {
		t.Fatalf("spelling errors: %+v %v", errors, err)
	}
	out, err := Format(context.Background(), input, "srt", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "00:00:01,000 --> 00:00:03,000") || !strings.Contains(out, "<i>bonjouur</i>") || !strings.Contains(out, "C’est beaucoup…") {
		t.Fatalf("structure changed or prose not formatted: %q", out)
	}
}
