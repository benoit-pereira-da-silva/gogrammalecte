// Package grammar implements Grammalecte's paragraph, sentence and token-graph
// analysis. Adapted from gc_engine.py by Olivier R., GPLv3 or later.
package grammar

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell"
	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/strtransform"
	"github.com/benoit-pereira-da-silva/gogrammalecte/internal/pyregex"
	"github.com/benoit-pereira-da-silva/gogrammalecte/internal/ruleexpr"
	"github.com/benoit-pereira-da-silva/gogrammalecte/rules"
	textutil "github.com/benoit-pereira-da-silva/gogrammalecte/text"
)

// EvalContext supplies the source DSL with the current regex match or graph
// token span. Text offsets and token references count Unicode code points.
type EvalContext struct {
	Context                context.Context
	Helpers                *HelperContext
	Text, OriginalText     string
	Tokens                 []*Token
	TokenOffset, LastToken int
	Match                  *pyregex.Match
	TargetGroup            int
	Country, AppContext    string
	CondMemo               any
	Options                map[string]bool
}

// Error preserves the reference diagnostic JSON field names and rune offsets.
type Error struct {
	Start       int      `json:"nStart"`
	End         int      `json:"nEnd"`
	LineID      string   `json:"sLineId"`
	RuleID      string   `json:"sRuleId"`
	Type        string   `json:"sType"`
	Color       []int    `json:"aColor"`
	Suggestions []string `json:"aSuggestions"`
	Message     string   `json:"sMessage"`
	URL         string   `json:"URL"`
	Underlined  string   `json:"sUnderlined,omitempty"`
	Before      string   `json:"sBefore,omitempty"`
	After       string   `json:"sAfter,omitempty"`
	withContext bool
}

// ParseOptions overrides options for one analysis. Nil Options uses the
// embedded Python defaults; a supplied map overrides individual defaults.
type ParseOptions struct {
	Country      string
	AppContext   string
	Options      map[string]bool
	IgnoredRules map[string]bool
	Context      bool
}

// Engine's compiled rules and checker are immutable and safe to share. Every
// Check call allocates independent token, tag, disambiguation and error state.
type Engine struct {
	document *rules.Document
	checker  Lexicon
	regex    []*pyregex.Regexp
	graphs   []rules.CompiledGraph
	actions  []map[string]rules.GraphAction
	patterns map[string]*pyregex.Regexp
	indexes  [][]graphIndex
}

// New compiles an independent source document. A nil checker selects the
// embedded French spelling dictionary. Expressions are evaluated only when a
// matching rule reaches them, and unsupported expressions return an error.
func New(document *rules.Document, checker Lexicon) (*Engine, error) {
	if document == nil {
		return nil, fmt.Errorf("nil rule document")
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}
	var copy rules.Document
	if err = json.Unmarshal(raw, &copy); err != nil {
		return nil, err
	}
	document = &copy
	graphs, err := rules.CompileGraphs(document)
	if err != nil {
		return nil, err
	}
	return newEngine(document, graphs, checker)
}
func newEngine(document *rules.Document, graphs []rules.CompiledGraph, checker Lexicon) (*Engine, error) {
	if checker == nil {
		var err error
		checker, err = graphspell.Default()
		if err != nil {
			return nil, err
		}
	}
	engine := &Engine{document: document, checker: checker, graphs: graphs, patterns: map[string]*pyregex.Regexp{}}
	for _, r := range document.RegexRules {
		pattern, err := pyregex.Compile(r.Pattern)
		if err != nil {
			return nil, fmt.Errorf("rule %s: %w", r.ID, err)
		}
		engine.regex = append(engine.regex, pattern)
	}
	for _, graph := range graphs {
		engine.indexes = append(engine.indexes, indexGraph(graph))
		actions := map[string]rules.GraphAction{}
		for _, action := range graph.Actions {
			actions[action.ID] = action
		}
		engine.actions = append(engine.actions, actions)
		for _, node := range graph.Nodes {
			for _, arc := range node.Arcs {
				if arc.Kind != "<re_value>" && arc.Kind != "<re_morph>" && arc.Kind != "<re_mmorph>" {
					continue
				}
				for i, pattern := range strings.SplitN(arc.Value, "¬", 2) {
					if pattern == "" || i == 1 && pattern == "*" {
						continue
					}
					if _, ok := engine.patterns[pattern]; ok {
						continue
					}
					compiled, err := pyregex.Compile(pattern)
					if err != nil {
						return nil, err
					}
					engine.patterns[pattern] = compiled
				}
			}
		}
	}
	return engine, nil
}

var defaultEngine struct {
	sync.Once
	engine *Engine
	err    error
}

// Default returns the complete embedded French grammar without file access.
func Default() (*Engine, error) {
	defaultEngine.Do(func() {
		source, err := rules.FrenchSource()
		if err != nil {
			defaultEngine.err = err
			return
		}
		doc, err := rules.Parse(source)
		if err != nil {
			defaultEngine.err = err
			return
		}
		graphs, err := rules.FrenchGraphs()
		if err != nil {
			defaultEngine.err = err
			return
		}
		defaultEngine.engine, defaultEngine.err = newEngine(doc, graphs, nil)
	})
	return defaultEngine.engine, defaultEngine.err
}

// Check analyzes one paragraph and returns diagnostics in reference insertion
// order. A higher-priority diagnostic replaces one at the same starting offset.
func (e *Engine) Check(ctx context.Context, input string, options ParseOptions) ([]Error, error) {
	errors, _, err := e.analyze(ctx, input, options, false)
	return errors, err
}

// Analysis retains paragraph diagnostics and the complete analyzed sentences.
type Analysis struct {
	ParagraphErrors []Error            `json:"lParagraphErrors"`
	Sentences       []SentenceAnalysis `json:"lSentences"`
}

// SentenceAnalysis includes the original text and the token snapshots retained
// before graph removal, together with their final disambiguation and labels.
type SentenceAnalysis struct {
	Start    int         `json:"nStart"`
	End      int         `json:"nEnd"`
	Sentence string      `json:"sSentence"`
	Tokens   []TokenInfo `json:"lTokens"`
	Errors   []Error     `json:"lGrammarErrors"`
}

// Analyze returns the reference's full-information analysis, including tokens
// that were merged or removed while applying grammar graphs.
func (e *Engine) Analyze(ctx context.Context, input string, options ParseOptions) (*Analysis, error) {
	_, result, err := e.analyze(ctx, input, options, true)
	return result, err
}

func (e *Engine) analyze(ctx context.Context, input string, options ParseOptions, full bool) ([]Error, *Analysis, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if !utf8.ValidString(input) {
		return nil, nil, fmt.Errorf("input is not valid UTF-8")
	}
	app := options.AppContext
	if app == "" {
		app = "Python"
	}
	country := options.Country
	if country == "" {
		country = "FR"
	}
	settings := map[string]bool{}
	defaults := e.document.Defaults[app]
	if defaults == nil {
		defaults = e.document.Defaults["Python"]
	}
	for key, value := range defaults {
		settings[key] = value
	}
	for key, value := range options.Options {
		settings[key] = value
	}
	state := &analysis{engine: e, ctx: ctx, original: input, text: input, options: options, errors: map[int]Error{}, priorities: map[int]int{}, helper: &HelperContext{Context: ctx, Checker: e.checker, TokenPositions: map[int]*Token{}, Tags: map[string][2]int{}, Options: settings}, country: country, app: app}
	var result *Analysis
	if full {
		result = &Analysis{ParagraphErrors: []Error{}, Sentences: []SentenceAnalysis{}}
		state.sentenceErrors = map[int]Error{}
	}
	if err := state.parseText(true); err != nil {
		return nil, nil, err
	}
	if full {
		result.ParagraphErrors = state.errorList()
		state.sentenceErrors = map[int]Error{}
		state.sentenceOrder = nil
	}
	clean := cleanText(state.text)
	runes, original := []rune(clean), []rune(input)
	for _, span := range textutil.SentenceBoundaries(clean) {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		if span.End-span.Start <= 2 || span.End-span.Start >= 3000 {
			continue
		}
		state.offset = span.Start
		state.sentence = string(runes[span.Start:span.End])
		state.sentence0 = sliceRunes(original, span.Start, span.End)
		if err := state.tokenize(false); err != nil {
			return nil, nil, err
		}
		var originalTokens []*Token
		if full {
			originalTokens = append([]*Token{}, state.tokens...)
		}
		if err := state.parseText(false); err != nil {
			return nil, nil, err
		}
		if full {
			sentence := SentenceAnalysis{Start: span.Start, End: span.End, Sentence: state.sentence0, Tokens: make([]TokenInfo, 0, len(originalTokens)), Errors: []Error{}}
			for _, token := range originalTokens {
				sentence.Tokens = append(sentence.Tokens, DescribeToken(token, state.engine.checker))
			}
			for _, position := range state.sentenceOrder {
				sentence.Errors = append(sentence.Errors, state.sentenceErrors[position])
			}
			result.Sentences = append(result.Sentences, sentence)
			state.sentenceErrors = map[int]Error{}
			state.sentenceOrder = nil
		}
	}
	out := make([]Error, 0, len(state.errors))
	for _, position := range state.order {
		if diagnostic, ok := state.errors[position]; ok {
			out = append(out, diagnostic)
		}
	}
	return out, result, nil
}

type analysis struct {
	engine                              *Engine
	ctx                                 context.Context
	original, text, sentence, sentence0 string
	offset                              int
	options                             ParseOptions
	helper                              *HelperContext
	tokens                              []*Token
	country, app                        string
	errors                              map[int]Error
	priorities                          map[int]int
	order                               []int
	sentenceErrors                      map[int]Error
	sentenceOrder                       []int
}

func (a *analysis) evalContext(text, original string, match *pyregex.Match, tokenOffset, last int, memo any) *EvalContext {
	return &EvalContext{Context: a.ctx, Helpers: a.helper, Text: text, OriginalText: original, Tokens: a.tokens, TokenOffset: tokenOffset, LastToken: last, Match: match, Country: a.country, AppContext: a.app, CondMemo: memo, Options: a.helper.Options}
}
func (a *analysis) tokenize(preserve bool) error {
	old := a.helper.TokenPositions
	var err error
	a.tokens, err = Tokenize(a.sentence, true)
	if err != nil {
		return err
	}
	a.helper.TokenPositions = map[int]*Token{}
	for _, token := range a.tokens {
		if preserve {
			if previous := old[token.Start]; previous != nil {
				token.Morphology = previous.Morphology
				token.Tags = previous.Tags
			}
		}
		if token.Type != "INFO" {
			a.helper.TokenPositions[token.Start] = token
		}
	}
	return nil
}
func (a *analysis) parseText(paragraph bool) error {
	text, original := a.sentence, a.sentence0
	if paragraph {
		text, original = a.text, a.original
	}
	changed := false
	for _, step := range a.engine.document.Steps {
		if step.Paragraph != paragraph {
			continue
		}
		if err := a.ctx.Err(); err != nil {
			return err
		}
		if step.GraphIndex >= 0 {
			if paragraph {
				return fmt.Errorf("graph called from paragraph pass")
			}
			if step.GraphIndex >= len(a.engine.graphs) {
				return fmt.Errorf("invalid graph index %d", step.GraphIndex)
			}
			if changed {
				a.sentence = text
				if err := a.tokenize(true); err != nil {
					return err
				}
				changed = false
			}
			graph := a.engine.graphs[step.GraphIndex]
			if enabled, exists := a.helper.Options[graph.Name]; !exists || enabled {
				if err := a.parseGraph(step.GraphIndex); err != nil {
					return err
				}
				text = a.sentence
			}
			continue
		}
		if step.RegexIndex < 0 || step.RegexIndex >= len(a.engine.document.RegexRules) {
			return fmt.Errorf("invalid regex rule index %d", step.RegexIndex)
		}
		rule := a.engine.document.RegexRules[step.RegexIndex]
		if a.options.IgnoredRules[rule.ID] || rule.Option != "" && !a.helper.Options[rule.Option] {
			continue
		}
		matches, err := a.engine.regex[step.RegexIndex].FindAll(a.ctx, text, -1)
		if err != nil {
			return fmt.Errorf("rule %s: %w", rule.ID, err)
		}
		for _, match := range matches {
			var memo any
			for _, action := range rule.Actions {
				eval := a.evalContext(text, original, &match, 0, 0, memo)
				eval.TargetGroup = action.Group
				memo = true
				if action.Condition != "" {
					memo, err = Evaluate(action.Condition, eval)
					if err != nil {
						return fmt.Errorf("%s %s condition: %w", rule.LineID, rule.ID, err)
					}
				}
				if !ruleexpr.Truth(memo) {
					if action.Kind == ">" {
						break
					}
					continue
				}
				switch action.Kind {
				case "-":
					offset := a.offset
					if paragraph {
						offset = 0
					}
					start, end := offset+match.Start(action.Group), offset+match.End(action.Group)
					if a.accept(start, rule.Priority) {
						suggestions, message, err := a.regexDiagnostic(action, rule, eval)
						if err != nil {
							return fmt.Errorf("%s %s: %w", rule.LineID, rule.ID, err)
						}
						a.addError(a.diagnostic(start, end, rule.LineID, rule.ID, rule.Option, message, suggestions, action.URL), rule.Priority)
					}
				case "~":
					replacement, err := a.regexRewrite(action, rule, eval)
					if err != nil {
						return fmt.Errorf("%s %s: %w", rule.LineID, rule.ID, err)
					}
					runes := []rune(text)
					start, end := match.Start(action.Group), match.End(action.Group)
					if start < 0 || end < start || end > len(runes) {
						return fmt.Errorf("invalid rewrite group %d", action.Group)
					}
					text = string(runes[:start]) + replacement + string(runes[end:])
					changed = true
				case "=":
					if !paragraph {
						if _, err := Evaluate(action.Value, eval); err != nil {
							return fmt.Errorf("%s %s: %w", rule.LineID, rule.ID, err)
						}
					}
				case ">":
				default:
					return fmt.Errorf("%s: unsupported action %q", rule.ID, action.Kind)
				}
			}
		}
	}
	if changed {
		if paragraph {
			a.text = text
		} else {
			a.sentence = text
		}
	}
	return nil
}
func (a *analysis) accept(position, priority int) bool {
	_, exists := a.errors[position]
	return !exists || priority > a.priorities[position]
}
func (a *analysis) addError(diagnostic Error, priority int) {
	if a.sentenceErrors != nil {
		if _, exists := a.sentenceErrors[diagnostic.Start]; !exists {
			a.sentenceOrder = append(a.sentenceOrder, diagnostic.Start)
		}
		a.sentenceErrors[diagnostic.Start] = diagnostic
	}
	if _, exists := a.errors[diagnostic.Start]; !exists {
		a.order = append(a.order, diagnostic.Start)
	}
	a.errors[diagnostic.Start] = diagnostic
	a.priorities[diagnostic.Start] = priority
}
func (a *analysis) removeError(position int) {
	if _, exists := a.errors[position]; !exists {
		return
	}
	delete(a.errors, position)
	for i, p := range a.order {
		if p == position {
			a.order = append(a.order[:i], a.order[i+1:]...)
			break
		}
	}
}
func (a *analysis) diagnostic(start, end int, line, id, option, message string, suggestions []string, url string) Error {
	if a.helper.Options["idrule"] {
		message += "  #" + line + " / " + id
	}
	kind := option
	if kind == "" {
		kind = "notype"
	}
	result := Error{Start: start, End: end, LineID: line, RuleID: id, Type: kind, Suggestions: suggestions, Message: message, URL: url}
	colors := a.engine.document.Colors[a.app]
	if colors == nil {
		colors = a.engine.document.Colors["Default"]
	}
	if color, ok := colors[option]; ok {
		result.Color = append([]int{}, color...)
	}
	if a.options.Context {
		result.withContext = true
		original := []rune(a.original)
		result.Underlined = sliceRunes(original, start, end)
		result.Before = sliceRunes(original, max(0, start-80), start)
		result.After = sliceRunes(original, end, end+80)
	}
	return result
}
func (a *analysis) evalString(expression string, eval *EvalContext) (string, error) {
	value, err := Evaluate(strings.TrimPrefix(expression, "="), eval)
	if err != nil {
		return "", err
	}
	if s, ok := ruleexpr.String(value); ok {
		return s, nil
	}
	return "", fmt.Errorf("expression returned %T, expected string: %s", value, expression)
}
func (a *analysis) regexDiagnostic(action rules.Action, rule rules.RegexRule, eval *EvalContext) ([]string, string, error) {
	suggestions := []string{}
	if action.Value != "_" {
		var value string
		var err error
		if strings.HasPrefix(action.Value, "=") {
			value, err = a.evalString(action.Value, eval)
		} else {
			value, err = expandMatch(action.Value, *eval.Match)
		}
		if err != nil {
			return nil, "", err
		}
		if value != "" || !strings.HasPrefix(action.Value, "=") {
			suggestions = strings.Split(value, "|")
		}
	}
	if rule.UppercaseSuggestions && startsUpper(eval.Match.Group(action.Group)) {
		suggestions = adaptCase(suggestions, strtransform.IsUpper(eval.Match.Group(action.Group)))
	}
	var message string
	var err error
	if strings.HasPrefix(action.Message, "=") {
		message, err = a.evalString(action.Message, eval)
	} else {
		message, err = expandMatch(action.Message, *eval.Match)
	}
	return suggestions, message, err
}
func (a *analysis) regexRewrite(action rules.Action, rule rules.RegexRule, eval *EvalContext) (string, error) {
	length := eval.Match.End(action.Group) - eval.Match.Start(action.Group)
	if length < 0 {
		return "", fmt.Errorf("rewrite target group did not participate")
	}
	if action.Value == "*" {
		return strings.Repeat(" ", length), nil
	}
	if action.Value == "_" || action.Value == "@" {
		return strings.Repeat(action.Value, length), nil
	}
	var value string
	var err error
	expression := strings.HasPrefix(action.Value, "=")
	if expression {
		value, err = a.evalString(action.Value, eval)
	} else {
		value, err = expandMatch(action.Value, *eval.Match)
	}
	if err != nil {
		return "", err
	}
	value += strings.Repeat(" ", max(0, length-utf8.RuneCountInString(value)))
	if expression && rule.UppercaseSuggestions && startsUpper(eval.Match.Group(action.Group)) {
		value = strtransform.Capitalize(value)
	}
	return value, nil
}
func startsUpper(s string) bool {
	r := []rune(s)
	return len(r) > 0 && strtransform.IsUpper(string(r[0]))
}
func adaptCase(values []string, allUpper bool) []string {
	for i, value := range values {
		if allUpper {
			values[i] = strtransform.Upper(value)
		} else {
			r := []rune(value)
			if len(r) > 0 {
				values[i] = strtransform.Upper(string(r[0])) + string(r[1:])
			}
		}
	}
	return values
}
func sliceRunes(r []rune, start, end int) string {
	start = max(0, min(start, len(r)))
	end = max(start, min(end, len(r)))
	return string(r[start:end])
}

// expandMatch interprets the replacement template once. Captured text is
// inserted literally, including any backslashes it contains.
func expandMatch(source string, match pyregex.Match) (string, error) {
	var out strings.Builder
	group := func(index int) error {
		if index < 0 || index >= len(match.Spans) {
			return fmt.Errorf("invalid match group %d", index)
		}
		out.WriteString(match.Group(index))
		return nil
	}
	for i := 0; i < len(source); {
		if source[i] != '\\' {
			out.WriteByte(source[i])
			i++
			continue
		}
		i++
		if i == len(source) {
			return "", fmt.Errorf("trailing backslash in replacement")
		}
		code := source[i]
		i++
		switch {
		case code == 'g':
			if i == len(source) || source[i] != '<' {
				return "", fmt.Errorf("missing < in replacement group")
			}
			i++
			end := strings.IndexByte(source[i:], '>')
			if end < 0 {
				return "", fmt.Errorf("unterminated replacement group")
			}
			name := source[i : i+end]
			i += end + 1
			index, err := strconv.Atoi(name)
			if err != nil {
				var exists bool
				index, exists = match.GroupIndex(name)
				if !exists {
					return "", fmt.Errorf("unknown replacement group %q", name)
				}
			}
			if err = group(index); err != nil {
				return "", err
			}
		case code >= '0' && code <= '9':
			digits := string(code)
			if code == '0' {
				for count := 0; count < 2 && i < len(source) && source[i] >= '0' && source[i] <= '7'; count++ {
					digits += string(source[i])
					i++
				}
				value, _ := strconv.ParseInt(digits, 8, 32)
				out.WriteRune(rune(value))
				continue
			}
			if i < len(source) && source[i] >= '0' && source[i] <= '9' {
				digits += string(source[i])
				i++
			}
			if len(digits) == 2 && digits[0] <= '7' && digits[1] <= '7' && i < len(source) && source[i] >= '0' && source[i] <= '7' {
				digits += string(source[i])
				i++
				value, _ := strconv.ParseInt(digits, 8, 32)
				if value > 255 {
					return "", fmt.Errorf("octal replacement escape is outside range: %s", digits)
				}
				out.WriteRune(rune(value))
				continue
			}
			index, _ := strconv.Atoi(digits)
			if err := group(index); err != nil {
				return "", err
			}
		default:
			escapes := map[byte]byte{'\\': '\\', 'a': '\a', 'b': '\b', 'f': '\f', 'n': '\n', 'r': '\r', 't': '\t', 'v': '\v'}
			if value, ok := escapes[code]; ok {
				out.WriteByte(value)
			} else if code >= 'a' && code <= 'z' || code >= 'A' && code <= 'Z' {
				return "", fmt.Errorf("invalid replacement escape \\%c", code)
			} else {
				out.WriteByte('\\')
				out.WriteByte(code)
			}
		}
	}
	return out.String(), nil
}

var repeatedAt = regexp.MustCompile(`@@+`)

func cleanText(text string) string {
	text = strings.NewReplacer(" ", " ", " ", " ", "'", "’", "‐", "-", "‑", "-").Replace(text)
	return repeatedAt.ReplaceAllStringFunc(text, func(s string) string { return strings.Repeat(" ", len(s)) })
}

func (a *analysis) errorList() []Error {
	out := make([]Error, 0, len(a.errors))
	for _, position := range a.order {
		if diagnostic, ok := a.errors[position]; ok {
			out = append(out, diagnostic)
		}
	}
	return out
}

// MarshalJSON retains all three context fields when context was requested,
// including empty strings at the beginning or end of a paragraph.
func (e Error) MarshalJSON() ([]byte, error) {
	type plain Error
	if e.withContext || e.Underlined != "" || e.Before != "" || e.After != "" {
		return json.Marshal(struct {
			plain
			Underlined string `json:"sUnderlined"`
			Before     string `json:"sBefore"`
			After      string `json:"sAfter"`
		}{plain(e), e.Underlined, e.Before, e.After})
	}
	return json.Marshal(plain(e))
}
func (e *Error) UnmarshalJSON(data []byte) error {
	type plain Error
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*e = Error(decoded)
	_, e.withContext = fields["sUnderlined"]
	return nil
}
