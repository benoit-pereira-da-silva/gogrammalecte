package pyregex

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestPinnedPythonPatterns(t *testing.T) {
	f, err := os.Open("testdata/patterns.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	var cases []struct {
		Pattern, Input string
		Matches        []struct {
			Spans  [][2]int
			Groups []*string
		}
	}
	if err := json.NewDecoder(z).Decode(&cases); err != nil {
		t.Fatal(err)
	}
	cache := map[string]*Regexp{}
	for _, tc := range cases {
		r, ok := cache[tc.Pattern]
		if !ok {
			r, err = Compile(tc.Pattern)
			if err != nil {
				t.Errorf("compile %q: %v", tc.Pattern, err)
				continue
			}
			cache[tc.Pattern] = r
		}
		got, err := r.FindAll(context.Background(), tc.Input, -1)
		if err != nil {
			t.Errorf("find %q in %q: %v", tc.Pattern, tc.Input, err)
			continue
		}
		if len(got) != len(tc.Matches) {
			t.Errorf("%q in %q: %d matches, want %d", tc.Pattern, tc.Input, len(got), len(tc.Matches))
			continue
		}
		for i, m := range got {
			if !reflect.DeepEqual(m.Spans, tc.Matches[i].Spans) {
				t.Errorf("%q in %q: spans %v want %v", tc.Pattern, tc.Input, m.Spans, tc.Matches[i].Spans)
			}
			for j, g := range tc.Matches[i].Groups {
				if g != nil && m.Group(j) != *g {
					t.Errorf("group %d got %q want %q", j, m.Group(j), *g)
				}
			}
		}
	}
}

func TestErrorsAndCancellation(t *testing.T) {
	for _, p := range []string{`(`, `(?<=a+)b`, `\1`, `(?P=name)`} {
		if _, err := Compile(p); err == nil {
			t.Errorf("invalid pattern accepted: %s", p)
		}
	}
	r, err := Compile(`(a+)+b`)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.FindAll(ctx, "aaaa", -1); err == nil {
		t.Fatal("cancellation ignored")
	}
}

func TestInitialClosingBracketInClass(t *testing.T) {
	for _, pattern := range []string{`[]a]`, `(?<=[]a])b`, `[^]a]+`} {
		r, err := Compile(pattern)
		if err != nil {
			t.Fatal(err)
		}
		input := "]"
		if pattern == `(?<=[]a])b` {
			input = "]b"
		}
		if pattern == `[^]a]+` {
			input = "bbb"
		}
		m, err := r.Search(context.Background(), input)
		if err != nil || m == nil {
			t.Fatalf("%s in %q: %v %v", pattern, input, m, err)
		}
	}
}

func TestNamedCaptureAccess(t *testing.T) {
	r, err := Compile(`(?P<word>\w+)`)
	if err != nil {
		t.Fatal(err)
	}
	m, err := r.Search(context.Background(), "été")
	if err != nil {
		t.Fatal(err)
	}
	index, ok := m.GroupIndex("word")
	if !ok || index != 1 || m.Group(index) != "été" {
		t.Fatalf("named capture unavailable: %d %v", index, ok)
	}
}

func TestTurkishCaseFoldInNegatedClasses(t *testing.T) {
	// Captured from Python 3.14.3: all four forms share the same insensitive
	// match class, including when the class is negated before matching.
	for _, pattern := range []string{`(?i)[^i]`, `(?i)[^a-z]`, `(?i)[^İ]`, `(?i)[^I]`, `(?i)[^ı]`} {
		r, err := Compile(pattern)
		if err != nil {
			t.Fatal(err)
		}
		for _, input := range []string{"i", "I", "İ", "ı"} {
			m, err := r.Search(context.Background(), input)
			if err != nil || m != nil {
				t.Errorf("%s incorrectly matched %q: %v %v", pattern, input, m, err)
			}
		}
		if m, err := r.Search(context.Background(), "é"); err != nil || m == nil {
			t.Errorf("%s failed to match é: %v %v", pattern, m, err)
		}
	}
	for _, pattern := range []string{`(?i)[İ]`, `(?i)[ı]`, `(?i)[iİ]`} {
		r, err := Compile(pattern)
		if err != nil {
			t.Fatal(err)
		}
		for _, input := range []string{"i", "I", "İ", "ı"} {
			m, err := r.Search(context.Background(), input)
			if err != nil || m == nil {
				t.Errorf("%s failed to match %q", pattern, input)
			}
		}
	}
}
