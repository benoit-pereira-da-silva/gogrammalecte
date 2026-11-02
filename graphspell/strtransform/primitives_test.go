package strtransform

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"testing"
)

func TestPrimitivePythonParity(t *testing.T) {
	var fixture struct {
		Transform []struct {
			Input, Normal, Simple, Clean, Exponent, Lower, Upper, Capitalize string
			IsUpper, IsTitle, IsDigit, IsAlpha                               bool
			Ngrams                                                           []struct {
				N    int
				Want []string
			}
		}
		Distances []struct {
			A, B, Substring string
			DL              int
			DLX, Jaro       float64
			Sift            []struct{ Offset, Want int }
		}
	}
	data, err := os.ReadFile("testdata/primitives.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Transform) == 0 || len(fixture.Distances) == 0 {
		t.Fatal("empty reference fixture")
	}
	for _, c := range fixture.Transform {
		t.Run(c.Input, func(t *testing.T) {
			t.Parallel()
			for name, pair := range map[string][2]string{"normal": {SpellingNormalization(c.Input), c.Normal}, "simple": {SimplifyWord(c.Input), c.Simple}, "clean": {CleanWord(c.Input), c.Clean}, "exponent": {NumbersToExponent(c.Input), c.Exponent}, "lower": {Lower(c.Input), c.Lower}, "upper": {Upper(c.Input), c.Upper}, "capitalize": {Capitalize(c.Input), c.Capitalize}} {
				if pair[0] != pair[1] {
					t.Errorf("%s(%q)=%q, want %q", name, c.Input, pair[0], pair[1])
				}
			}
			for name, pair := range map[string][2]bool{"upper": {IsUpper(c.Input), c.IsUpper}, "title": {IsTitle(c.Input), c.IsTitle}, "digit": {IsDigit(c.Input), c.IsDigit}, "alpha": {IsAlpha(c.Input), c.IsAlpha}} {
				if pair[0] != pair[1] {
					t.Errorf("is%s(%q)=%v, want %v", name, c.Input, pair[0], pair[1])
				}
			}
			for _, n := range c.Ngrams {
				if got := Ngrams(c.Input, n.N); !reflect.DeepEqual(got, n.Want) {
					t.Errorf("Ngrams(%q,%d)=%q want %q", c.Input, n.N, got, n.Want)
				}
			}
		})
	}
	for _, c := range fixture.Distances {
		t.Run(c.A+"/"+c.B, func(t *testing.T) {
			t.Parallel()
			if got := DistanceDamerauLevenshtein(c.A, c.B); got != c.DL {
				t.Errorf("DL=%d want %d", got, c.DL)
			}
			if got := DistanceDamerauLevenshteinX(c.A, c.B); math.Abs(got-c.DLX) > 1e-12 {
				t.Errorf("DLX=%v want %v", got, c.DLX)
			}
			if got := DistanceJaroWinkler(c.A, c.B); math.Abs(got-c.Jaro) > 1e-12 {
				t.Errorf("Jaro=%v want %v", got, c.Jaro)
			}
			if got := LongestCommonSubstring(c.A, c.B); got != c.Substring {
				t.Errorf("substring=%q want %q", got, c.Substring)
			}
			for _, s := range c.Sift {
				if got := DistanceSift4(c.A, c.B, s.Offset); got != s.Want {
					t.Errorf("Sift4 offset %d =%d want %d", s.Offset, got, s.Want)
				}
			}
		})
	}
}
