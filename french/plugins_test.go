package french

import "testing"

func TestDatePythonParity(t *testing.T) {
	var f struct {
		Cases []struct {
			Day, Month, Year, Weekday string
			Valid                     bool
			Matches                   map[string]bool
		}
	}
	readFixture(t, "dates", &f)
	for _, c := range f.Cases {
		_, valid := CheckDate(c.Day, c.Month, c.Year)
		if valid != c.Valid || WeekdayName(c.Day, c.Month, c.Year) != c.Weekday {
			t.Errorf("date mismatch %s/%s/%s", c.Day, c.Month, c.Year)
		}
		for weekday, want := range c.Matches {
			if WeekdayMatches(weekday, c.Day, c.Month, c.Year) != want {
				t.Errorf("weekday mismatch %s %s/%s/%s", weekday, c.Day, c.Month, c.Year)
			}
		}
	}
}
func TestTypographyPythonParity(t *testing.T) {
	type Pair struct{ Input, Output string }
	var f struct {
		Numbers []struct {
			Input, Output string
			Simple        bool
		}
		Demonstratives, Standards, Inclusive, Ligatures []Pair
		Split                                           []struct {
			Input  string
			Output [2]string
		}
		Joins []struct{ Verb, Suffix, Output string }
	}
	readFixture(t, "typography", &f)
	for _, c := range f.Numbers {
		if got := FormatNumber(c.Input, c.Simple); got != c.Output {
			t.Errorf("FormatNumber %q got %q want %q", c.Input, got, c.Output)
		}
	}
	for _, c := range f.Demonstratives {
		if got := SuggestDemonstrative(c.Input); got != c.Output {
			t.Errorf("demonstrative %q got %q want %q", c.Input, got, c.Output)
		}
	}
	for _, c := range f.Standards {
		if got := FormatFrenchStandard(c.Input); got != c.Output {
			t.Errorf("standard %q got %q want %q", c.Input, got, c.Output)
		}
	}
	for _, c := range f.Inclusive {
		if NormalizeInclusiveWriting(c.Input) != c.Output {
			t.Errorf("inclusive %s", c.Input)
		}
	}
	for _, c := range f.Ligatures {
		if UndoLigature(c.Input) != c.Output {
			t.Errorf("ligature %s", c.Input)
		}
	}
	for _, c := range f.Split {
		v, s := SplitVerb(c.Input)
		if v != c.Output[0] || s != c.Output[1] {
			t.Errorf("split %s", c.Input)
		}
	}
	for _, c := range f.Joins {
		if JoinVerbAndSuffix(c.Verb, c.Suffix) != c.Output {
			t.Errorf("join %s%s", c.Verb, c.Suffix)
		}
	}
}
