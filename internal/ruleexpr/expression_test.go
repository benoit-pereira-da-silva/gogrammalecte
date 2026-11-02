package ruleexpr

import (
	"encoding/json"
	"fmt"
	"github.com/benoit-pereira-da-silva/gogrammalecte/rules"
	"os"
	"reflect"
	"strings"
	"testing"
)

type fixtureEnv struct{}

func (fixtureEnv) Resolve(s string) (any, error) { return nil, fmt.Errorf("unknown name %s", s) }
func (fixtureEnv) Reference(kind string, n int) (any, error) {
	if kind == "\\" && n == 1 {
		return "été", nil
	}
	return nil, fmt.Errorf("unknown reference")
}

func TestAllPinnedRuleExpressionsCompile(t *testing.T) {
	source, err := rules.FrenchSource()
	if err != nil {
		t.Fatal(err)
	}
	doc, err := rules.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	expressions := map[string]string{}
	for _, r := range doc.RegexRules {
		for _, a := range r.Actions {
			if a.Condition != "" {
				expressions[a.Condition] = r.ID
			}
			if strings.HasPrefix(a.Value, "=") || a.Kind == "=" {
				expressions[strings.TrimPrefix(a.Value, "=")] = r.ID
			}
			if strings.HasPrefix(a.Message, "=") {
				expressions[a.Message[1:]] = r.ID
			}
		}
	}
	graphs, err := rules.CompileGraphs(doc)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range graphs {
		for _, a := range g.Actions {
			if a.Condition != "" {
				expressions[a.Condition] = a.ID
			}
			if a.ValueExpression {
				expressions[strings.TrimPrefix(a.Value, "=")] = a.ID
			}
			if a.MessageExpression {
				expressions[strings.TrimPrefix(a.Message, "=")] = a.ID
			}
		}
	}
	for expression, id := range expressions {
		if _, err := Compile(expression); err != nil {
			t.Errorf("%s: %v", id, err)
		}
	}
	t.Logf("compiled %d unique source expressions", len(expressions))
}
func (fixtureEnv) Call(name string, args []any) (any, error) {
	return nil, fmt.Errorf("unexpected call %s", name)
}
func TestPythonExpressions(t *testing.T) {
	raw, err := os.ReadFile("testdata/expressions.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Expression string
		Result     any
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		e, err := Compile(tc.Expression)
		if err != nil {
			t.Errorf("compile %s: %v", tc.Expression, err)
			continue
		}
		got, err := e.Eval(fixtureEnv{})
		if n, ok := tc.Result.(float64); ok {
			tc.Result = int(n)
		}
		if err != nil || !reflect.DeepEqual(got, tc.Result) {
			t.Errorf("%s got %#v want %#v err %v", tc.Expression, got, tc.Result, err)
		}
	}
	e, err := Compile(`\1[:-1]+"s"`)
	if err != nil {
		t.Fatal(err)
	}
	got, err := e.Eval(fixtureEnv{})
	if got != "éts" || err != nil {
		t.Fatalf("%v %v", got, err)
	}
	for _, s := range []string{`foo(`, `"abc`, `a b`, `1 +`} {
		if _, err := Compile(s); err == nil {
			t.Errorf("invalid expression accepted: %s", s)
		}
	}
}
