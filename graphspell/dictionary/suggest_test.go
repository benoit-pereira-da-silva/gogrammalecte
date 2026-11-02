package dictionary

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestSuggestionPythonParity(t *testing.T) {
	data, e := os.ReadFile("testdata/suggestions.json")
	if e != nil {
		t.Fatal(e)
	}
	var f struct {
		Cases []struct {
			Dictionary, Word string
			Limit            int
			Split            bool `json:"split_numbers"`
			Suggestions      []string
		}
	}
	if e = json.Unmarshal(data, &f); e != nil {
		t.Fatal(e)
	}
	small, e := Read(bytes.NewReader(fixture(t).Dictionaries[0].Data))
	if e != nil {
		t.Fatal(e)
	}
	full, e := Default()
	if e != nil {
		t.Fatal(e)
	}
	for _, c := range f.Cases {
		t.Run(c.Dictionary+"/"+c.Word, func(t *testing.T) {
			d := small
			if c.Dictionary == "full" {
				d = full
			}
			got := d.Suggest(c.Word, c.Limit, c.Split)
			if !reflect.DeepEqual(got, c.Suggestions) {
				t.Errorf("got %#v, Python returned %#v", got, c.Suggestions)
			}
		})
	}
}
func TestSuggestionCancellation(t *testing.T) {
	d, e := Default()
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := d.SuggestContext(ctx, "gramaire", 10, false); e != context.Canceled {
		t.Fatalf("got %v", e)
	}
}
