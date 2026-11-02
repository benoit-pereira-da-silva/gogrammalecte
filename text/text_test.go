package text

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type oracleCase struct {
	Name       string
	Input      string
	Paragraphs []string
	Boundaries []Span
	Sentences  []string
}

func loadOracle(t *testing.T) []oracleCase {
	t.Helper()
	data, err := os.ReadFile("testdata/text.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []oracleCase
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Cases) == 0 {
		t.Fatal("the Python oracle fixture must not be empty")
	}
	return fixture.Cases
}

func TestPythonParity(t *testing.T) {
	for _, test := range loadOracle(t) {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			if got := Paragraphs(test.Input); !reflect.DeepEqual(got, test.Paragraphs) {
				t.Errorf("Paragraphs(%q) = %#v; Python returned %#v", test.Input, got, test.Paragraphs)
			}
			if got := SentenceBoundaries(test.Input); !reflect.DeepEqual(got, test.Boundaries) {
				t.Errorf("SentenceBoundaries(%q) = %#v; Python returned %#v", test.Input, got, test.Boundaries)
			}
			if got := Sentences(test.Input); !reflect.DeepEqual(got, test.Sentences) {
				t.Errorf("Sentences(%q) = %#v; Python returned %#v", test.Input, got, test.Sentences)
			}
		})
	}
}
