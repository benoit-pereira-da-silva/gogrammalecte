package grammar

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestFullInformationPythonParity(t *testing.T) {
	file, err := os.Open("testdata/fullinfo.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	var cases []struct {
		Input    string
		Analysis any
	}
	if err = json.NewDecoder(reader).Decode(&cases); err != nil {
		t.Fatal(err)
	}
	engine, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Input, func(t *testing.T) {
			got, err := engine.Analyze(context.Background(), tc.Input, ParseOptions{Context: true})
			if err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			var actual any
			if err = json.Unmarshal(raw, &actual); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(actual, tc.Analysis) {
				expected, _ := json.Marshal(tc.Analysis)
				t.Errorf("got %s\nwant %s", raw, expected)
			}
		})
	}
}

func TestDiagnosticContextEmptyFields(t *testing.T) {
	engine := engineForSource(t, "__[s](context)__ x <<- ->> y && message\n")
	diagnostics, err := engine.Check(context.Background(), "x", ParseOptions{Context: true})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(diagnostics[0])
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	json.Unmarshal(raw, &fields)
	for _, key := range []string{"sUnderlined", "sBefore", "sAfter"} {
		if _, exists := fields[key]; !exists {
			t.Errorf("requested context field %s is absent in %s", key, raw)
		}
	}
	var roundtrip Error
	if err = json.Unmarshal(raw, &roundtrip); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(roundtrip, diagnostics[0]) {
		t.Fatalf("context diagnostic JSON roundtrip differs: %#v %#v", roundtrip, diagnostics[0])
	}
}
