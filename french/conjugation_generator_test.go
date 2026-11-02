package french

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"testing"
)

func TestConjugationGenerationPythonParity(t *testing.T) {
	var fixture struct {
		Cases []struct {
			Word     string
			Variable bool
			Group    string
			Count    int
			Hash     string
		}
		Rules []struct {
			Word     string
			Variable bool
			Group    int
			Expected []ConjugationRule
		}
	}
	readFixture(t, "conj_generator", &fixture)
	for _, c := range fixture.Cases {
		forms := GenerateConjugations(c.Word, DefaultVerbTag, c.Variable)
		h := sha256.New()
		for _, f := range forms {
			fmt.Fprintf(h, "%s\x00%s\x00", f.Word, f.Tags)
		}
		if len(forms) != c.Count || fmt.Sprintf("%x", h.Sum(nil)) != c.Hash || GuessVerbGroup(c.Word) != c.Group {
			t.Errorf("generation mismatch for %q variable=%v", c.Word, c.Variable)
		}
	}
	for _, c := range fixture.Rules {
		if got := ConjugationRules(c.Word, c.Variable, c.Group); !reflect.DeepEqual(got, c.Expected) {
			t.Errorf("rules mismatch for %q variable=%v group=%d", c.Word, c.Variable, c.Group)
		}
	}
}
