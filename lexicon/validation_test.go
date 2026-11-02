package lexicon

import (
	"encoding/json"
	"os"
	"testing"
)

func TestValidationReference(t *testing.T) {
	var cases []struct{ Line, Output string }
	data, err := os.ReadFile("testdata/validation.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		e := ParseEntry(c.Line)
		if got := e.ValidationMessage(); got != c.Output {
			t.Fatalf("validation(%q) = %q, want %q", c.Line, got, c.Output)
		}
	}
	d, err := EmbeddedDictionary()
	if err != nil {
		t.Fatal(err)
	}
	if messages := d.ValidationMessages(); len(messages) != 0 {
		t.Fatalf("pinned dictionary has unexpected warnings: %v", messages)
	}
}
