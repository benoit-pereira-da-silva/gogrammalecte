package french

import "testing"

func TestUnitPythonParity(t *testing.T) {
	var f struct {
		Units []struct {
			Input    string
			Expected bool
		}
	}
	readFixture(t, "analyser", &f)
	for _, c := range f.Units {
		if got := PossiblyUnit(c.Input); got != c.Expected {
			t.Errorf("unit %q got %v want %v", c.Input, got, c.Expected)
		}
	}
}

type morphologyFixture map[string][]string

func (m morphologyFixture) Lookup(s string) bool    { _, ok := m[s]; return ok }
func (m morphologyFixture) Morph(s string) []string { return m[s] }
func TestAnalyserPythonParity(t *testing.T) {
	var f struct {
		Lexicon morphologyFixture
		Pairs   []struct {
			First, Second, POS    string
			Apposition, Agreement bool
			AgreementAny          bool `json:"agreement_any"`
		}
	}
	readFixture(t, "analyser", &f)
	for _, c := range f.Pairs {
		if IsApposition(f.Lexicon, c.First, c.Second) != c.Apposition {
			t.Errorf("apposition %s %s", c.First, c.Second)
		}
		if CombinedNounPOS(f.Lexicon, c.First, c.Second) != c.POS {
			t.Errorf("POS %s %s", c.First, c.Second)
		}
		a, b := MorphToken{Value: c.First}, MorphToken{Value: c.Second}
		if TokensAgree(f.Lexicon, a, b, true) != c.Agreement || TokensAgree(f.Lexicon, a, b, false) != c.AgreementAny {
			t.Errorf("agreement %s %s", c.First, c.Second)
		}
	}
}
