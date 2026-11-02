package grammar

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"testing"
)

func TestFullPythonDiagnosticParity(t *testing.T) {
	f, err := os.Open("testdata/engine.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	var cases []struct {
		Line                 int
		Input, Option        string
		Errors               []Error
		UnorderedSuggestions []string `json:"unordered_suggestions"`
	}
	if err := json.NewDecoder(z).Decode(&cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 14160 {
		t.Fatalf("incomplete corpus: %d cases", len(cases))
	}
	engine, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	failures := 0
	for _, tc := range cases {
		options := ParseOptions{}
		if tc.Option != "" {
			options.Options = map[string]bool{tc.Option: true}
		}
		got, err := engine.Check(context.Background(), tc.Input, options)
		// Only these six cases changed order across Python hash seeds 0,1,2,3.
		// Every other suggestion list and every diagnostic retains strict order.
		for i := range got {
			if slices.Contains(tc.UnorderedSuggestions, got[i].RuleID) {
				got[i].Suggestions = slices.Clone(got[i].Suggestions)
				slices.Sort(got[i].Suggestions)
			}
		}
		for i := range tc.Errors {
			if slices.Contains(tc.UnorderedSuggestions, tc.Errors[i].RuleID) {
				slices.Sort(tc.Errors[i].Suggestions)
			}
		}
		if err != nil || !reflect.DeepEqual(got, tc.Errors) {
			failures++
			if failures <= 35 {
				t.Errorf("line %d %q:\n got %+v\nwant %+v\nerror: %v", tc.Line, tc.Input, got, tc.Errors, err)
			}
		}
	}
	if failures > 0 {
		t.Fatalf("%d / %d diagnostic cases differ", failures, len(cases))
	}
}
