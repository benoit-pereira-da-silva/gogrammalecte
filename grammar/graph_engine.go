package grammar

// Token graph execution is adapted from Grammalecte's gc_engine.py by Olivier R.
// The source is distributed under GPLv3 or later.

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/benoit-pereira-da-silva/gogrammalecte/french"
	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/strtransform"
	"github.com/benoit-pereira-da-silva/gogrammalecte/internal/ruleexpr"
	"github.com/benoit-pereira-da-silva/gogrammalecte/rules"
)

type graphIndex struct {
	literals   map[string]int
	categories map[string][]rules.GraphArc
}
type graphTransition struct {
	kind string
	node int
}
type graphPointer struct{ first, node, multiEnd int }

func indexGraph(graph rules.CompiledGraph) []graphIndex {
	nodes := make([]graphIndex, len(graph.Nodes))
	for i, node := range graph.Nodes {
		nodes[i] = graphIndex{map[string]int{}, map[string][]rules.GraphArc{}}
		for _, arc := range node.Arcs {
			if arc.Kind == "literal" {
				nodes[i].literals[arc.Value] = arc.Next
			} else {
				nodes[i].categories[arc.Kind] = append(nodes[i].categories[arc.Kind], arc)
			}
		}
	}
	return nodes
}
func (a *analysis) graphSearch(pattern, value string) (bool, error) {
	compiled, ok := a.engine.patterns[pattern]
	if !ok {
		var err error
		compiled, err = helperPattern(pattern)
		if err != nil {
			return false, err
		}
	}
	match, err := compiled.Search(a.ctx, value)
	return match != nil, err
}
func (a *analysis) matchesMorph(source string, morphology []string, regex bool) (bool, error) {
	if len(morphology) == 0 {
		return false, nil
	}
	positive, negative, hasNegative := strings.Cut(source, "¬")
	match := func(pattern, morph string) (bool, error) {
		if regex {
			return a.graphSearch(pattern, morph)
		}
		return strings.Contains(morph, pattern), nil
	}
	if hasNegative && negative == "*" {
		if positive == "" {
			return false, nil
		}
		for _, morph := range morphology {
			ok, err := match(positive, morph)
			if err != nil || !ok {
				return false, err
			}
		}
		return true, nil
	}
	if negative != "" {
		for _, morph := range morphology {
			ok, err := match(negative, morph)
			if err != nil {
				return false, err
			}
			if ok {
				return false, nil
			}
		}
	}
	if positive == "" {
		return true, nil
	}
	for _, morph := range morphology {
		ok, err := match(positive, morph)
		if err != nil || ok {
			return ok, err
		}
	}
	return false, nil
}
func (a *analysis) nextNodes(graphNumber, nodeNumber int, token *Token, keep bool) ([]graphTransition, error) {
	node := a.engine.indexes[graphNumber][nodeNumber]
	out := []graphTransition{}
	addLiteral := func(value string) {
		if next, ok := node.literals[value]; ok {
			out = append(out, graphTransition{"literal", next})
		}
	}
	addLiteral(token.Value)
	chars := []rune(token.Value)
	if strtransform.IsTitle(string(chars[:min(2, len(chars))])) {
		addLiteral(strtransform.Lower(token.Value))
	} else if strtransform.IsUpper(token.Value) {
		addLiteral(strtransform.Lower(token.Value))
		addLiteral(strtransform.Capitalize(token.Value))
	}
	if token.Type != "INFO" && token.Type != "PUNC" && token.Type != "SIGN" {
		for _, arc := range node.categories["<re_value>"] {
			positive, negative, _ := strings.Cut(arc.Value, "¬")
			if negative != "" {
				matches, err := a.graphSearch(negative, token.Value)
				if err != nil {
					return nil, err
				}
				if matches {
					continue
				}
			}
			matches := positive == ""
			var err error
			if !matches {
				matches, err = a.graphSearch(positive, token.Value)
				if err != nil {
					return nil, err
				}
			}
			if matches {
				out = append(out, graphTransition{arc.Kind, arc.Next})
			}
		}
	}
	if strings.HasPrefix(token.Type, "WORD") {
		if arcs := node.categories["<lemmas>"]; len(arcs) > 0 {
			for _, lemma := range a.engine.checker.Lemmas(token.Value) {
				for _, arc := range arcs {
					if arc.Value == lemma {
						out = append(out, graphTransition{arc.Kind, arc.Next})
					}
				}
			}
		}
		for _, arc := range node.categories["<phonet>"] {
			phonetic := strings.TrimRight(arc.Value, "!")
			if strings.HasSuffix(arc.Value, "!") {
				if token.Value == phonetic || startsUpper(token.Value) && (strtransform.Lower(token.Value) == phonetic || strtransform.IsUpper(token.Value) && strtransform.Capitalize(token.Value) == phonetic) {
					continue
				}
			}
			if french.ArePhoneticallySimilar(token.Value, phonetic) {
				out = append(out, graphTransition{arc.Kind, arc.Next})
			}
		}
		morphology := token.Morphology
		if len(node.categories["<morph>"])+len(node.categories["<re_morph>"]) > 0 && morphology == nil {
			morphology = a.engine.checker.Morph(token.Value)
		}
		for _, kind := range []string{"<morph>", "<re_morph>"} {
			for _, arc := range node.categories[kind] {
				matched, err := a.matchesMorph(arc.Value, morphology, kind == "<re_morph>")
				if err != nil {
					return nil, err
				}
				if matched {
					out = append(out, graphTransition{arc.Kind, arc.Next})
				}
			}
		}
		if token.MultiStartTo != nil && token.Multi != nil {
			for _, arc := range node.categories["<re_mmorph>"] {
				matched, err := a.matchesMorph(arc.Value, token.Multi.Morphology, true)
				if err != nil {
					return nil, err
				}
				if matched {
					out = append(out, graphTransition{arc.Kind, arc.Next})
				}
			}
		}
	}
	if len(token.Tags) > 0 && len(node.categories["<tags>"]) > 0 {
		tags := make([]string, 0, len(token.Tags))
		for tag, enabled := range token.Tags {
			if enabled {
				tags = append(tags, tag)
			}
		}
		sort.Strings(tags)
		for _, tag := range tags {
			for _, arc := range node.categories["<tags>"] {
				if arc.Value == tag {
					out = append(out, graphTransition{arc.Kind, arc.Next})
				}
			}
		}
	}
	for _, arc := range node.categories["<meta>"] {
		if arc.Value == "*" || token.Type == arc.Value || strings.Contains(arc.Value, "¬") && !strings.Contains(arc.Value, token.Type) {
			out = append(out, graphTransition{arc.Kind, arc.Next})
		}
	}
	if len(out) == 0 && keep {
		out = append(out, graphTransition{"", -1})
	}
	if next, ok := node.literals["<>"]; ok {
		more, err := a.nextNodes(graphNumber, next, token, true)
		if err != nil {
			return nil, err
		}
		out = append(out, more...)
	}
	return out, nil
}
func (a *analysis) parseGraph(graphNumber int) error {
	pointers := []graphPointer{}
	rewrite := false
	for i, token := range a.tokens {
		if err := a.ctx.Err(); err != nil {
			return err
		}
		nextPointers := []graphPointer{}
		for _, pointer := range pointers {
			if pointer.multiEnd != -1 {
				if token.Index <= pointer.multiEnd {
					nextPointers = append(nextPointers, pointer)
				}
				if token.Index != pointer.multiEnd {
					continue
				}
			}
			next, err := a.nextNodes(graphNumber, pointer.node, token, false)
			if err != nil {
				return fmt.Errorf("graph %s: %w", a.engine.graphs[graphNumber].Name, err)
			}
			for _, transition := range next {
				if transition.kind == "" {
					nextPointers = append(nextPointers, pointer)
					continue
				}
				multiEnd := -1
				if transition.kind == "<re_mmorph>" {
					multiEnd = *token.MultiStartTo
				}
				nextPointers = append(nextPointers, graphPointer{pointer.first, transition.node, multiEnd})
			}
		}
		pointers = nextPointers
		next, err := a.nextNodes(graphNumber, 0, token, false)
		if err != nil {
			return err
		}
		for _, transition := range next {
			if transition.kind == "" {
				continue
			}
			multiEnd := -1
			if transition.kind == "<re_mmorph>" {
				multiEnd = *token.MultiStartTo
			}
			pointers = append(pointers, graphPointer{i, transition.node, multiEnd})
		}
		for j := range pointers {
			pointer := &pointers[j]
			if pointer.multiEnd != -1 {
				if token.Index < pointer.multiEnd {
					continue
				}
				if token.Index >= pointer.multiEnd {
					pointer.multiEnd = -1
				}
			}
			changed, err := a.executeGraphActions(graphNumber, pointer.node, pointer.first-1, i)
			if err != nil {
				return err
			}
			rewrite = rewrite || changed
		}
	}
	if rewrite {
		return a.rewriteTokens()
	}
	return nil
}
func (a *analysis) tokenRange(action rules.GraphAction, offset, last int) (int, int, error) {
	start, end := offset+action.StartToken, offset+action.EndToken
	if action.StartToken <= 0 {
		start = last + action.StartToken
	}
	if action.EndToken <= 0 {
		end = last + action.EndToken
	}
	if start < 0 || end < 0 || start >= len(a.tokens) || end >= len(a.tokens) {
		return 0, 0, fmt.Errorf("invalid token range %d:%d in %s", start, end, action.ID)
	}
	return start, end, nil
}
func (a *analysis) executeGraphActions(graphNumber, nodeNumber, offset, last int) (bool, error) {
	changed := false
	graph := a.engine.graphs[graphNumber]
	for _, arc := range a.engine.indexes[graphNumber][nodeNumber].categories["<rules>"] {
		var memo any
		for _, id := range graph.Nodes[arc.Next].Actions {
			if a.options.IgnoredRules[id] {
				continue
			}
			action, exists := a.engine.actions[graphNumber][id]
			if !exists {
				return false, fmt.Errorf("missing graph action %s", id)
			}
			if action.Option != "" && !a.helper.Options[action.Option] {
				continue
			}
			eval := a.evalContext(a.sentence, a.sentence0, nil, offset, last, memo)
			memo = true
			if action.Condition != "" {
				var err error
				memo, err = Evaluate(action.Condition, eval)
				if err != nil {
					return false, fmt.Errorf("%s %s condition: %w", arc.Value, id, err)
				}
			}
			if !ruleexpr.Truth(memo) {
				if action.Kind == ">" {
					break
				}
				continue
			}
			if action.Kind == ">" {
				continue
			}
			if action.Kind == "=" {
				if _, err := Evaluate(action.Value, eval); err != nil {
					return false, fmt.Errorf("%s %s: %w", arc.Value, id, err)
				}
				continue
			}
			start, end, err := a.tokenRange(action, offset, last)
			if err != nil {
				return false, err
			}
			switch action.Kind {
			case "-":
				first := a.tokens[start]
				if first.Immunity == "*" || first.Immunity != "" && strings.Contains(first.Immunity, action.Option) {
					continue
				}
				errorStart, errorEnd := a.offset+first.Start, a.offset+a.tokens[end].End
				if action.StartLimit == ">" {
					errorStart = a.offset + first.End
				}
				if action.EndLimit == "<" {
					errorEnd = a.offset + a.tokens[end].Start
				}
				if !a.accept(errorStart, action.Priority) {
					continue
				}
				suggestions, message, err := a.graphDiagnostic(action, eval, start, errorStart, errorEnd)
				if err != nil {
					return false, fmt.Errorf("%s %s: %w", arc.Value, id, err)
				}
				a.addError(a.diagnostic(errorStart, errorEnd, arc.Value, id, action.Option, message, suggestions, action.URL), action.Priority)
			case "~":
				if err := a.prepareRewrite(action, eval, start, end); err != nil {
					return false, fmt.Errorf("%s %s: %w", arc.Value, id, err)
				}
				changed = true
			case "/":
				for i := start; i <= end; i++ {
					if a.tokens[i].Tags == nil {
						a.tokens[i].Tags = map[string]bool{}
					}
					for _, tag := range strings.Split(action.Value, "|") {
						a.tokens[i].Tags[tag] = true
					}
				}
				for _, tag := range strings.Split(action.Value, "|") {
					old, exists := a.helper.Tags[tag]
					if !exists {
						a.helper.Tags[tag] = [2]int{start, end}
					} else {
						a.helper.Tags[tag] = [2]int{min(old[0], start), max(old[1], end)}
					}
				}
			case "!":
				immunity := action.Value
				if immunity == "" {
					immunity = "*"
				}
				for i := start; i <= end; i++ {
					a.tokens[i].Immunity = immunity
					a.removeError(a.offset + a.tokens[i].Start)
				}
			case "&":
				morphology := []string{":HM"}
				if action.Value != "" {
					morphology = strings.Split(action.Value, "|")
				}
				multi := &MultiToken{Start: start, End: end, Tokens: a.tokens[start : end+1], Morphology: morphology}
				startIndex, endIndex := a.tokens[start].Index, a.tokens[end].Index
				a.tokens[start].MultiStartTo = &endIndex
				a.tokens[end].MultiEndFrom = &startIndex
				a.tokens[start].Multi = multi
				a.tokens[end].Multi = multi
			default:
				return false, fmt.Errorf("unsupported graph action %s: %q", id, action.Kind)
			}
		}
	}
	return changed, nil
}
func (a *analysis) expandTokens(source string, offset, last int) (string, error) {
	var out strings.Builder
	for i := 0; i < len(source); {
		if source[i] != '\\' {
			out.WriteByte(source[i])
			i++
			continue
		}
		start := i
		i++
		negative := i < len(source) && source[i] == '-'
		if negative {
			i++
		}
		digits := i
		for i < len(source) && source[i] >= '0' && source[i] <= '9' {
			i++
		}
		if digits == i {
			out.WriteString(source[start:i])
			continue
		}
		number, _ := strconv.Atoi(source[start+1 : i])
		index := offset + number
		if negative {
			index = last + number + 1
		}
		if index < 0 || index >= len(a.tokens) {
			return "", fmt.Errorf("token reference %s out of range", source[start:i])
		}
		out.WriteString(a.tokens[index].Value)
	}
	return out.String(), nil
}
func (a *analysis) graphDiagnostic(action rules.GraphAction, eval *EvalContext, first, start, end int) ([]string, string, error) {
	suggestions := []string{}
	if action.Value != "_" {
		var value string
		var err error
		if action.ValueExpression {
			value, err = a.evalString(action.Value, eval)
		} else {
			value, err = a.expandTokens(action.Value, eval.TokenOffset, eval.LastToken)
		}
		if err != nil {
			return nil, "", err
		}
		if value != "" || !action.ValueExpression {
			suggestions = strings.Split(value, "|")
		}
	}
	if action.CaseSensitive && startsUpper(a.tokens[first].Value) {
		suggestions = adaptCase(suggestions, strtransform.IsUpper(sliceRunes([]rune(a.sentence), start, end)))
	}
	var message string
	var err error
	if action.MessageExpression {
		message, err = a.evalString(action.Message, eval)
	} else {
		message, err = a.expandTokens(action.Message, eval.TokenOffset, eval.LastToken)
	}
	return suggestions, message, err
}
func (a *analysis) prepareRewrite(action rules.GraphAction, eval *EvalContext, start, end int) error {
	value := action.Value
	if value == "*" && !action.ValueExpression {
		for i := start; i <= end; i++ {
			a.tokens[i].ToRemove = true
		}
		return nil
	}
	if strings.HasPrefix(value, "␣") && !action.ValueExpression {
		a.tokens[start].MergeUntil = &end
		if value != "␣" {
			expanded, err := a.expandTokens(strings.TrimPrefix(value, "␣"), eval.TokenOffset, eval.LastToken)
			if err != nil {
				return err
			}
			a.tokens[start].MergedValue = &expanded
		}
		return nil
	}
	if value == "_" && !action.ValueExpression {
		for i := start; i <= end; i++ {
			replacement := "_"
			a.tokens[i].NewValue = &replacement
		}
		return nil
	}
	var err error
	if action.ValueExpression {
		value, err = a.evalString(value, eval)
	} else {
		value, err = a.expandTokens(value, eval.TokenOffset, eval.LastToken)
	}
	if err != nil {
		return err
	}
	uppercase := action.CaseSensitive && startsUpper(a.tokens[start].Value)
	if start == end {
		if uppercase {
			value = adaptCase([]string{value}, false)[0]
		}
		a.tokens[start].NewValue = &value
		return nil
	}
	values := strings.Split(value, "|")
	if len(values) != end-start+1 {
		return fmt.Errorf("rewrite has %d values for %d tokens", len(values), end-start+1)
	}
	for i, replacement := range values {
		token := a.tokens[start+i]
		if replacement == "" || replacement == "*" {
			token.ToRemove = true
		} else {
			if uppercase {
				replacement = adaptCase([]string{replacement}, false)[0]
			}
			token.NewValue = &replacement
		}
	}
	return nil
}
func (a *analysis) replaceSentence(start, end int, value string) {
	runes := []rune(a.sentence)
	a.sentence = sliceRunes(runes, 0, start) + value + sliceRunes(runes, end, len(runes))
}
func (a *analysis) rewriteTokens() error {
	tokens := make([]*Token, 0, len(a.tokens))
	mergeUntil := 0
	var merger *Token
	for i, token := range a.tokens {
		keep := true
		if token.Type != "INFO" {
			if mergeUntil != 0 && i <= mergeUntil {
				if merger == nil {
					return fmt.Errorf("invalid token merge state")
				}
				merger.Value += strings.Repeat(" ", max(0, token.Start-merger.End)) + token.Value
				merger.End = token.End
				token.Merged = true
				keep = false
				if i == mergeUntil && merger.MergedValue != nil {
					merger.Value = *merger.MergedValue
					filler := strings.Repeat(" ", max(0, token.End-merger.Start-utf8.RuneCountInString(*merger.MergedValue)))
					a.replaceSentence(merger.Start, token.End, *merger.MergedValue+filler)
				}
			}
			if token.MergeUntil != nil {
				if i > mergeUntil {
					merger = token
				}
				mergeUntil = max(mergeUntil, *token.MergeUntil)
				token.MergeUntil = nil
			} else if token.ToRemove {
				a.replaceSentence(token.Start, token.End, strings.Repeat(" ", max(0, token.End-token.Start)))
				keep = false
			}
		}
		if keep {
			tokens = append(tokens, token)
			if token.NewValue != nil {
				token.RealValue = token.Value
				token.Value = *token.NewValue
				length := utf8.RuneCountInString(token.RealValue)
				runes := []rune(token.Value)
				replacement := string(runes[:min(length, len(runes))]) + strings.Repeat(" ", max(0, length-len(runes)))
				a.replaceSentence(token.Start, token.End, replacement)
				token.NewValue = nil
			}
		}
	}
	a.tokens = tokens
	return nil
}
