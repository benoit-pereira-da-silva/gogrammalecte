package french

import (
	"encoding/json"
	"fmt"
	"testing"
)

type suggestionFixture struct {
	Morphologies map[string][]string
	Valid        map[string]bool
	Cases        []struct {
		Function string
		Args     []json.RawMessage
		Expected json.RawMessage
	}
}

func (f suggestionFixture) Morph(word string) []string { return f.Morphologies[word] }
func (f suggestionFixture) IsValid(word string) bool   { return f.Valid[word] }

func TestSuggestionPythonParity(t *testing.T) {
	var fixture suggestionFixture
	readFixture(t, "suggestions", &fixture)
	s := NewSuggester(fixture)
	for _, c := range fixture.Cases {
		str := func(i int) string {
			if i >= len(c.Args) {
				return ""
			}
			var v string
			if err := json.Unmarshal(c.Args[i], &v); err != nil {
				t.Fatal(err)
			}
			return v
		}
		boolean := func(i int) bool {
			if i >= len(c.Args) {
				return false
			}
			var v bool
			if err := json.Unmarshal(c.Args[i], &v); err != nil {
				t.Fatal(err)
			}
			return v
		}
		var got any
		var err error
		switch c.Function {
		case "suggVerb":
			got = s.Verb(str(0), str(1), boolean(2))
		case "suggVerbPpas":
			got = s.PastParticiple(str(0), str(1))
		case "suggVerbTense":
			got, err = s.VerbTense(str(0), str(1), str(2))
		case "suggVerbFrom":
			got = s.VerbFrom(str(0), str(1), str(2))
		case "suggVerbImpe":
			got = s.Imperative(str(0), boolean(1))
		case "suggVerbInfi":
			got = s.Infinitives(str(0))
		case "suggVerbMode":
			got, err = s.VerbMode(str(0), str(1), str(2))
		case "suggPlur":
			got = s.Plural(str(0), boolean(1))
		case "suggSing":
			got = s.Singular(str(0), boolean(1))
		case "suggMasSing":
			got = s.MasculineSingular(str(0), boolean(1))
		case "suggMasPlur":
			got = s.MasculinePlural(str(0), boolean(1))
		case "suggFemSing":
			got = s.FeminineSingular(str(0), boolean(1))
		case "suggFemPlur":
			got = s.FemininePlural(str(0), boolean(1))
		case "suggAgree":
			got = s.Agree(str(0), str(1))
		case "hasFemForm":
			got = s.HasFeminineForm(str(0))
		case "hasMasForm":
			got = s.HasMasculineForm(str(0))
		case "switchGender":
			var plural *bool
			if err := json.Unmarshal(c.Args[1], &plural); err != nil {
				t.Fatal(err)
			}
			got = s.SwitchGender(str(0), plural)
		case "switchPlural":
			got = s.SwitchPlural(str(0))
		case "suggSimil":
			got, err = s.Similar(str(0), str(1), boolean(2), boolean(3))
		case "suggLesLa":
			got = s.LesLa(str(0))
		default:
			t.Fatalf("unknown fixture function: %s", c.Function)
		}
		encoded, _ := json.Marshal(got)
		if err != nil || string(encoded) != string(c.Expected) {
			t.Errorf("%s(%s): got %s (%v), want %s", c.Function, c.Args, encoded, err, c.Expected)
		}
	}
}

func TestSuggestionRejectsUnsupportedExpressions(t *testing.T) {
	s := NewSuggester(suggestionFixture{})
	if _, err := s.Similar("test", "(?<=x)y", false, false); err == nil {
		t.Fatal("expected unsupported lookbehind error")
	}
	if _, err := s.VerbMode("est", ":Invalid", "il"); err == nil {
		t.Fatal("expected unknown tense error")
	}
	if _, err := s.VerbTense("est", ":bad", ":3s"); err == nil {
		t.Fatal("expected unknown tense error")
	}
}

func TestSynonymTupleValidation(t *testing.T) {
	var set SynonymSet
	if err := json.Unmarshal([]byte(`["noun"]`), &set); err == nil {
		t.Fatal("expected malformed tuple error")
	} else if fmt.Sprint(err) == "" {
		t.Fatal("empty tuple error")
	}
}
