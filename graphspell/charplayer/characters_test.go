package charplayer

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestPythonParity(t *testing.T) {
	var f struct {
		Distances []struct {
			A, B string
			Want float64
		}
		Replacements []struct {
			Previous, Current, Next string
			Want                    []string
		}
	}
	b, e := os.ReadFile("testdata/characters.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, &f); e != nil {
		t.Fatal(e)
	}
	for _, c := range f.Distances {
		if g := DistanceBetweenChars([]rune(c.A)[0], []rune(c.B)[0]); g != c.Want {
			t.Errorf("distance %s/%s %v want %v", c.A, c.B, g, c.Want)
		}
	}
	for _, c := range f.Replacements {
		if g := OneToMany(c.Previous, c.Current, c.Next); !reflect.DeepEqual(g, c.Want) {
			t.Errorf("replacement %q/%q/%q %q want %q", c.Previous, c.Current, c.Next, g, c.Want)
		}
	}
}
func TestTablesAreIsolated(t *testing.T) {
	a := TwoToMany("qu")
	a[0] = "changed"
	if TwoToMany("qu")[0] == "changed" {
		t.Fatal("mutable shared table")
	}
	m := KeyboardMap("AZERTY")
	m["a"] = "changed"
	if KeyboardMap("azerty")["a"] == "changed" {
		t.Fatal("mutable keyboard table")
	}
	if !reflect.DeepEqual(KeyboardNames(), []string{"azerty", "bépo", "colemak", "dvorak", "qwerty", "qwertz"}) {
		t.Fatal("keyboard order")
	}
	if DistanceBetweenChars('😀', '😀') != 0 || DistanceBetweenChars('😀', 'a') != 1 {
		t.Fatal("unknown character distance")
	}
}

func TestReplacementContextIsOneCharacter(t *testing.T) {
	if got := OneToMany("bc", "c", ""); !reflect.DeepEqual(got, []string{"cc", "ss", "qu", "ch"}) {
		t.Fatalf("multi-character context incorrectly treated as a consonant: %q", got)
	}
}
