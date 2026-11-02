// Graph minimization is adapted from Olivier R.'s darg.py (MPL 2.0), which
// credits Steve Hanov's incremental acyclic word-graph algorithm.
package rules

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// GraphArc is an ordered typed transition. Kind is "literal" or one of the
// upstream categories such as "<lemmas>", "<re_morph>" or "<rules>".
type GraphArc struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
	Next  int    `json:"next"`
}

// GraphNode is either a transition node or an ordered action-list node. A
// non-nil Actions slice identifies the latter, including an empty action list.
type GraphNode struct {
	Arcs    []GraphArc `json:"arcs,omitempty"`
	Actions []string   `json:"actions,omitempty"`
}
type ruleEdge struct {
	label string
	child *ruleNode
}
type ruleNode struct {
	id    int
	final bool
	edges []ruleEdge
}
type pendingRuleEdge struct {
	parent *ruleNode
	index  int
	child  *ruleNode
}

// MinimizeGraphPaths constructs the canonical minimal acyclic graph from token
// paths ending in a "##<source line>" marker and an action identifier.
func MinimizeGraphPaths(input [][]string) ([]GraphNode, error) {
	if len(input) == 0 {
		return []GraphNode{{Arcs: []GraphArc{}}}, nil
	}
	paths := make([][]string, len(input))
	for i, path := range input {
		if len(path) < 2 || !strings.HasPrefix(path[len(path)-2], "##") || path[len(path)-1] == "" {
			return nil, fmt.Errorf("invalid graph action path %d", i)
		}
		paths[i] = append([]string{}, path...)
	}
	sort.SliceStable(paths, func(i, j int) bool {
		a, b := paths[i], paths[j]
		for n := 0; n < len(a) && n < len(b); n++ {
			if a[n] != b[n] {
				return a[n] < b[n]
			}
		}
		return len(a) < len(b)
	})
	root := &ruleNode{}
	nextID := 1
	pending := []pendingRuleEdge{}
	registry := map[string]*ruleNode{}
	nodes := []*ruleNode{}
	keyOf := func(node *ruleNode) string {
		var b strings.Builder
		if node.final {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
		for _, edge := range node.edges {
			b.WriteString(strconv.Itoa(len(edge.label)))
			b.WriteByte(':')
			b.WriteString(edge.label)
			b.WriteByte(':')
			b.WriteString(strconv.Itoa(edge.child.id))
			b.WriteByte(';')
		}
		return b.String()
	}
	minimize := func(downTo int) {
		for i := len(pending) - 1; i >= downTo; i-- {
			edge := pending[i]
			key := keyOf(edge.child)
			if shared, ok := registry[key]; ok {
				edge.parent.edges[edge.index].child = shared
			} else {
				registry[key] = edge.child
				nodes = append(nodes, edge.child)
			}
		}
		pending = pending[:downTo]
	}
	previous := []string{}
	for _, path := range paths {
		common := 0
		for common < len(path) && common < len(previous) && path[common] == previous[common] {
			common++
		}
		minimize(common)
		node := root
		if len(pending) > 0 {
			node = pending[len(pending)-1].child
		}
		for i := common; i < len(path); i++ {
			child := &ruleNode{id: nextID}
			nextID++
			node.edges = append(node.edges, ruleEdge{path[i], child})
			pending = append(pending, pendingRuleEdge{node, len(node.edges) - 1, child})
			if i == len(path)-2 {
				node.final = true
			}
			node = child
		}
		node.final = true
		previous = path
	}
	minimize(0)
	nodes = append([]*ruleNode{root}, nodes...)
	ids := map[*ruleNode]int{}
	for i, node := range nodes {
		ids[node] = i
	}
	result := make([]GraphNode, len(nodes))
	for i, node := range nodes {
		result[i].Arcs = []GraphArc{}
		for _, edge := range node.edges {
			kind, value := classifyGraphArc(edge.label)
			result[i].Arcs = append(result[i].Arcs, GraphArc{kind, value, ids[edge.child]})
		}
	}
	for _, node := range result {
		for _, arc := range node.Arcs {
			if arc.Kind == "<rules>" {
				target := &result[arc.Next]
				if target.Actions == nil {
					target.Actions = []string{}
					for _, a := range target.Arcs {
						target.Actions = append(target.Actions, a.Value)
					}
					sort.Strings(target.Actions)
					target.Arcs = nil
				}
			}
		}
	}
	return result, nil
}
func classifyGraphArc(label string) (string, string) {
	if strings.HasPrefix(label, "##") {
		return "<rules>", label[1:]
	}
	if len([]rune(label)) > 1 {
		for prefix, kind := range map[string]string{"@": "<re_morph>", "$": "<morph>", "&": "<re_mmorph>", "~": "<re_value>", ">": "<lemmas>", "%": "<phonet>", "*": "<meta>", "/": "<tags>"} {
			if strings.HasPrefix(label, prefix) {
				return kind, label[len(prefix):]
			}
		}
	}
	return "literal", label
}
