package subtitle

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestProjectSubtitlePayloadsAndPreserveSourceRanges(t *testing.T) {
	cases := []struct {
		format, input string
		want          []string
	}{
		{"srt", "\ufeff1\r\n00:00:01,000 --> 00:00:03,500 X1:10 X2:20\r\nLes <i>chat</i>\r\nsont ici.\r\n\r\n2\r\n00:00:04,000 --> 00:00:05,000\r\nbonjouur", []string{"Les chat sont ici.", "bonjouur"}},
		{"vtt", "WEBVTT - titre ignoré\nLanguage: fr\n\nNOTE textee ignoré\nsecretinconnu\n\nSTYLE\n::cue { color: red; }\n\nREGION\nid:zone\n\nidentifiant-inconnu\n00:01.000 --> 00:03.500 position:20%\n<v Personneinconnue>Les <c.rouge>chat</c> <00:02.000>sont ici.</v>\n", []string{"Les chat sont ici."}},
		{"ass", "[Script Info]\nTitle: texteinconnu\n[V4+ Styles]\nStyle: inconnue,Arial,20\n[Events]\nFormat: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\nComment: 0,0:00:01.00,0:00:03.00,Default,Inconnu,0,0,0,,texteinconnu\nDialogue: 0,0:00:01.00,0:00:03.00,Default,Inconnu,0,0,0,,{\\i1}Les chat{\\i0}\\Nsont ici.\nDialogue: 0,0:00:04.00,0:00:05.00,Default,,0,0,0,,{\\p1}m 0 0 l 100 100{\\p0}Bonjour.", []string{"Les chat sont ici.", "Bonjour."}},
		{"ssa", "[Events]\nFormat: Start, End, Text\nDialogue: 0:00:01.00,0:00:03.00,Les chat sont ici.", []string{"Les chat sont ici."}},
		{"sbv", "0:00:01.000,0:00:03.500\nLes chat\nsont ici.\n\n0:00:04.000,0:00:05.000\nBonjour.", []string{"Les chat sont ici.", "Bonjour."}},
		{"microdvd", "{1}{1}25.000\n{10}{50}{y:i}Les chat|sont ici.\n{51}{70}Bonjour.", []string{"Les chat sont ici.", "Bonjour."}},
		{"sub", "[INFORMATION]\n[TITLE]Titreinconnu\n[SUBTITLE]\n00:00:01.00,00:00:03.50\nLes chat[br]sont ici.", []string{"Les chat sont ici."}},
		{"lrc", "[ar:Inconnu]\n[ti:Chansoninconnue]\n[offset:100]\n[00:01.20][00:04.50]Les <00:01.50>chat sont ici.\n[00:06.00]Bonjour.", []string{"Les chat sont ici.", "Bonjour."}},
		{"ttml", `<?xml version="1.0"?><tt xmlns="http://www.w3.org/ns/ttml"><head><metadata>texteinconnu</metadata></head><body><div><p begin="00:00:01.000" end="00:00:03.500">Les <span style="inconnue">chat</span><br/>sont ici.</p></div></body></tt>`, []string{"Les chat sont ici."}},
	}
	for _, tc := range cases {
		t.Run(tc.format, func(t *testing.T) {
			cues, err := Project(tc.input, tc.format)
			if err != nil {
				t.Fatal(err)
			}
			got := []string{}
			for _, cue := range cues {
				got = append(got, cue.Text)
				if len(cue.Spans) != utf8.RuneCountInString(cue.Text) {
					t.Fatal("rune mapping length")
				}
				for _, span := range cue.Spans {
					if span.Start < 0 || span.End <= span.Start || span.End > len(tc.input) {
						t.Fatalf("bad source span: %+v", span)
					}
				}
				if index := strings.Index(cue.Text, "chat"); index >= 0 {
					start := utf8.RuneCountInString(cue.Text[:index])
					a, b, exact, ok := cue.SourceRange(start, start+4)
					if !ok || !exact || tc.input[a:b] != "chat" {
						t.Fatalf("unsafe source range: %d:%d %v %v", a, b, exact, ok)
					}
				}
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("payloads = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestSubtitleSyntaxAndLineBreaksAreNeverEditable(t *testing.T) {
	for _, input := range []string{
		"1\n00:00:01,000 --> 00:00:03,000\nLes <i>chat</i> sont ici.",
		"1\n00:00:01,000 --> 00:00:03,000\nLes chat\nsont ici.",
		"1\n00:00:01,000 --> 00:00:03,000\nLes chat&nbsp;sont ici.",
	} {
		cues, err := Project(input, "srt")
		if err != nil || len(cues) != 1 {
			t.Fatalf("%v %+v", err, cues)
		}
		_, _, editable, ok := cues[0].SourceRange(0, utf8.RuneCountInString(cues[0].Text))
		if !ok || editable {
			t.Fatal("markup, entity or line break editable")
		}
	}
}

func TestDetectFormatsAndIgnoreIncompleteTimecodes(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"WEBVTT\n\n", "vtt"}, {"1\n00:00:01,000 --> 00:00:03,000\nBonjour.", "srt"},
		{"[Events]\nDialogue: 0,0:00:01.00,0:00:03.00,Default,,0,0,0,,Bonjour.", "ass"},
		{"{10}{20}Bonjour.", "microdvd"}, {"[00:01.00]Bonjour.", "lrc"},
		{`<tt xmlns="http://www.w3.org/ns/ttml"><body/></tt>`, "ttml"}, {"Les chat sont ici.", ""},
	} {
		if got := Detect(tc.input); got != tc.want {
			t.Errorf("Detect(%q)=%q, want %q", tc.input, got, tc.want)
		}
	}
	cues, err := Project("1\n00:00:01,000 --> 00:00:03,\ntexteinconnu", "srt")
	if err != nil || len(cues) != 0 {
		t.Fatalf("incomplete timing analyzed as prose: %+v %v", cues, err)
	}
	if _, err := Project("Bonjour.", "unsupported"); err == nil {
		t.Fatal("unknown format accepted")
	}
	if _, err := Project(string([]byte{255}), "srt"); err == nil {
		t.Fatal("invalid UTF-8 accepted")
	}
}

func TestTTMLOnlyBodyParagraphsAndWhitespace(t *testing.T) {
	input := `<tt xmlns="http://www.w3.org/ns/ttml"><head><metadata><p>inconnu</p></metadata></head><body><div><metadata><p>inconnu</p></metadata><p>Les  chat <span> sont</span> ici.</p><p xml:space="preserve">Les  chat</p></div></body></tt>`
	cues, err := Project(input, "ttml")
	if err != nil || len(cues) != 2 || cues[0].Text != "Les chat sont ici." || cues[1].Text != "Les  chat" {
		t.Fatalf("TTML paragraphs: %+v %v", cues, err)
	}
}

func FuzzProjectSourceRanges(f *testing.F) {
	for _, source := range []string{"1\n00:00:01,000 --> 00:00:03,000\nLes <i>chat</i> sont ici.", `<tt><body><p>Les chat&amp;🐈</p></body></tt>`, "[Events]\nFormat: Text\nDialogue: {\\p1}m 0 0{\\p0}Bonjour."} {
		f.Add(source)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if !utf8.ValidString(input) {
			return
		}
		for _, format := range []string{"srt", "vtt", "ass", "sbv", "sub", "lrc", "ttml"} {
			cues, err := Project(input, format)
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range cues {
				if len(c.Spans) != utf8.RuneCountInString(c.Text) {
					t.Fatal("span count")
				}
				previous := 0
				for _, s := range c.Spans {
					if s.Start < previous || s.End <= s.Start || s.End > len(input) || !utf8.ValidString(input[s.Start:s.End]) {
						t.Fatalf("invalid source span: %+v", s)
					}
					previous = s.Start
				}
			}
		}
	})
}

func TestBOMOnFirstFrameOrLyricCue(t *testing.T) {
	for _, tc := range []struct{ format, input string }{
		{"microdvd", "\ufeff{10}{50}Les chat sont ici."},
		{"lrc", "\ufeff[00:01.00]Les chat sont ici."},
	} {
		cues, err := Project(tc.input, tc.format)
		if err != nil || len(cues) != 1 || cues[0].Text != "Les chat sont ici." {
			t.Fatalf("BOM cue: %+v %v", cues, err)
		}
	}
}
