package french

import "testing"

func TestMorphologyPythonParity(t *testing.T) {
	var f struct {
		Cases []struct {
			Morphologies []string
			GenderNumber [2]string `json:"gender_number"`
			Predicates   map[string]bool
		}
		Single []struct {
			Value, Lemma string
			Matches      map[string]bool
		}
	}
	readFixture(t, "morphology", &f)
	for _, c := range f.Cases {
		g, n := GenderNumber(c.Morphologies)
		if g != c.GenderNumber[0] || n != c.GenderNumber[1] {
			t.Errorf("gender/number %v", c.Morphologies)
		}
		for name, want := range c.Predicates {
			got, e := MorphologyPredicate(name, c.Morphologies)
			if e != nil || got != want {
				t.Errorf("%s %v got %v want %v err %v", name, c.Morphologies, got, want, e)
			}
		}
	}
	for _, c := range f.Single {
		if LemmaOfMorphology(c.Value) != c.Lemma {
			t.Errorf("lemma %q", c.Value)
		}
		for name, want := range c.Matches {
			got, e := MatchMorphology(name, c.Value)
			if e != nil || got != want {
				t.Errorf("pattern %s %q got %v want %v err %v", name, c.Value, got, want, e)
			}
		}
	}
}
