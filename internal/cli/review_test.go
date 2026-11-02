package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestIgnoreTokenGraphRule(t *testing.T) {
	var output bytes.Buffer
	if err := execute([]string{"check", "--ignore", "g2__conj_il__b1_a1_1"}, strings.NewReader("Il mues."), &output); err != nil {
		t.Fatal(err)
	}
	var diagnostics []struct {
		RuleID string `json:"sRuleId"`
	}
	if err := json.Unmarshal(output.Bytes(), &diagnostics); err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.RuleID == "g2__conj_il__b1_a1_1" {
			t.Fatalf("ignored token graph rule still emitted: %s", output.String())
		}
	}
}
