package french

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"
)

func readFixture(t *testing.T, name string, v any) {
	t.Helper()
	b, e := os.ReadFile("testdata/" + name + ".json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, v); e != nil {
		t.Fatal(e)
	}
}
func TestConjugationPythonParity(t *testing.T) {
	var f struct {
		Tenses, Persons []string
		Cases           []struct {
			Word, SHA256, Type string
			Names, Similar     []string
		}
	}
	readFixture(t, "conjugation", &f)
	if len(f.Cases) != 8403 {
		t.Fatal("incomplete reference")
	}
	for _, c := range f.Cases {
		h := sha256.New()
		for _, tense := range f.Tenses {
			for _, person := range f.Persons {
				got, e := GetConjugation(c.Word, tense, person)
				if e != nil {
					t.Fatal(e)
				}
				has, e := HasConjugation(c.Word, tense, person)
				if e != nil {
					t.Fatal(e)
				}
				flag := "0"
				if has {
					flag = "1"
				}
				for _, s := range []string{c.Word, tense, person, got, flag} {
					h.Write([]byte(s))
					h.Write([]byte{0})
				}
			}
		}
		if hex.EncodeToString(h.Sum(nil)) != c.SHA256 {
			t.Errorf("conjugation mismatch: %s", c.Word)
		}
		if got, ok := VerbType(c.Word); !ok || got != c.Type {
			t.Errorf("verb type %s", c.Word)
		}
		if !reflect.DeepEqual(DerivedNames(c.Word), c.Names) {
			t.Errorf("derived names %s", c.Word)
		}
		if !reflect.DeepEqual(SimilarInfinitiveForms(c.Word), c.Similar) {
			t.Errorf("similar forms %s", c.Word)
		}
	}
}
func TestConjugationErrors(t *testing.T) {
	if IsVerb("not-a-verb") {
		t.Fatal("unknown verb accepted")
	}
	if _, e := GetConjugation("not-a-verb", ":Ip", ":1s"); !errors.Is(e, ErrUnknownVerb) {
		t.Fatalf("unknown verb: %v", e)
	}
	if _, e := GetConjugation("aimer", ":unknown", ":1s"); !errors.Is(e, ErrUnknownTense) {
		t.Fatalf("unknown tense: %v", e)
	}
}
func TestInflectionPythonParity(t *testing.T) {
	var f struct {
		Cases []struct {
			Word             string
			Masculine        bool
			Singular, Plural []string
			HasMisc          bool `json:"has_misc"`
			Misc             []string
		}
	}
	readFixture(t, "inflection", &f)
	for _, c := range f.Cases {
		if IsMasculineForm(c.Word) != c.Masculine || HasIrregularPlural(c.Word) != c.HasMisc || !reflect.DeepEqual(FeminineForms(c.Word, false), c.Singular) || !reflect.DeepEqual(FeminineForms(c.Word, true), c.Plural) || !reflect.DeepEqual(IrregularPlurals(c.Word), c.Misc) {
			t.Errorf("inflection mismatch %s", c.Word)
		}
	}
}
func TestPhoneticPythonParity(t *testing.T) {
	var f struct {
		Cases []struct {
			Word        string
			Similar     []string
			Has         bool
			Selected    map[string][]string
			HasFiltered map[string]bool `json:"has_filtered"`
		}
	}
	readFixture(t, "phonetic", &f)
	for _, c := range f.Cases {
		if !reflect.DeepEqual(PhoneticSimilar(c.Word), c.Similar) {
			t.Errorf("similar: %s", c.Word)
		}
		got, e := HasPhoneticSimilar(c.Word, "")
		if e != nil || got != c.Has {
			t.Errorf("has similar %s", c.Word)
		}
		for pattern, want := range c.Selected {
			got, e := SelectPhoneticSimilar(c.Word, pattern)
			if e != nil || !reflect.DeepEqual(got, want) {
				t.Errorf("select %s %s got %v want %v err %v", c.Word, pattern, got, want, e)
			}
			has, e := HasPhoneticSimilar(c.Word, pattern)
			if e != nil || has != c.HasFiltered[pattern] {
				t.Errorf("has filtered %s %s", c.Word, pattern)
			}
		}
	}
}
func TestPhoneticErrorsAndCopies(t *testing.T) {
	if _, e := SelectPhoneticSimilar("sang", "(?=x)"); e == nil {
		t.Fatal("unsupported pattern accepted")
	}
	if !ArePhoneticallySimilar("SANG", "sans") || ArePhoneticallySimilar("unknown", "sans") {
		t.Fatal("phonetic sets differ")
	}
	s := PhoneticSimilar("sang")
	s[0] = "modified"
	if PhoneticSimilar("sang")[0] == "modified" {
		t.Fatal("shared data escaped")
	}
}
func TestVerbTablesPythonParity(t *testing.T) {
	var f struct {
		Cases []struct {
			Word, Info, Auxiliary, Raw, Group string
			Flags                             []bool
			Table                             map[string]string
			Incomplete                        bool
			PronominalLabel                   string `json:"pronominal_label"`
			Pronominal                        int
		}
	}
	readFixture(t, "verbs", &f)
	for _, c := range f.Cases {
		v, e := NewVerb(c.Word, "")
		if e != nil {
			t.Fatal(e)
		}
		o := VerbOptions{c.Flags[0], c.Flags[1], c.Flags[2], c.Flags[3], c.Flags[4]}
		if v.Info != c.Info || v.Auxiliary != c.Auxiliary || v.RawInfo != c.Raw || v.Group != c.Group || v.Incomplete != c.Incomplete || v.PronominalLabel != c.PronominalLabel || v.Pronominal != c.Pronominal {
			t.Errorf("verb metadata %s", c.Word)
		}
		got := v.Table(o)
		if !reflect.DeepEqual(got, c.Table) {
			for k, want := range c.Table {
				if got[k] != want {
					t.Errorf("%s flags %v %s got %q want %q", c.Word, c.Flags, k, got[k], want)
				}
			}
		}
	}
}
func TestThesaurusPythonParity(t *testing.T) {
	var f struct {
		Cases []struct {
			Word     string
			Synonyms []SynonymSet
		}
	}
	readFixture(t, "thesaurus", &f)
	for _, c := range f.Cases {
		if got := Synonyms(c.Word); !reflect.DeepEqual(got, c.Synonyms) {
			t.Errorf("synonyms %s", c.Word)
		}
	}
}
