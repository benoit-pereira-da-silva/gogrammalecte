package text

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestRenderingPythonParity(t *testing.T) {
	data, err := os.ReadFile("testdata/rendering.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Wrap []struct {
			Input  string
			Width  int
			Output []string
		}
		Flow []struct {
			Input  string
			Width  int
			Indent string
			Output []string
		}
		Readable []struct {
			Error    Diagnostic
			Spelling bool
			Output   string
		}
		Readables []struct {
			Errors   []Diagnostic
			Width    int
			Spelling bool
			Output   string
		}
		Paragraphs []struct {
			Input             string
			Grammar, Spelling []Diagnostic
			Width             int
			Output            string
			Merged            []Diagnostic
		}
		Lines []struct {
			Lines     []json.RawMessage
			Paragraph string
			Spans     [][3]int
		}
		Coordinates []struct {
			Grammar, Spelling []Diagnostic
			Spans             [][3]int
			ResultGrammar     []Diagnostic `json:"result_grammar"`
			ResultSpelling    []Diagnostic `json:"result_spelling"`
		}
	}
	if err = json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	for _, c := range f.Wrap {
		got, err := Wrap(c.Input, c.Width)
		if err != nil || !reflect.DeepEqual(got, c.Output) {
			t.Errorf("wrap %q width=%d got %q (%v), want %q", c.Input, c.Width, got, err, c.Output)
		}
	}
	for _, c := range f.Flow {
		got, err := wrapMessage(c.Input, c.Width, c.Indent)
		if err != nil || !reflect.DeepEqual(got, c.Output) {
			t.Errorf("flow %q width=%d indent=%q got %q (%v), want %q", c.Input, c.Width, c.Indent, got, err, c.Output)
		}
	}
	for _, c := range f.Readable {
		got, err := ReadableError(c.Error, c.Spelling)
		if err != nil || got != c.Output {
			t.Errorf("readable got %q (%v), want %q", got, err, c.Output)
		}
	}
	for _, c := range f.Readables {
		got, err := ReadableErrors(c.Errors, c.Width, c.Spelling)
		if err != nil || got != c.Output {
			t.Errorf("readables width=%d got %q (%v), want %q", c.Width, got, err, c.Output)
		}
	}
	for _, c := range f.Paragraphs {
		got, merged, err := GenerateParagraph(c.Input, c.Grammar, c.Spelling, c.Width)
		a, _ := json.Marshal(merged)
		b, _ := json.Marshal(c.Merged)
		if err != nil || got != c.Output || string(a) != string(b) {
			t.Errorf("paragraph %q width=%d got %q (%v), want %q; merged=%s want=%s", c.Input, c.Width, got, err, c.Output, a, b)
		}
	}
	for _, c := range f.Lines {
		var lines []NumberedLine
		for _, raw := range c.Lines {
			var tuple []json.RawMessage
			if err = json.Unmarshal(raw, &tuple); err != nil {
				t.Fatal(err)
			}
			var line NumberedLine
			if err = json.Unmarshal(tuple[0], &line.Number); err != nil {
				t.Fatal(err)
			}
			if err = json.Unmarshal(tuple[1], &line.Text); err != nil {
				t.Fatal(err)
			}
			lines = append(lines, line)
		}
		got, spans := CreateParagraphWithLines(lines)
		if got != c.Paragraph || !reflect.DeepEqual(spans, lineSpans(c.Spans)) {
			t.Errorf("line merge got %q %#v want %q %#v", got, spans, c.Paragraph, c.Spans)
		}
	}
	for _, c := range f.Coordinates {
		if err := ConvertToXY(c.Grammar, c.Spelling, lineSpans(c.Spans)); err != nil {
			t.Fatal(err)
		}
		got, _ := json.Marshal([]any{c.Grammar, c.Spelling})
		want, _ := json.Marshal([]any{c.ResultGrammar, c.ResultSpelling})
		if string(got) != string(want) {
			t.Errorf("coordinates got %s want %s", got, want)
		}
	}
}
func lineSpans(values [][3]int) []LineSpan {
	out := make([]LineSpan, len(values))
	for i, v := range values {
		out[i] = LineSpan{v[0], v[1], v[2]}
	}
	return out
}
func TestRenderingRejectsInvalidInputs(t *testing.T) {
	if _, err := Wrap("abc", 0); err == nil {
		t.Fatal("expected invalid width error")
	}
	if _, err := ReadableError(Diagnostic{"nStart": 0}, false); err == nil {
		t.Fatal("expected incomplete diagnostic error")
	}
	if _, _, err := GenerateParagraph("a", []Diagnostic{{"nStart": "invalid", "nEnd": 1}}, nil, 80); err == nil {
		t.Fatal("expected invalid diagnostic offset error")
	}
}
