// Package rules parses the pinned Grammalecte rule language into a source model.
// Parsing does not execute rules or certify an executable grammar engine.
package rules

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// The original French rules are by Olivier R., licensed under GPLv3 or later.
//
//go:embed data/fr.grx.gz
var frenchRules []byte

// FrenchSource returns a fresh copy of the embedded upstream rule source.
func FrenchSource() (string, error) {
	r, e := gzip.NewReader(bytes.NewReader(frenchRules))
	if e != nil {
		return "", e
	}
	defer r.Close()
	b, e := io.ReadAll(r)
	return string(b), e
}

// Definition retains insertion order because macro expansion is ordered.
type Definition struct {
	Name, Value string
	Line        int
}

// SourceLine retains the original location of a directive or expression.
type SourceLine struct {
	Line int
	Text string
}

// Action is an unevaluated rule action. Expressions remain source text.
type Action struct {
	Condition, Kind, Value, Message, URL string
	Group                                int
}

// RegexRule is a paragraph or sentence trigger with its ordered actions.
type RegexRule struct {
	ID, LineID, Option, Pattern string
	Priority, Line              int
	Paragraph                   bool
	UppercaseSuggestions        bool
	Actions                     []Action
}

// GraphGroup preserves the complete token-pattern and action source for a group.
type GraphGroup struct {
	Name   string
	Line   int
	Source []SourceLine
}

// Graph retains graph identity and ordered groups before token expansion.
type Graph struct {
	Name, Code string
	Line       int
	Groups     []GraphGroup
}

// Step preserves the interleaving of regex passes and graph calls.
type Step struct {
	RegexIndex, GraphIndex int
	Paragraph              bool
}

// Document is the lossless source front end used by subsequent compilation stages.
type Document struct {
	Definitions []Definition
	Declensions []Definition
	Directives  []SourceLine
	Tests       []SourceLine
	RegexRules  []RegexRule
	Graphs      []Graph
	Steps       []Step
	Defaults    map[string]map[string]bool
	Colors      map[string]map[string][]int
	Labels      map[string]map[string]any
}

var ruleHeader = regexp.MustCompile(`^__([\[<])([isu])([\]>])(?:/([a-zA-Z0-9]+))?\(([^)]+)\)(?:!([0-9]))?__\s*`)
var actionMarker = regexp.MustCompile(`([-~=>])([0-9]*)>>`)
var graphHeader = regexp.MustCompile(`^@@@@GRAPH: *([\pL\pN_]+) *\| *([\pL\pN_]+) *(?:_)?$`)
var jsGroups = regexp.MustCompile(`@@\S+`)
var jsPattern = regexp.MustCompile(`<js>.+</js>i?`)
var messageURL = regexp.MustCompile(`[|] *(https?://.*)`)
var optionPrefixes = []string{"OPTGROUP/", "OPTSOFTWARE:", "OPT/", "COLOR/", "OPTCOLORTHEME:", "OPTCOLOR/", "OPTLANG/", "OPTDEFAULTUILANG:", "OPTLABEL/", "OPTPRIORITY/"}

// Parse reads rule source without executing conditions, actions or graph code.
// It rejects unrecognized statements with their source line rather than dropping them.
func Parse(source string) (*Document, error) {
	d := &Document{Defaults: map[string]map[string]bool{}}
	var raw []SourceLine
	graph := -1
	for i, line := range strings.Split(source, "\n") {
		n := i + 1
		trim := strings.TrimSpace(line)
		fail := func(s string) (*Document, error) { return nil, fmt.Errorf("line %d: %s", n, s) }
		if strings.HasPrefix(line, "#END") {
			break
		}
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "    ##") {
			continue
		}
		if strings.HasPrefix(line, "DEF:") || strings.HasPrefix(line, "DECL:") {
			name, value, ok := strings.Cut(strings.TrimSpace(line[strings.IndexByte(line, ':')+1:]), " ")
			value = strings.TrimSpace(value)
			if !ok || name == "" || value == "" {
				return fail("invalid definition")
			}
			entry := Definition{name, value, n}
			if strings.HasPrefix(line, "DEF:") {
				d.Definitions = append(d.Definitions, entry)
			} else {
				if !strings.HasPrefix(name, "+") {
					return fail("invalid declension")
				}
				d.Declensions = append(d.Declensions, entry)
			}
			continue
		}
		if strings.HasPrefix(line, "TEST:") {
			d.Tests = append(d.Tests, SourceLine{n, strings.TrimSpace(line[5:])})
			continue
		}
		if strings.HasPrefix(line, "TODO:") || strings.HasPrefix(line, "!!") {
			continue
		}
		directive := false
		for _, prefix := range optionPrefixes {
			if strings.HasPrefix(line, prefix) {
				directive = true
				break
			}
		}
		if directive {
			d.Directives = append(d.Directives, SourceLine{n, line})
			continue
		}
		if strings.HasPrefix(line, "@@@@GRAPH:") {
			m := graphHeader.FindStringSubmatch(trim)
			if m == nil {
				return fail("invalid graph declaration")
			}
			for _, g := range d.Graphs {
				if g.Name == m[1] || g.Code == m[2] {
					return fail("duplicate graph identity")
				}
			}
			d.Graphs = append(d.Graphs, Graph{Name: m[1], Code: m[2], Line: n})
			graph = len(d.Graphs) - 1
			raw = append(raw, SourceLine{n, fmt.Sprintf("@@@@%d", graph)})
			continue
		}
		if strings.HasPrefix(line, "@@@@END_GRAPH") || strings.HasPrefix(line, "@@@@ENDGRAPH") {
			if graph < 0 {
				return fail("graph end without graph")
			}
			graph = -1
			continue
		}
		if trim == "@@@@" {
			continue
		}
		if graph >= 0 {
			g := &d.Graphs[graph]
			if strings.HasPrefix(line, "__") && strings.HasSuffix(trim, "__") {
				name := strings.TrimSuffix(strings.TrimPrefix(trim, "__"), "__")
				g.Groups = append(g.Groups, GraphGroup{Name: name, Line: n})
				continue
			}
			if trim != "" && len(g.Groups) == 0 {
				return fail("graph content without a group")
			}
			if len(g.Groups) > 0 {
				j := len(g.Groups) - 1
				g.Groups[j].Source = append(g.Groups[j].Source, SourceLine{n, line})
			}
			continue
		}
		if trim == "" {
			continue
		}
		if strings.HasPrefix(line, "    ") {
			if len(raw) == 0 {
				return fail("continuation without a rule")
			}
			raw[len(raw)-1].Text += " " + trim
			continue
		}
		if trim != "[++]" && !strings.HasPrefix(trim, "__") {
			return fail("unknown statement")
		}
		raw = append(raw, SourceLine{n, trim})
	}
	if graph >= 0 {
		return nil, fmt.Errorf("line %d: unterminated graph", d.Graphs[graph].Line)
	}
	priorities := map[string]int{}
	var contexts []string
	for _, directive := range d.Directives {
		key, value, _ := strings.Cut(directive.Text, ":")
		fields := strings.Fields(value)
		switch {
		case key == "OPTSOFTWARE":
			contexts = fields
			for _, c := range contexts {
				d.Defaults[c] = map[string]bool{}
			}
		case strings.HasPrefix(key, "OPT/"):
			if len(fields) != len(contexts) || len(contexts) == 0 {
				return nil, fmt.Errorf("line %d: option context count mismatch", directive.Line)
			}
			for i, v := range fields {
				if v != "True" && v != "False" {
					return nil, fmt.Errorf("line %d: invalid boolean", directive.Line)
				}
				d.Defaults[contexts[i]][strings.TrimPrefix(key, "OPT/")] = v == "True"
			}
		case strings.HasPrefix(key, "OPTPRIORITY/"):
			n, e := strconv.Atoi(strings.TrimSpace(value))
			if e != nil || n < 0 || n > 9 {
				return nil, fmt.Errorf("line %d: invalid priority", directive.Line)
			}
			priorities[strings.TrimPrefix(key, "OPTPRIORITY/")] = n
		}
	}
	paragraph := true
	seen := map[string]bool{}
	for _, line := range raw {
		if line.Text == "[++]" {
			paragraph = false
			continue
		}
		if strings.HasPrefix(line.Text, "@@@@") {
			if paragraph {
				return nil, fmt.Errorf("line %d: graph call in paragraph pass", line.Line)
			}
			index, _ := strconv.Atoi(line.Text[4:])
			d.Steps = append(d.Steps, Step{RegexIndex: -1, GraphIndex: index, Paragraph: false})
			continue
		}
		r, e := parseRegexRule(line, paragraph, d.Definitions, priorities)
		if e != nil {
			return nil, e
		}
		if seen[r.ID] {
			return nil, fmt.Errorf("line %d: duplicate rule %s", line.Line, r.ID)
		}
		seen[r.ID] = true
		d.Steps = append(d.Steps, Step{RegexIndex: len(d.RegexRules), GraphIndex: -1, Paragraph: paragraph})
		d.RegexRules = append(d.RegexRules, r)
	}
	if err := parseOptionMetadata(d); err != nil {
		return nil, err
	}
	return d, nil
}

func parseRegexRule(line SourceLine, paragraph bool, definitions []Definition, priorities map[string]int) (RegexRule, error) {
	r := RegexRule{Line: line.Line, Paragraph: paragraph, Priority: 4}
	fail := func(message string) (RegexRule, error) { return r, fmt.Errorf("line %d: %s", line.Line, message) }
	indexes := ruleHeader.FindStringSubmatchIndex(line.Text)
	if indexes == nil {
		return fail("invalid regex rule header")
	}
	m := ruleHeader.FindStringSubmatch(line.Text)
	r.ID = m[5]
	r.UppercaseSuggestions = m[2] == "i"
	r.Option = m[4]
	r.LineID = "#" + strconv.Itoa(line.Line) + "s"
	if paragraph {
		r.LineID = "#" + strconv.Itoa(line.Line) + "p"
	}
	if n, ok := priorities[r.Option]; ok {
		r.Priority = n
	}
	if m[6] != "" {
		r.Priority, _ = strconv.Atoi(m[6])
	}
	pattern, body, ok := strings.Cut(line.Text[indexes[1]:], " <<-")
	if !ok {
		return fail("missing action")
	}
	pattern = strings.TrimSpace(pattern)
	if loc := jsGroups.FindStringIndex(pattern); loc != nil {
		pattern = strings.TrimSpace(pattern[:loc[0]])
	}
	if loc := jsPattern.FindStringIndex(pattern); loc != nil {
		pattern = strings.TrimSpace(pattern[:loc[0]])
	}
	pattern = unquote(pattern)
	for _, def := range definitions {
		pattern = strings.ReplaceAll(pattern, "{"+def.Name+"}", def.Value)
	}
	if m[1] == "[" && !strings.HasPrefix(pattern, "^") && !startsWithAny(pattern, "’',") {
		pattern = `(?<![\w.,–-])` + pattern
	}
	if m[3] == "]" && !strings.HasSuffix(pattern, "$") && !endsWithAny(pattern, "’',") {
		pattern += `(?![\w–-])`
	}
	switch m[2] {
	case "i":
		if !strings.HasPrefix(pattern, "(?i)") {
			pattern = "(?i)" + pattern
		}
	case "s":
		pattern = strings.ReplaceAll(pattern, "(?i)", "")
	case "u":
		pattern = uppercasePattern(strings.ReplaceAll(pattern, "(?i)", ""))
	}
	r.Pattern = pattern
	for _, source := range strings.Split(body, " <<- ") {
		location := actionMarker.FindStringSubmatchIndex(source)
		if location == nil {
			return fail("invalid action")
		}
		match := actionMarker.FindStringSubmatch(source)
		a := Action{Condition: strings.TrimSpace(source[:location[0]]), Kind: match[1], Value: strings.TrimSpace(source[location[1]:])}
		a.Group, _ = strconv.Atoi(match[2])
		if a.Kind == "-" {
			value, message, found := strings.Cut(a.Value, " && ")
			if !found {
				return fail("missing diagnostic message")
			}
			a.Value = strings.TrimSpace(value)
			a.Message = strings.TrimSpace(message)
			if location := messageURL.FindStringSubmatchIndex(a.Message); location != nil {
				a.URL = strings.TrimSpace(a.Message[location[2]:location[3]])
				a.Message = strings.TrimSpace(a.Message[:location[0]])
			}
		}
		if a.Kind != ">" && a.Value == "" {
			return fail("empty action")
		}
		a.Value = unquote(a.Value)
		r.Actions = append(r.Actions, a)
	}
	return r, nil
}
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
func startsWithAny(s, chars string) bool {
	for _, r := range chars {
		if strings.HasPrefix(s, string(r)) {
			return true
		}
	}
	return false
}
func endsWithAny(s, chars string) bool {
	for _, r := range chars {
		if strings.HasSuffix(s, string(r)) {
			return true
		}
	}
	return false
}

// uppercasePattern mirrors the upstream selective case-expansion mode.
func uppercasePattern(input string) string {
	rs := []rune(input)
	var out strings.Builder
	state := 0
	for i, c := range rs {
		if c == '[' {
			state = 1
		}
		if state == 1 && c == ']' {
			state = 0
		}
		if c == '<' && i > 3 && string(rs[i-3:i]) == "(?P" {
			state = 2
		}
		if state == 2 && c == '>' {
			state = 0
		}
		if c == '?' && i > 0 && rs[i-1] == '(' && (i+1 >= len(rs) || rs[i+1] != ':') {
			state = 5
		}
		if state == 5 && c == ')' {
			state = 0
		}
		if unicode.IsLetter(c) && unicode.IsLower(c) && state == 0 {
			out.WriteRune('[')
			out.WriteRune(unicode.ToUpper(c))
			out.WriteRune(c)
			out.WriteRune(']')
		} else if unicode.IsLetter(c) && unicode.IsLower(c) && state == 1 && (i+1 >= len(rs) || rs[i+1] != '-') {
			if i >= 2 && rs[i-1] == '-' && unicode.IsLower(rs[i-2]) {
				out.WriteRune(c)
				out.WriteRune(unicode.ToUpper(rs[i-2]))
				out.WriteRune('-')
				out.WriteRune(unicode.ToUpper(c))
			} else {
				out.WriteRune(unicode.ToUpper(c))
				out.WriteRune(c)
			}
		} else {
			out.WriteRune(c)
		}
		if c == '\\' {
			state = 4
		} else if state == 4 {
			state = 0
		}
	}
	return out.String()
}
