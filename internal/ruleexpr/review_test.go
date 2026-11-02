package ruleexpr

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type reviewEnvironment struct{ fixtureEnv }

func (reviewEnvironment) Resolve(name string) (any, error) {
	if name == "empty" {
		return map[string]bool{}, nil
	}
	return fixtureEnv{}.Resolve(name)
}
func TestReviewedPythonSemantics(t *testing.T) {
	data, err := os.ReadFile("testdata/review.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Expression string
		Result     any
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		e, err := Compile(c.Expression)
		if err != nil {
			t.Errorf("%s: %v", c.Expression, err)
			continue
		}
		got, err := e.Eval(reviewEnvironment{})
		if n, ok := c.Result.(float64); ok {
			c.Result = int(n)
		}
		if err != nil || !reflect.DeepEqual(got, c.Result) {
			t.Errorf("%s got %#v (%v), want %#v", c.Expression, got, err, c.Result)
		}
	}
}
func TestReviewedExpressionsRejectUnsafeOrUnsupportedValues(t *testing.T) {
	for _, source := range []string{`"ab" * 9223372036854775807`, `int("1__2")`, `int("²")`, `int("\x1c1")`, `+"a"`, `str({1,2})`} {
		e, err := Compile(source)
		if err != nil {
			continue
		}
		if _, err = e.Eval(reviewEnvironment{}); err == nil {
			t.Errorf("invalid or unsupported expression accepted: %s", source)
		}
	}
	for _, source := range []string{`"\U00110000"`, `"\uD800"`, `01`} {
		if _, err := Compile(source); err == nil {
			t.Errorf("unsupported literal accepted: %s", source)
		}
	}
}
