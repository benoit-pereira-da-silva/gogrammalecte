package tokenizer

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestPythonParity(t *testing.T) {
	var f struct {
		Cases []struct {
			Language, Input string
			Edges           bool
			Tokens          []Token
		}
	}
	b, e := os.ReadFile("testdata/tokens.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, &f); e != nil {
		t.Fatal(e)
	}
	if len(f.Cases) == 0 {
		t.Fatal("empty fixture")
	}
	for _, c := range f.Cases {
		t.Run(c.Language+":"+c.Input, func(t *testing.T) {
			t.Parallel()
			got := New(c.Language).Tokens(c.Input, c.Edges)
			if !reflect.DeepEqual(got, c.Tokens) {
				t.Errorf("got %#v, want %#v", got, c.Tokens)
			}
		})
	}
}
