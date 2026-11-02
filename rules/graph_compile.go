// The graph rule language and compilation semantics are adapted from Olivier R.'s
// compile_rules_graph.py, distributed by Grammalecte under GPLv3 or later.
package rules

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/benoit-pereira-da-silva/gogrammalecte/internal/pyregex"
)

// GraphAction contains an executable action's metadata and its unevaluated DSL
// expressions. Positive token references are remapped from selected groups to
// token positions; negative references retain their end-relative meaning.
type GraphAction struct {
	ID                string `json:"id"`
	Line              int    `json:"line"`
	Option            string `json:"option,omitempty"`
	Condition         string `json:"condition,omitempty"`
	Kind              string `json:"kind"`
	Value             string `json:"value"`
	ValueExpression   bool   `json:"value_expression,omitempty"`
	StartToken        int    `json:"start_token,omitempty"`
	EndToken          int    `json:"end_token,omitempty"`
	StartLimit        string `json:"start_limit,omitempty"`
	EndLimit          string `json:"end_limit,omitempty"`
	CaseSensitive     bool   `json:"case_sensitive,omitempty"`
	Priority          int    `json:"priority,omitempty"`
	Message           string `json:"message,omitempty"`
	MessageExpression bool   `json:"message_expression,omitempty"`
	URL               string `json:"url,omitempty"`
}

// CompiledGraph is a minimized token graph with ordered actions. Node zero is
// the root. RuleCount counts expanded paths before graph minimization. Lemmas
// maps each referenced lemma to the first source line at which it occurred.
type CompiledGraph struct {
	Name      string         `json:"name"`
	Code      string         `json:"code"`
	Nodes     []GraphNode    `json:"nodes"`
	Actions   []GraphAction  `json:"actions"`
	RuleCount int            `json:"rule_count"`
	Lemmas    map[string]int `json:"lemmas"`
}

var graphActionMarker = regexp.MustCompile(`([-=~/!>&])(-?[0-9]+\.?|)(:\.?-?[0-9]+|)(:|)>>`)
var graphReference = regexp.MustCompile(`\\([0-9]+)`)
var graphURL = regexp.MustCompile(`[|] *(https?://.*)`)

// CompileGraphs compiles every token graph without generating foreign-language
// code. Conditions and computed action values remain DSL expressions for an
// evaluator to execute. Compilation is deterministic and does not mutate doc.
func CompileGraphs(doc *Document) ([]CompiledGraph, error) {
	if doc == nil {
		return nil, fmt.Errorf("nil rule document")
	}
	priorities := map[string]int{}
	for _, line := range doc.Directives {
		key, value, ok := strings.Cut(line.Text, ":")
		if ok && strings.HasPrefix(key, "OPTPRIORITY/") {
			n, e := strconv.Atoi(strings.TrimSpace(value))
			if e != nil || n < 0 || n > 9 {
				return nil, fmt.Errorf("line %d: invalid option priority", line.Line)
			}
			priorities[strings.TrimPrefix(key, "OPTPRIORITY/")] = n
		}
	}
	compiled := make([]CompiledGraph, 0, len(doc.Graphs))
	names, codes, groups := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, source := range doc.Graphs {
		if source.Name == "" || source.Code == "" || names[source.Name] || codes[source.Code] {
			return nil, fmt.Errorf("line %d: empty or duplicate graph identity", source.Line)
		}
		names[source.Name], codes[source.Code] = true, true
		graph := CompiledGraph{Name: source.Name, Code: source.Code, Actions: []GraphAction{}, Lemmas: map[string]int{}}
		paths := [][]string{}
		actionIndexes := map[string]int{}
		for _, group := range source.Groups {
			name, priority, e := graphGroupName(group)
			if e != nil {
				return nil, e
			}
			if groups[name] {
				return nil, fmt.Errorf("line %d: duplicate graph rule group %q", group.Line, name)
			}
			groups[name] = true
			anti := map[string]bool{}
			block := 1
			patterns, actions := []SourceLine{}, []SourceLine{}
			continuation := false
			flush := func() error {
				if len(patterns) == 0 {
					if len(actions) > 0 {
						return fmt.Errorf("line %d: graph actions without a pattern", actions[0].Line)
					}
					return nil
				}
				if len(actions) == 0 || continuation {
					return fmt.Errorf("line %d: graph pattern has no complete action", patterns[0].Line)
				}
				for _, pattern := range patterns {
					text := pattern.Text
					antipattern := strings.HasPrefix(text, "!!") && strings.HasSuffix(text, "¡¡")
					if antipattern {
						text = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(text, "!!"), "¡¡"))
					}
					expansions, e := ExpandGraphTokens(text, doc.Definitions, doc.Declensions)
					if e != nil {
						return fmt.Errorf("line %d: %w", pattern.Line, e)
					}
					for _, expanded := range expansions {
						key, _ := json.Marshal(expanded.Tokens)
						if antipattern {
							anti[string(key)] = true
							continue
						}
						if anti[string(key)] {
							continue
						}
						tokens := append([]string{}, expanded.Tokens...)
						positions := map[int]int{}
						groupIndex := 0
						for i, token := range tokens {
							if strings.HasPrefix(token, "(") && strings.HasSuffix(token, ")") {
								tokens[i] = token[1 : len(token)-1]
								groupIndex++
								positions[groupIndex] = i + 1
							}
							if strings.HasPrefix(token, ">") && token != ">" {
								lemma := token[1:]
								if _, ok := graph.Lemmas[lemma]; !ok {
									graph.Lemmas[lemma] = pattern.Line
								}
							}
						}
						for index, sourceAction := range actions {
							action, e := parseGraphAction(sourceAction, priority, positions, priorities)
							if e != nil {
								return e
							}
							baseID := fmt.Sprintf("%s__%s__b%d_a%d", source.Code, name, block, index+1)
							for variant := 1; ; variant++ {
								id := baseID + "_" + strconv.Itoa(variant)
								action.ID = id
								if existing, ok := actionIndexes[id]; ok {
									if graph.Actions[existing] != action {
										continue
									}
								} else {
									actionIndexes[id] = len(graph.Actions)
									graph.Actions = append(graph.Actions, action)
								}
								path := append([]string{}, tokens...)
								paths = append(paths, append(path, "##"+strconv.Itoa(pattern.Line), id))
								break
							}
						}
					}
				}
				patterns, actions = nil, nil
				block++
				return nil
			}
			for _, line := range group.Source {
				text := strings.TrimRightFunc(line.Text, unicode.IsSpace)
				if strings.ContainsRune(text, '\t') {
					return nil, fmt.Errorf("line %d: tabs are not allowed in graph rules", line.Line)
				}
				switch {
				case len(text) > 4 && strings.HasPrefix(text, "    ") && !unicode.IsSpace([]rune(text[4:])[0]):
					patterns = append(patterns, SourceLine{line.Line, strings.TrimSpace(text)})
				case strings.HasPrefix(text, "        ||"):
					if len(patterns) == 0 {
						return nil, fmt.Errorf("line %d: token continuation without a pattern", line.Line)
					}
					patterns[len(patterns)-1].Text += " " + strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), "||"))
				case strings.HasPrefix(text, "        <<- "):
					if continuation {
						return nil, fmt.Errorf("line %d: unfinished graph action", line.Line)
					}
					action := strings.TrimSpace(text[12:])
					actions = append(actions, SourceLine{line.Line, action})
					continuation = !graphActionMarker.MatchString(action)
				case strings.HasPrefix(text, "        && "):
					if len(actions) == 0 {
						return nil, fmt.Errorf("line %d: action message without an action", line.Line)
					}
					actions[len(actions)-1].Text += text
				case strings.HasPrefix(text, "        ") && continuation:
					actions[len(actions)-1].Text += " " + strings.TrimSpace(text)
					continuation = !graphActionMarker.MatchString(text)
				case strings.TrimSpace(text) == "":
					if e := flush(); e != nil {
						return nil, e
					}
				default:
					return nil, fmt.Errorf("line %d: unknown graph statement %q", line.Line, text)
				}
			}
			if e := flush(); e != nil {
				return nil, e
			}
		}
		var e error
		graph.Nodes, e = MinimizeGraphPaths(paths)
		if e != nil {
			return nil, fmt.Errorf("graph %s: %w", source.Name, e)
		}
		if e = validateGraphPatterns(graph.Nodes); e != nil {
			return nil, fmt.Errorf("graph %s: %w", source.Name, e)
		}
		graph.RuleCount = len(paths)
		compiled = append(compiled, graph)
	}
	return compiled, nil
}

func graphGroupName(group GraphGroup) (string, int, error) {
	name := group.Name
	priority := -1
	if i := strings.LastIndexByte(name, '!'); i >= 0 {
		n, e := strconv.Atoi(name[i+1:])
		if e != nil || n < 0 || n > 9 {
			return "", 0, fmt.Errorf("line %d: invalid graph group priority", group.Line)
		}
		priority = n
		name = name[:i]
	}
	if name == "" {
		return "", 0, fmt.Errorf("line %d: empty graph group name", group.Line)
	}
	for _, r := range name {
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return "", 0, fmt.Errorf("line %d: invalid graph group name %q", group.Line, name)
		}
	}
	return name, priority, nil
}

func remapGraphReferences(text string, positions map[int]int) string {
	return graphReference.ReplaceAllStringFunc(text, func(match string) string {
		n, _ := strconv.Atoi(match[1:])
		if position, ok := positions[n]; ok {
			return "\\" + strconv.Itoa(position)
		}
		return match
	})
}

func parseGraphAction(source SourceLine, priority int, positions map[int]int, priorities map[string]int) (GraphAction, error) {
	action := GraphAction{Line: source.Line}
	text := strings.TrimSpace(source.Text)
	fail := func(message string) (GraphAction, error) {
		return GraphAction{}, fmt.Errorf("line %d: %s", source.Line, message)
	}
	if strings.HasPrefix(text, "/") {
		if end := strings.IndexByte(text[1:], '/'); end >= 0 {
			option := text[1 : end+1]
			valid := option != ""
			for _, r := range option {
				if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
					valid = false
				}
			}
			if valid {
				action.Option = option
				text = strings.TrimSpace(text[end+2:])
			}
		}
	}
	if priority < 0 {
		var ok bool
		priority, ok = priorities[action.Option]
		if !ok {
			priority = 4
		}
	}
	loc := graphActionMarker.FindStringSubmatchIndex(text)
	if loc == nil {
		return fail("graph action marker is missing")
	}
	get := func(group int) string { return text[loc[group*2]:loc[group*2+1]] }
	action.Condition = strings.TrimPrefix(remapGraphReferences(strings.TrimSpace(text[:loc[0]]), positions), "=")
	action.Kind = get(1)
	startText, endText, casing := get(2), get(3), get(4)
	value := remapGraphReferences(strings.TrimSpace(text[loc[1]:]), positions)
	start, end := 1, 0
	startLimit, endLimit := "<", ">"
	if startText != "" {
		if action.Kind != "-" && (strings.HasSuffix(startText, ".") || strings.HasPrefix(endText, ":.")) {
			return fail("token boundary selection is only valid for error actions")
		}
		if strings.HasSuffix(startText, ".") {
			startLimit = ">"
		}
		var e error
		start, e = strconv.Atoi(strings.TrimSuffix(startText, "."))
		if e != nil {
			return fail("invalid graph action start token")
		}
		end = start
		if endText != "" {
			if strings.HasPrefix(endText, ":.") {
				endLimit = "<"
			}
			end, e = strconv.Atoi(strings.TrimLeft(endText, ":."))
			if e != nil {
				return fail("invalid graph action end token")
			}
		}
		if mapped, ok := positions[start]; ok {
			start = mapped
		}
		if end != 0 {
			if mapped, ok := positions[end]; ok {
				end = mapped
			}
		}
	}
	if start < 0 {
		start++
	}
	if end < 0 {
		end++
	}
	if action.Kind == ">" {
		return action, nil
	}
	if action.Kind == "-" {
		suggestion, message, ok := strings.Cut(value, " && ")
		if !ok {
			return fail("error action message is missing")
		}
		value = strings.TrimSpace(suggestion)
		action.Message = strings.TrimSpace(message)
		if found := graphURL.FindStringSubmatchIndex(action.Message); found != nil {
			action.URL = strings.TrimSpace(action.Message[found[2]:found[3]])
			action.Message = strings.TrimSpace(action.Message[:found[0]])
		}
		if action.Message == "" {
			return fail("error action message is empty")
		}
		action.MessageExpression = strings.HasPrefix(action.Message, "=")
		if action.MessageExpression {
			action.Message = strings.TrimPrefix(action.Message, "=")
		}
		action.StartLimit, action.EndLimit = startLimit, endLimit
		action.Priority = priority
	}
	if value == "" && action.Kind != "!" {
		return fail("graph action value is empty")
	}
	switch action.Kind {
	case "-", "~":
		action.CaseSensitive = casing == ""
		action.ValueExpression = strings.HasPrefix(value, "=")
		if action.ValueExpression {
			value = value[1:]
		} else if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) && len(value) > 1 {
			value = value[1 : len(value)-1]
		}
		action.StartToken, action.EndToken = start, end
	case "!", "/", "&":
		action.StartToken, action.EndToken = start, end
	case "=":
		action.ValueExpression = true
		value = strings.TrimPrefix(value, "=")
	default:
		return fail("unknown graph action kind")
	}
	action.Value = value
	return action, nil
}

func validateGraphPatterns(nodes []GraphNode) error {
	seen := map[string]bool{}
	for _, node := range nodes {
		for _, arc := range node.Arcs {
			if arc.Kind != "<re_value>" && arc.Kind != "<re_morph>" && arc.Kind != "<re_mmorph>" {
				continue
			}
			if seen[arc.Value] {
				continue
			}
			seen[arc.Value] = true
			parts := strings.SplitN(arc.Value, "¬", 2)
			for i, part := range parts {
				if part == "" || i == 1 && part == "*" {
					continue
				}
				if _, err := pyregex.Compile(part); err != nil {
					return fmt.Errorf("invalid %s graph pattern: %w", arc.Kind, err)
				}
			}
		}
	}
	return nil
}
