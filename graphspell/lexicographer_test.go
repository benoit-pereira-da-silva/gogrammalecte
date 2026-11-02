package graphspell

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestLexicographerPythonParity(t *testing.T) {
	data, err := os.ReadFile("testdata/lexicographer.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Readable []struct{ Input, Output string }
		Words    []struct {
			Input    string
			Split    [3]string
			Meaning  string
			Analysis []json.RawMessage
		}
		Tokens []struct {
			Input struct {
				Type       string   `json:"sType"`
				Value      string   `json:"sValue"`
				Morphology []string `json:"lMorph"`
			}
			Output TokenLabels
		}
	}
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	checker, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range fixture.Readable {
		if got := ReadableMorphology(c.Input); got != c.Output {
			t.Errorf("readable %q got %q want %q", c.Input, got, c.Output)
		}
	}
	for _, c := range fixture.Words {
		a, b, d := SplitWord(c.Input)
		if [3]string{a, b, d} != c.Split || AnalyzeWord(c.Input) != c.Meaning {
			t.Errorf("split/meaning mismatch %q", c.Input)
		}
		want := []WordAnalysis{}
		for _, raw := range c.Analysis {
			var tuple []json.RawMessage
			if err = json.Unmarshal(raw, &tuple); err != nil {
				t.Fatal(err)
			}
			var w WordAnalysis
			if err = json.Unmarshal(tuple[0], &w.Word); err != nil {
				t.Fatal(err)
			}
			var pairs [][2]string
			if err = json.Unmarshal(tuple[1], &pairs); err != nil {
				t.Fatal(err)
			}
			for _, pair := range pairs {
				w.Morphologies = append(w.Morphologies, MorphologyLabel{pair[0], pair[1]})
			}
			want = append(want, w)
		}
		if got := checker.Analyze(c.Input); !reflect.DeepEqual(got, want) {
			t.Errorf("analysis %q got %#v want %#v", c.Input, got, want)
		}
	}
	for _, c := range fixture.Tokens {
		if got := LabelToken(c.Input.Type, c.Input.Value, c.Input.Morphology, checker); !reflect.DeepEqual(got, c.Output) {
			t.Errorf("labels %q/%s cached=%v got %#v want %#v", c.Input.Value, c.Input.Type, c.Input.Morphology, got, c.Output)
		}
	}
}
