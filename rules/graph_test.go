package rules

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type graphOracle struct {
	Definitions, Declensions []Definition
	Expansions               []struct {
		Source   string
		Expanded []TokenExpansion
	}
	Paths [][]string
	Graph map[string]json.RawMessage
}

func loadGraphOracle(t *testing.T) graphOracle {
	t.Helper()
	b, e := os.ReadFile("testdata/graph_compiler.json")
	if e != nil {
		t.Fatal(e)
	}
	var f graphOracle
	if e = json.Unmarshal(b, &f); e != nil {
		t.Fatal(e)
	}
	return f
}
func TestGraphTokenExpansionPythonParity(t *testing.T) {
	f := loadGraphOracle(t)
	for _, c := range f.Expansions {
		t.Run(c.Source, func(t *testing.T) {
			got, e := ExpandGraphTokens(c.Source, f.Definitions, f.Declensions)
			if e != nil {
				t.Fatal(e)
			}
			if !reflect.DeepEqual(got, c.Expanded) {
				t.Fatalf("got %#v, Python returned %#v", got, c.Expanded)
			}
		})
	}
}
func TestDARGPythonParity(t *testing.T) {
	f := loadGraphOracle(t)
	nodes, e := MinimizeGraphPaths(f.Paths)
	if e != nil {
		t.Fatal(e)
	}
	got := graphAsPython(nodes)
	raw, _ := json.Marshal(got)
	var actual map[string]json.RawMessage
	json.Unmarshal(raw, &actual)
	if len(actual) != len(f.Graph) {
		t.Fatalf("got %d nodes, want %d", len(actual), len(f.Graph))
	}
	for id, want := range f.Graph {
		var a, b any
		json.Unmarshal(actual[id], &a)
		json.Unmarshal(want, &b)
		if !reflect.DeepEqual(a, b) {
			t.Errorf("node %s: got %s, want %s", id, actual[id], want)
		}
	}
}
func TestCompileGraphActionsAndCaptures(t *testing.T) {
	source := `[++]
@@@@GRAPH: test | T
__capture!7__
    ({DET}) ?très¿ (grand) chat
        <<- /agreement/ morph(\2, ":A") -2:3:>> =\1+"|"+\2 && ="message "+\2 | https://example.test/rule
        <<- =>> define(\2, ":A")

@@@@END_GRAPH
`
	source = "DEF: DET [le|la]\n" + source
	doc, e := Parse(source)
	if e != nil {
		t.Fatal(e)
	}
	compiled, e := CompileGraphs(doc)
	if e != nil {
		t.Fatal(e)
	}
	if len(compiled) != 1 {
		t.Fatal(len(compiled))
	}
	g := compiled[0]
	if g.RuleCount != 8 {
		t.Fatalf("expanded rules %d", g.RuleCount)
	}
	if len(g.Actions) != 4 {
		t.Fatalf("actions %#v", g.Actions)
	}
	first := g.Actions[0]
	if first.Condition != `morph(\2, ":A")` || first.StartToken != 2 || first.EndToken != 3 || first.Priority != 7 || first.CaseSensitive || !first.ValueExpression || !first.MessageExpression {
		t.Fatalf("unexpected action %#v", first)
	}
	if first.URL != "https://example.test/rule" {
		t.Fatal(first.URL)
	}
}
func TestGraphAntipatternAndUnknownStatements(t *testing.T) {
	source := `[++]
@@@@GRAPH: test | T
__rule__
    !! le chat ¡¡
    [le|un] chat
        <<- ->> chien && replacement

@@@@END_GRAPH
`
	doc, e := Parse(source)
	if e != nil {
		t.Fatal(e)
	}
	gs, e := CompileGraphs(doc)
	if e != nil {
		t.Fatal(e)
	}
	if gs[0].RuleCount != 1 {
		t.Fatal(gs[0].RuleCount)
	}
	bad := *doc
	bad.Graphs = append([]Graph{}, doc.Graphs...)
	bad.Graphs[0].Groups = []GraphGroup{{Name: "invalid", Source: []SourceLine{{Line: 5, Text: " unexpected"}}}}
	if _, e := CompileGraphs(&bad); e == nil {
		t.Fatal("accepted unrecognized graph statement")
	}
}
func TestCompileEmbeddedGraphs(t *testing.T) {
	source, e := FrenchSource()
	if e != nil {
		t.Fatal(e)
	}
	doc, e := Parse(source)
	if e != nil {
		t.Fatal(e)
	}
	graphs, e := CompileGraphs(doc)
	if e != nil {
		t.Fatal(e)
	}
	embedded, e := FrenchGraphs()
	if e != nil {
		t.Fatal(e)
	}
	generatedJSON, _ := json.Marshal(graphs)
	embeddedJSON, _ := json.Marshal(embedded)
	if string(generatedJSON) != string(embeddedJSON) {
		t.Fatal("native regenerated graphs differ from embedded resource")
	}
	f, e := os.Open("testdata/graphs_full.json.gz")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	z, e := gzip.NewReader(f)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	var oracle []struct {
		Name, Code string
		RuleCount  int `json:"rule_count"`
		Graph      map[string]json.RawMessage
		Lemmas     map[string]int
		Actions    map[string][]any
	}
	if e = json.NewDecoder(z).Decode(&oracle); e != nil {
		t.Fatal(e)
	}
	if len(graphs) != len(oracle) {
		t.Fatalf("got %d graphs, want %d", len(graphs), len(oracle))
	}
	for i, g := range graphs {
		want := oracle[i]
		t.Run(g.Name, func(t *testing.T) {
			if g.Name != want.Name || g.Code != want.Code || g.RuleCount != want.RuleCount || !reflect.DeepEqual(g.Lemmas, want.Lemmas) {
				t.Fatalf("graph identity, expanded paths or lemmas differ: %s %s %d / %s %s %d", g.Name, g.Code, g.RuleCount, want.Name, want.Code, want.RuleCount)
			}
			actual := graphAsPython(g.Nodes)
			if len(actual) != len(want.Graph) {
				t.Fatalf("got %d nodes, want %d", len(actual), len(want.Graph))
			}
			for id, raw := range want.Graph {
				var expected, got any
				json.Unmarshal(raw, &expected)
				b, _ := json.Marshal(actual[id])
				json.Unmarshal(b, &got)
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("node %s differs: got %s, want %s", id, b, raw)
				}
			}
			if len(g.Actions) != len(want.Actions) {
				t.Fatalf("got %d actions, want %d", len(g.Actions), len(want.Actions))
			}
			for _, action := range g.Actions {
				expected, ok := want.Actions[action.ID]
				if !ok {
					t.Errorf("unexpected action %s", action.ID)
					continue
				}
				got := graphActionAsPython(action)
				// The reference stores generated function names; this compiler preserves
				// the DSL instead. Compare their expression roles and all literal metadata.
				if expected[2] != "" {
					expected[2] = "<expression>"
				}
				if action.Kind == "=" {
					expected[4] = "<expression>"
				} else if v, ok := expected[4].(string); ok && strings.HasPrefix(v, "=") {
					expected[4] = "=<expression>"
				}
				if action.Kind == "-" {
					if strings.HasPrefix(expected[11].(string), "=") {
						expected[11] = "=<expression>"
					}
				}
				raw, _ := json.Marshal(got)
				var normalized []any
				json.Unmarshal(raw, &normalized)
				if !reflect.DeepEqual(normalized, expected) {
					t.Errorf("%s: got %#v, want %#v", action.ID, normalized, expected)
				}
			}
		})
	}
}
func graphActionAsPython(a GraphAction) []any {
	var option any = false
	if a.Option != "" {
		option = a.Option
	}
	condition := a.Condition
	if condition != "" {
		condition = "<expression>"
	}
	value := a.Value
	if a.ValueExpression {
		value = "=<expression>"
	}
	if a.Kind == "=" {
		value = "<expression>"
	}
	fields := []any{"#" + strconv.Itoa(a.Line), option, condition, a.Kind, value}
	switch a.Kind {
	case "-":
		message := a.Message
		if a.MessageExpression {
			message = "=<expression>"
		}
		return append(fields, a.StartToken, a.EndToken, a.StartLimit, a.EndLimit, a.CaseSensitive, a.Priority, message, a.URL)
	case "~":
		return append(fields, a.StartToken, a.EndToken, a.CaseSensitive)
	case "!", "/", "&":
		return append(fields, a.StartToken, a.EndToken)
	}
	return fields
}

func TestGraphRejectsInvalidRegex(t *testing.T) {
	doc := &Document{Graphs: []Graph{{Name: "bad", Code: "B", Groups: []GraphGroup{{Name: "invalid", Source: []SourceLine{{Line: 1, Text: "    ~["}, {Line: 2, Text: "        <<- ->> replacement && message"}}}}}}}
	if _, err := CompileGraphs(doc); err == nil {
		t.Fatal("accepted malformed graph regex")
	}
}

// graphAsPython is used only by differential tests of the upstream data format.
func graphAsPython(nodes []GraphNode) map[string]any {
	out := map[string]any{}
	for i, node := range nodes {
		if node.Actions != nil {
			out[strconv.Itoa(i)] = node.Actions
			continue
		}
		values := map[string]any{}
		for _, arc := range node.Arcs {
			if arc.Kind == "literal" {
				values[arc.Value] = arc.Next
			} else {
				category, ok := values[arc.Kind].(map[string]int)
				if !ok {
					category = map[string]int{}
					values[arc.Kind] = category
				}
				category[arc.Value] = arc.Next
			}
		}
		out[strconv.Itoa(i)] = values
	}
	return out
}
