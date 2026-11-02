package strtransform

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
)

type outcome struct {
	Result string `json:"result"`
	Error  string `json:"error"`
}

type affixFixtures struct {
	Pairs []struct {
		Name       string  `json:"name"`
		Flexion    string  `json:"flexion"`
		Stem       string  `json:"stem"`
		SuffixCode string  `json:"suffix_code"`
		AffixCode  string  `json:"affix_code"`
		Suffix     outcome `json:"suffix"`
		Affix      outcome `json:"affix"`
	} `json:"pairs"`
	Decodes []struct {
		Kind string `json:"kind"`
		Word string `json:"word"`
		Code string `json:"code"`
		outcome
	} `json:"decodes"`
}

func loadAffixFixtures(t *testing.T) affixFixtures {
	t.Helper()
	data, err := os.ReadFile("testdata/affix.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures affixFixtures
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures.Pairs) != 285 || len(fixtures.Decodes) != 17 {
		t.Fatalf("incomplete reference fixtures: %d pairs, %d decoding cases", len(fixtures.Pairs), len(fixtures.Decodes))
	}
	return fixtures
}

func assertOutcome(t *testing.T, got string, err error, want outcome) {
	t.Helper()
	if want.Error != "" {
		if !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("expected invalid-code error (Python %s), got %q, %v", want.Error, got, err)
		}
		return
	}
	if err != nil || got != want.Result {
		t.Fatalf("got %q, %v; want %q", got, err, want.Result)
	}
}

func TestAffixCodesMatchPython(t *testing.T) {
	for _, pair := range loadAffixFixtures(t).Pairs {
		t.Run(pair.Name, func(t *testing.T) {
			if got := DefineSuffixCode(pair.Flexion, pair.Stem); got != pair.SuffixCode {
				t.Errorf("suffix code = %q, want %q", got, pair.SuffixCode)
			}
			if got := DefineAffixCode(pair.Flexion, pair.Stem); got != pair.AffixCode {
				t.Errorf("affix code = %q, want %q", got, pair.AffixCode)
			}
			got, err := ChangeWordWithSuffixCode(pair.Flexion, pair.SuffixCode)
			assertOutcome(t, got, err, pair.Suffix)
			got, err = ChangeWordWithAffixCode(pair.Flexion, pair.AffixCode)
			assertOutcome(t, got, err, pair.Affix)
		})
	}
}

func TestCodeDecodingMatchesPython(t *testing.T) {
	for _, fixture := range loadAffixFixtures(t).Decodes {
		t.Run(fixture.Kind+"/"+fixture.Code, func(t *testing.T) {
			var got string
			var err error
			if fixture.Kind == "suffix" {
				got, err = ChangeWordWithSuffixCode(fixture.Word, fixture.Code)
			} else {
				got, err = ChangeWordWithAffixCode(fixture.Word, fixture.Code)
			}
			assertOutcome(t, got, err, fixture.outcome)
		})
	}
}
