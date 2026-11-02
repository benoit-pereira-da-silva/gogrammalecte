package grammar

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/benoit-pereira-da-silva/gogrammalecte/french"
	"github.com/benoit-pereira-da-silva/gogrammalecte/internal/ruleexpr"
)

var expressions sync.Map

// Evaluate executes one parsed source expression against the current rule span.
// It exposes only grammar functions; no arbitrary host execution is available.
func Evaluate(source string, ctx *EvalContext) (any, error) {
	if ctx == nil || ctx.Helpers == nil || ctx.Helpers.Checker == nil {
		return nil, fmt.Errorf("expression requires an initialized grammar context")
	}
	if ctx.Context != nil {
		if err := ctx.Context.Err(); err != nil {
			return nil, err
		}
	}
	cached, ok := expressions.Load(source)
	if !ok {
		compiled, err := ruleexpr.Compile(source)
		if err != nil {
			return nil, err
		}
		cached, _ = expressions.LoadOrStore(source, compiled)
	}
	result, err := cached.(*ruleexpr.Expression).Eval(expressionEnvironment{ctx})
	if err != nil {
		return nil, fmt.Errorf("evaluate %q: %w", source, err)
	}
	return result, nil
}

type expressionEnvironment struct{ ctx *EvalContext }
type expressionReference struct {
	value    string
	word     *Word
	token    *Token
	position int
}

func (r expressionReference) Text() string { return r.value }

type expressionFunction string

func (e expressionEnvironment) Resolve(name string) (any, error) {
	switch name {
	case "sCountry":
		return e.ctx.Country, nil
	case "sContext", "_sAppContext":
		return e.ctx.AppContext, nil
	case "__also__", "bCondMemo":
		return e.ctx.CondMemo, nil
	case "__else__":
		return !ruleexpr.Truth(e.ctx.CondMemo), nil
	case "sSentence":
		return e.ctx.Text, nil
	case "sSentence0":
		return e.ctx.OriginalText, nil
	}
	if strings.HasPrefix(name, "sugg") || strings.HasPrefix(name, "switch") {
		return expressionFunction(name), nil
	}
	return nil, fmt.Errorf("unknown grammar name %s", name)
}
func (e expressionEnvironment) Reference(kind string, index int) (any, error) {
	c := e.ctx
	if kind == "@" {
		index = c.TargetGroup
		kind = "\\"
	}
	if c.Match != nil {
		if index < 0 || index >= len(c.Match.Spans) {
			return nil, fmt.Errorf("capture index %d out of range", index)
		}
		if kind == "/" {
			start, end := c.Match.Start(index), c.Match.End(index)
			if start < 0 {
				return "", nil
			}
			return runeSlice(c.OriginalText, start, end), nil
		}
		if kind != "\\" {
			return nil, fmt.Errorf("invalid regex reference %s", kind)
		}
		word := &Word{Start: c.Match.Start(index), Value: c.Match.Group(index)}
		return expressionReference{value: word.Value, word: word, token: c.Helpers.TokenPositions[word.Start]}, nil
	}
	var position int
	switch kind {
	case "\\":
		if index > 0 {
			position = c.TokenOffset + index
		} else {
			position = c.LastToken + index + 1
		}
	case ">":
		position = c.LastToken + index
	case "<":
		position = c.TokenOffset + 1 - index
	default:
		return nil, fmt.Errorf("unsupported graph reference %s", kind)
	}
	token := TokenAt(c.Tokens, position)
	if token == nil {
		return nil, fmt.Errorf("token index %d unavailable", position)
	}
	return expressionReference{value: token.Value, word: &Word{Start: token.Start, Value: token.Value}, token: token, position: max(0, min(position, len(c.Tokens)-1))}, nil
}
func runeSlice(s string, start, end int) string {
	r := []rune(s)
	start = max(0, min(start, len(r)))
	end = max(start, min(end, len(r)))
	return string(r[start:end])
}
func (e expressionEnvironment) span() (int, int, error) {
	c := e.ctx
	if c.Match != nil {
		return c.Match.Start(0), c.Match.End(0), nil
	}
	first, last := TokenAt(c.Tokens, c.TokenOffset+1), TokenAt(c.Tokens, c.LastToken)
	if first == nil || last == nil {
		return 0, 0, fmt.Errorf("expression requires a regex match or token span")
	}
	return first.Start, last.End, nil
}

type expressionArgs struct {
	values []any
	err    error
}

func (a *expressionArgs) value(i int) any {
	if i >= len(a.values) {
		return nil
	}
	return a.values[i]
}
func (a *expressionArgs) str(i int) string {
	if i >= len(a.values) {
		return ""
	}
	s, ok := ruleexpr.String(a.values[i])
	if !ok {
		a.err = fmt.Errorf("argument %d must be a string", i+1)
	}
	return s
}
func (a *expressionArgs) integer(i int, fallback int) int {
	if i >= len(a.values) || a.values[i] == nil {
		return fallback
	}
	n, ok := a.values[i].(int)
	if !ok {
		a.err = fmt.Errorf("argument %d must be an integer", i+1)
	}
	return n
}
func (a *expressionArgs) bound(i int) *int {
	if i >= len(a.values) || a.values[i] == nil {
		return nil
	}
	n := a.integer(i, 0)
	return &n
}
func (a *expressionArgs) boolean(i int, fallback bool) bool {
	if i >= len(a.values) {
		return fallback
	}
	return ruleexpr.Truth(a.values[i])
}
func (a *expressionArgs) token(i int) *Token {
	if r, ok := a.value(i).(expressionReference); ok && r.token != nil {
		return r.token
	}
	a.err = fmt.Errorf("argument %d must identify a graph token", i+1)
	return &Token{}
}
func (a *expressionArgs) position(i int) int {
	if r, ok := a.value(i).(expressionReference); ok && r.token != nil {
		return r.position
	}
	a.err = fmt.Errorf("argument %d must identify a graph token", i+1)
	return 0
}
func (a *expressionArgs) word(i int) *Word {
	switch r := a.value(i).(type) {
	case expressionReference:
		return r.word
	case *Word:
		return r
	case nil:
		return nil
	}
	a.err = fmt.Errorf("argument %d must identify a word", i+1)
	return nil
}
func (a *expressionArgs) require(n int) bool {
	if len(a.values) < n {
		a.err = fmt.Errorf("requires at least %d arguments", n)
		return false
	}
	return true
}

type expressionLexicon struct{ Lexicon }

func (e expressionLexicon) Lookup(s string) bool {
	if l, ok := e.Lexicon.(interface{ Lookup(string) bool }); ok {
		return l.Lookup(s)
	}
	return e.IsValid(s)
}
func morphToken(t *Token) french.MorphToken {
	return french.MorphToken{Value: t.Value, Morphologies: t.Morphology}
}

func (e expressionEnvironment) Call(name string, values []any) (result any, err error) {
	a := &expressionArgs{values: values}
	defer func() {
		if err == nil && a.err != nil {
			err = a.err
		}
	}()
	c, h := e.ctx, e.ctx.Helpers
	lexicon := expressionLexicon{h.Checker}
	sugg := french.NewSuggester(h.Checker)
	switch name {
	case "option":
		if !a.require(1) {
			return nil, a.err
		}
		return c.Options[a.str(0)], nil
	case "spell":
		if !a.require(1) {
			return nil, a.err
		}
		return h.Checker.IsValid(a.str(0)), nil
	case "re.search":
		if !a.require(2) {
			return nil, a.err
		}
		return h.Look(a.str(1), a.str(0), "")
	case "analyse":
		if !a.require(2) {
			return nil, a.err
		}
		return h.Analyse(a.str(0), a.str(1), a.str(2))
	case "morph", "morph0", "morphx", "morphVC":
		if !a.require(2) {
			return nil, a.err
		}
		if c.Match != nil {
			return h.Morph(a.word(0), a.str(1), a.str(2), a.boolean(3, false))
		}
		token := a.token(0)
		switch name {
		case "morph":
			return h.GMorph(token, a.str(1), a.str(2), a.bound(3), a.bound(4))
		case "morph0":
			return h.GMorph0(token, a.str(1), a.str(2), a.bound(3), a.bound(4))
		case "morphx":
			return h.GMorphX(token, a.str(1), a.str(2))
		default:
			return h.GMorphVC(token, a.str(1), a.str(2))
		}
	case "morph2":
		if !a.require(3) {
			return nil, a.err
		}
		first := a.token(0)
		position := a.position(0)
		if a.err != nil {
			return nil, a.err
		}
		return h.GMorph2(first, TokenAt(c.Tokens, position+1), a.str(1), a.str(2), a.str(3), a.boolean(4, true))
	case "value":
		if !a.require(2) {
			return nil, a.err
		}
		return h.GValue(a.token(0), a.str(1), a.bound(2), a.bound(3)), nil
	case "tag":
		if !a.require(2) {
			return nil, a.err
		}
		return h.GTag(a.token(0), a.str(1)), nil
	case "tagbefore":
		if !a.require(2) {
			return nil, a.err
		}
		return h.GTagBefore(a.token(0), a.str(1)), nil
	case "tagafter":
		if !a.require(2) {
			return nil, a.err
		}
		return h.GTagAfter(a.token(0), a.str(1)), nil
	case "meta":
		if !a.require(2) {
			return nil, a.err
		}
		return h.GMeta(a.token(0), a.str(1)), nil
	case "space":
		if !a.require(2) {
			return nil, a.err
		}
		first := a.token(0)
		position := a.position(0)
		if a.err != nil {
			return nil, a.err
		}
		return h.GSpace(first, TokenAt(c.Tokens, position+1), a.integer(1, 0), a.bound(2)), nil
	case "info":
		return true, nil
	case "word":
		if !a.require(1) {
			return nil, a.err
		}
		n := a.integer(0, 1)
		start, end, err := e.span()
		if err != nil {
			return nil, err
		}
		if n == 1 {
			return h.NextWord1(c.Text, end)
		}
		if n == -1 {
			return h.PreviousWord1(c.Text, start)
		}
		if n > 0 {
			return h.NextWord(c.Text, end, n)
		}
		return h.PreviousWord(c.Text, start, -n)
	case "before", "before0", "after", "after0", "textarea", "textarea0", "start", "realstart", "start0", "realstart0", "end", "realend", "end0", "realend0":
		input := c.Text
		if strings.HasSuffix(name, "0") {
			input = c.OriginalText
		}
		base := strings.TrimSuffix(name, "0")
		start, end := 0, 0
		if base != "textarea" {
			var err error
			start, end, err = e.span()
			if err != nil {
				return nil, err
			}
		}
		pattern, negative := a.str(0), a.str(1)
		switch base {
		case "before", "start", "realstart":
			input = runeSlice(input, 0, start)
		case "after", "end", "realend":
			input = runeSlice(input, end, len([]rune(input)))
		}
		switch base {
		case "start":
			pattern = "^ *$|, *$"
		case "realstart", "realend":
			pattern = "^ *$"
		case "end":
			pattern = "^ *$|^,"
		}
		return h.Look(input, pattern, negative)
	case "select":
		if !a.require(2) {
			return nil, a.err
		}
		if c.Match != nil {
			word := a.word(0)
			if word == nil {
				return false, a.err
			}
			return h.Select(word.Start, word.Value, a.str(1), a.str(2))
		}
		return h.GSelect(a.token(0), a.str(1), a.str(2))
	case "define":
		if !a.require(2) {
			return nil, a.err
		}
		if c.Match != nil {
			word := a.word(0)
			if word == nil {
				return false, a.err
			}
			return h.Define(word.Start, a.str(1)), nil
		}
		return h.GDefine(a.token(0), a.str(1)), nil
	case "definefrom":
		if !a.require(1) {
			return nil, a.err
		}
		return h.GDefineFrom(a.token(0), a.bound(1), a.bound(2)), nil
	case "addmorph":
		if !a.require(2) {
			return nil, a.err
		}
		return h.GAddMorph(a.token(0), a.str(1)), nil
	case "setmeta":
		if !a.require(2) {
			return nil, a.err
		}
		return h.GSetMeta(a.token(0), a.str(1)), nil
	case "rewrite":
		if !a.require(3) {
			return nil, a.err
		}
		return h.GRewrite(a.token(0), a.str(1), a.str(2), a.boolean(3, false))
	case "agreement":
		if !a.require(2) {
			return nil, a.err
		}
		return french.TokensAgree(lexicon, morphToken(a.token(0)), morphToken(a.token(1)), a.boolean(2, true)), nil
	case "apposition":
		if !a.require(2) {
			return nil, a.err
		}
		return french.IsApposition(lexicon, a.str(0), a.str(1)), nil
	case "queryNamesPOS":
		if !a.require(2) {
			return nil, a.err
		}
		return french.CombinedNounPOS(lexicon, a.str(0), a.str(1)), nil
	case "mbUnit":
		if !a.require(1) {
			return nil, a.err
		}
		return french.PossiblyUnit(a.str(0)), nil
	case "checkDate":
		if !a.require(3) {
			return nil, a.err
		}
		_, valid := french.CheckDate(a.str(0), a.str(1), a.str(2))
		return valid, nil
	case "checkDay":
		if !a.require(4) {
			return nil, a.err
		}
		return french.WeekdayMatches(a.str(0), a.str(1), a.str(2), a.str(3)), nil
	case "getDay":
		if !a.require(3) {
			return nil, a.err
		}
		return french.WeekdayName(a.str(0), a.str(1), a.str(2)), nil
	case "formatNumber":
		if !a.require(1) {
			return nil, a.err
		}
		return french.FormatNumber(a.str(0), a.boolean(1, false)), nil
	case "formatNF":
		if !a.require(1) {
			return nil, a.err
		}
		return french.FormatFrenchStandard(a.str(0)), nil
	case "undoLigature":
		if !a.require(1) {
			return nil, a.err
		}
		return french.UndoLigature(a.str(0)), nil
	case "normalizeInclusiveWriting":
		if !a.require(1) {
			return nil, a.err
		}
		return french.NormalizeInclusiveWriting(a.str(0)), nil
	case "hasSimil":
		if !a.require(1) {
			return nil, a.err
		}
		return french.HasPhoneticSimilar(a.str(0), a.str(1))
	case "hasFemForm":
		if !a.require(1) {
			return nil, a.err
		}
		return sugg.HasFeminineForm(a.str(0)), nil
	case "hasMasForm":
		if !a.require(1) {
			return nil, a.err
		}
		return sugg.HasMasculineForm(a.str(0)), nil
	case "suggVerb":
		if !a.require(2) {
			return nil, a.err
		}
		var extra []func(string) string
		var callbackErr error
		if len(values) > 3 && values[3] != nil {
			function, ok := values[3].(expressionFunction)
			if !ok {
				return nil, fmt.Errorf("invalid suggestion callback")
			}
			extra = append(extra, func(word string) string {
				arguments := []any{word}
				if len(values) > 4 {
					arguments = values[4:]
				}
				v, err := e.Call(string(function), arguments)
				if err != nil {
					callbackErr = err
					return ""
				}
				s, _ := ruleexpr.String(v)
				return s
			})
		}
		out := sugg.Verb(a.str(0), a.str(1), a.boolean(2, false), extra...)
		return out, callbackErr
	case "suggVerbPpas":
		if !a.require(1) {
			return nil, a.err
		}
		return sugg.PastParticiple(a.str(0), a.str(1)), nil
	case "suggVerbTense":
		if !a.require(3) {
			return nil, a.err
		}
		return sugg.VerbTense(a.str(0), a.str(1), a.str(2))
	case "suggVerbFrom":
		if !a.require(2) {
			return nil, a.err
		}
		return sugg.VerbFrom(a.str(0), a.str(1), a.str(2)), nil
	case "suggVerbImpe":
		if !a.require(1) {
			return nil, a.err
		}
		return sugg.Imperative(a.str(0), a.boolean(1, false)), nil
	case "suggVerbInfi":
		if !a.require(1) {
			return nil, a.err
		}
		return sugg.Infinitives(a.str(0)), nil
	case "suggVerbMode":
		if !a.require(3) {
			return nil, a.err
		}
		return sugg.VerbMode(a.str(0), a.str(1), a.str(2))
	case "suggPlur":
		if !a.require(1) {
			return nil, a.err
		}
		return sugg.Plural(a.str(0), a.boolean(1, false)), nil
	case "suggSing":
		if !a.require(1) {
			return nil, a.err
		}
		return sugg.Singular(a.str(0), a.boolean(1, true)), nil
	case "suggMasSing":
		if !a.require(1) {
			return nil, a.err
		}
		return sugg.MasculineSingular(a.str(0), a.boolean(1, false)), nil
	case "suggMasPlur":
		if !a.require(1) {
			return nil, a.err
		}
		return sugg.MasculinePlural(a.str(0), a.boolean(1, false)), nil
	case "suggFemSing":
		if !a.require(1) {
			return nil, a.err
		}
		return sugg.FeminineSingular(a.str(0), a.boolean(1, false)), nil
	case "suggFemPlur":
		if !a.require(1) {
			return nil, a.err
		}
		return sugg.FemininePlural(a.str(0), a.boolean(1, false)), nil
	case "suggAgree":
		if !a.require(2) {
			return nil, a.err
		}
		if c.Match == nil {
			if first, ok := a.value(0).(expressionReference); ok && first.token != nil {
				if second, ok := a.value(1).(expressionReference); ok && second.token != nil {
					return sugg.AgreeTokens(morphToken(first.token), morphToken(second.token)), nil
				}
			}
		}
		return sugg.Agree(a.str(0), a.str(1)), nil
	case "switchGender":
		if !a.require(1) {
			return nil, a.err
		}
		var plural *bool
		if len(values) > 1 && values[1] != nil {
			v := a.boolean(1, false)
			plural = &v
		}
		return sugg.SwitchGender(a.str(0), plural), nil
	case "switchPlural":
		if !a.require(1) {
			return nil, a.err
		}
		return sugg.SwitchPlural(a.str(0)), nil
	case "suggSimil":
		if !a.require(1) {
			return nil, a.err
		}
		return sugg.Similar(a.str(0), a.str(1), a.boolean(2, false), a.boolean(3, false))
	case "suggCeOrCet":
		if !a.require(1) {
			return nil, a.err
		}
		return french.SuggestDemonstrative(a.str(0)), nil
	case "suggLesLa":
		if !a.require(1) {
			return nil, a.err
		}
		return sugg.LesLa(a.str(0)), nil
	}
	return nil, fmt.Errorf("unknown grammar function %s", name)
}

// contextOrBackground keeps standalone expression evaluation cancellable.
func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
