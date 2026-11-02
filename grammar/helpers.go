package grammar

// Grammar helpers ported from gc_core/py/lang_core/gc_functions.py and
// gc_lang/fr/modules/gce_analyseur.py by Olivier R. (GPLv3).

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/benoit-pereira-da-silva/gogrammalecte/internal/pyregex"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Lexicon interface {
	Morph(string) []string
	IsValid(string) bool
	Lemmas(string) []string
}

// HelperContext contains the mutable analysis state for a single parse.
// A context must not be shared by concurrent parses; compiled patterns are
// cached separately and can be shared safely.
type HelperContext struct {
	Context        context.Context
	Checker        Lexicon
	TokenPositions map[int]*Token
	Tags           map[string][2]int
	Options        map[string]bool
}

func NewHelperContext(checker Lexicon) *HelperContext {
	return &HelperContext{Context: context.Background(), Checker: checker, TokenPositions: map[int]*Token{}, Tags: map[string][2]int{}, Options: map[string]bool{}}
}
func (h *HelperContext) context() context.Context {
	if h.Context == nil {
		return context.Background()
	}
	return h.Context
}

var helperPatterns sync.Map

func helperPattern(pattern string) (*pyregex.Regexp, error) {
	if value, ok := helperPatterns.Load(pattern); ok {
		return value.(*pyregex.Regexp), nil
	}
	r, err := pyregex.Compile(pattern)
	if err != nil {
		return nil, err
	}
	actual, _ := helperPatterns.LoadOrStore(pattern, r)
	return actual.(*pyregex.Regexp), nil
}
func (h *HelperContext) search(pattern, input string) (*pyregex.Match, error) {
	r, err := helperPattern(pattern)
	if err != nil {
		return nil, err
	}
	return r.Search(h.context(), input)
}
func (h *HelperContext) Option(name string) bool { return h.Options[name] }

func (h *HelperContext) outsideWord(input string, position int, pattern string, forward bool) (*Word, error) {
	r := []rune(input)
	part := ""
	if forward {
		part = string(r[sliceIndex(position, len(r)):])
	} else {
		part = string(r[:sliceIndex(position, len(r))])
	}
	m, err := h.search(pattern, part)
	if err != nil || m == nil {
		return nil, err
	}
	start := m.Start(1)
	if forward {
		start += position
	}
	return &Word{start, m.Group(1)}, nil
}
func (h *HelperContext) NextWord(input string, start, n int) (*Word, error) {
	if n < 1 {
		return nil, fmt.Errorf("word number must be positive: %d", n)
	}
	return h.outsideWord(input, start, `^(?: +[\w%-]+){`+strconv.Itoa(n-1)+`} +([\w%-]+)`, true)
}
func (h *HelperContext) PreviousWord(input string, end, n int) (*Word, error) {
	if n < 1 {
		return nil, fmt.Errorf("word number must be positive: %d", n)
	}
	return h.outsideWord(input, end, `([\w%-]+) +(?:[\w%-]+ +){`+strconv.Itoa(n-1)+`}$`, false)
}
func (h *HelperContext) NextWord1(input string, start int) (*Word, error) {
	return h.outsideWord(input, start, `^ +(\w[\w-]*)`, true)
}
func (h *HelperContext) PreviousWord1(input string, end int) (*Word, error) {
	return h.outsideWord(input, end, `(\w[\w-]*) +$`, false)
}

func (h *HelperContext) Look(input, pattern, negative string) (bool, error) {
	if negative != "" {
		m, err := h.search(negative, input)
		if err != nil || m != nil {
			return false, err
		}
	}
	m, err := h.search(pattern, input)
	return m != nil, err
}
func (h *HelperContext) LookCheck1(input string, offset int, pattern, morphPattern, negative string) (bool, error) {
	m, err := h.search(pattern, input)
	if err != nil || m == nil {
		return false, err
	}
	if len(m.Spans) < 2 {
		return false, nil
	}
	return h.Morph(&Word{m.Start(1) + offset, m.Group(1)}, morphPattern, negative, false)
}

func (h *HelperContext) matchMorphologies(morphologies []string, pattern, negative string) (bool, error) {
	if len(morphologies) == 0 {
		return false, nil
	}
	if negative != "" && negative != "*" {
		for _, m := range morphologies {
			match, err := h.search(negative, m)
			if err != nil || match != nil {
				return false, err
			}
		}
	}
	for _, m := range morphologies {
		match, err := h.search(pattern, m)
		if err != nil {
			return false, err
		}
		if negative == "*" {
			if match == nil {
				return false, nil
			}
		} else if match != nil {
			return true, nil
		}
	}
	return negative == "*", nil
}
func (h *HelperContext) Morph(word *Word, pattern, negative string, noWord bool) (bool, error) {
	if word == nil {
		return noWord, nil
	}
	var morphologies []string
	if token := h.TokenPositions[word.Start]; token != nil {
		morphologies = token.Morphology
	}
	if morphologies == nil {
		morphologies = h.Checker.Morph(word.Value)
	}
	return h.matchMorphologies(morphologies, pattern, negative)
}
func (h *HelperContext) Analyse(word, pattern, negative string) (bool, error) {
	return h.matchMorphologies(h.Checker.Morph(word), pattern, negative)
}

func titleCase(input string) string {
	r := []rune(lowerCase(input))
	if len(r) > 0 {
		return cases.Title(language.Und).String(string(r[0])) + string(r[1:])
	}
	return string(r)
}
func lowerCase(input string) string { return cases.Lower(language.Und).String(input) }
func isUpper(input string) bool {
	found := false
	for _, r := range input {
		if unicode.IsLower(r) || unicode.IsTitle(r) {
			return false
		}
		if unicode.IsUpper(r) {
			found = true
		}
	}
	return found
}
func isTitle(input string) bool {
	found, previous := false, false
	for _, r := range input {
		if unicode.IsUpper(r) || unicode.IsTitle(r) {
			if previous {
				return false
			}
			found = true
			previous = true
		} else if unicode.IsLower(r) {
			if !previous {
				return false
			}
			found = true
			previous = true
		} else {
			previous = false
		}
	}
	return found
}
func (h *HelperContext) GValue(token *Token, values string, left, right *int) bool {
	value := "|" + SliceText(token.Value, left, right) + "|"
	if strings.Contains(values, value) {
		return true
	}
	r := []rune(token.Value)
	prefix := r
	if len(prefix) > 2 {
		prefix = prefix[:2]
	}
	if isTitle(string(prefix)) {
		return strings.Contains(values, lowerCase(value))
	}
	if isUpper(token.Value) {
		value = "|" + titleCase(value[1:])
		return strings.Contains(values, value) || strings.Contains(values, lowerCase(value))
	}
	return false
}
func (h *HelperContext) tokenMorphologies(token *Token, left, right *int, cached bool) []string {
	if cached && token.Morphology != nil {
		return token.Morphology
	}
	return h.Checker.Morph(SliceText(token.Value, left, right))
}
func (h *HelperContext) GMorph(token *Token, pattern, negative string, left, right *int) (bool, error) {
	return h.matchMorphologies(h.tokenMorphologies(token, left, right, true), pattern, negative)
}
func (h *HelperContext) GMorph0(token *Token, pattern, negative string, left, right *int) (bool, error) {
	return h.matchMorphologies(h.tokenMorphologies(token, left, right, false), pattern, negative)
}
func (h *HelperContext) GMorphX(token *Token, pattern, negative string) (bool, error) {
	if token.Multi == nil {
		return false, nil
	}
	return h.matchMorphologies(token.Multi.Morphology, pattern, negative)
}
func (h *HelperContext) GMorph2(first, second *Token, merger, pattern, negative string, setMorph bool) (bool, error) {
	morphologies := h.Checker.Morph(first.Value + merger + second.Value)
	matched, err := h.matchMorphologies(morphologies, pattern, negative)
	if matched && err == nil && setMorph {
		first.Morphology = cloneMorphologies(morphologies)
	}
	return matched, err
}

// GMorphVC inspects the verb head of a hyphenated compound. Cached analyses
// retain precedence, as in the Python graph helper.
func (h *HelperContext) GMorphVC(token *Token, pattern, negative string) (bool, error) {
	r := []rune(token.Value)
	end := -1
	count := 0
	for i, c := range r {
		if c == '-' {
			end = i
			count++
		}
	}
	if count > 1 {
		if strings.Contains(token.Value, "-t-") {
			end -= 2
		} else {
			m, err := h.search(`-l(?:es?|a)-(?:[mt]oi|nous|leur)$|(?:[nv]ous|lui|leur)-en$`, token.Value)
			if err != nil {
				return false, err
			}
			if m != nil {
				for i := end - 1; i >= 0; i-- {
					if r[i] == '-' {
						end = i
						break
					}
				}
			}
		}
	}
	left := 0
	return h.GMorph(token, pattern, negative, &left, &end)
}

func (h *HelperContext) GTagBefore(token *Token, tag string) bool {
	bounds, ok := h.Tags[tag]
	return ok && token.Index > bounds[0]
}
func (h *HelperContext) GTagAfter(token *Token, tag string) bool {
	bounds, ok := h.Tags[tag]
	return ok && token.Index < bounds[1]
}
func (h *HelperContext) GTag(token *Token, tag string) bool   { return token.Tags[tag] }
func (h *HelperContext) GMeta(token *Token, kind string) bool { return token.Type == kind }
func (h *HelperContext) GSpace(first, second *Token, minimum int, maximum *int) bool {
	space := second.Start - first.End
	return space >= minimum && (maximum == nil || space <= *maximum)
}

func cloneMorphologies(morphologies []string) []string {
	out := make([]string, len(morphologies))
	copy(out, morphologies)
	return out
}
func (h *HelperContext) selectMorphologies(token *Token, morphologies []string, pattern, negative string) (bool, error) {
	if len(morphologies) < 2 || pattern == "" && negative == "" {
		return true, nil
	}
	var selected []string
	for _, m := range morphologies {
		if pattern != "" {
			match, err := h.search(pattern, m)
			if err != nil {
				return false, err
			}
			if match == nil {
				continue
			}
		}
		if negative != "" {
			match, err := h.search(negative, m)
			if err != nil {
				return false, err
			}
			if match != nil {
				continue
			}
		}
		selected = append(selected, m)
	}
	if len(selected) > 0 && len(selected) != len(morphologies) {
		token.Morphology = selected
	}
	return true, nil
}
func (h *HelperContext) Select(position int, word, pattern, negative string) (bool, error) {
	token := h.TokenPositions[position]
	if word == "" || token == nil {
		return true, nil
	}
	return h.selectMorphologies(token, h.Checker.Morph(word), pattern, negative)
}
func (h *HelperContext) Define(position int, morphologies string) bool {
	if token := h.TokenPositions[position]; token != nil {
		token.Morphology = strings.Split(morphologies, "|")
	}
	return true
}
func (h *HelperContext) GSelect(token *Token, pattern, negative string) (bool, error) {
	return h.selectMorphologies(token, h.tokenMorphologies(token, nil, nil, true), pattern, negative)
}
func (h *HelperContext) GAddMorph(token *Token, morphologies string) bool {
	token.Morphology = append(cloneMorphologies(h.tokenMorphologies(token, nil, nil, true)), strings.Split(morphologies, "|")...)
	return true
}
func (h *HelperContext) GDefine(token *Token, morphologies string) bool {
	token.Morphology = strings.Split(morphologies, "|")
	return true
}
func (h *HelperContext) GDefineFrom(token *Token, left, right *int) bool {
	token.Morphology = cloneMorphologies(h.tokenMorphologies(token, left, right, false))
	return true
}
func (h *HelperContext) GSetMeta(token *Token, kind string) bool { token.Type = kind; return true }

func (h *HelperContext) GRewrite(token *Token, old, replacement string, regex bool) (bool, error) {
	morphologies := h.tokenMorphologies(token, nil, nil, true)
	out := make([]string, len(morphologies))
	for i, morph := range morphologies {
		if !regex {
			out[i] = strings.ReplaceAll(morph, old, replacement)
			continue
		}
		pattern, err := helperPattern(old)
		if err != nil {
			return false, err
		}
		matches, err := pattern.FindAll(h.context(), morph, -1)
		if err != nil {
			return false, err
		}
		r := []rune(morph)
		position := 0
		var value strings.Builder
		for _, match := range matches {
			value.WriteString(string(r[position:match.Start(0)]))
			expanded, err := expandReplacement(replacement, match)
			if err != nil {
				return false, err
			}
			value.WriteString(expanded)
			position = match.End(0)
		}
		value.WriteString(string(r[position:]))
		out[i] = value.String()
	}
	token.Morphology = out
	return true, nil
}

// expandReplacement implements Python's numeric group references and standard
// replacement escapes. Unknown alphabetic escapes and named group references
// are explicit errors rather than silently changing replacement semantics.
func expandReplacement(replacement string, match pyregex.Match) (string, error) {
	r := []rune(replacement)
	var out strings.Builder
	for i := 0; i < len(r); i++ {
		if r[i] != '\\' {
			out.WriteRune(r[i])
			continue
		}
		i++
		if i == len(r) {
			return "", fmt.Errorf("trailing backslash in replacement")
		}
		c := r[i]
		if c >= '1' && c <= '9' {
			group := int(c - '0')
			if i+1 < len(r) && r[i+1] >= '0' && r[i+1] <= '9' {
				i++
				group = group*10 + int(r[i]-'0')
			}
			if group >= len(match.Spans) {
				return "", fmt.Errorf("invalid replacement group %d", group)
			}
			out.WriteString(match.Group(group))
			continue
		}
		if c == 'g' {
			if i+1 >= len(r) || r[i+1] != '<' {
				return "", fmt.Errorf("invalid replacement group reference")
			}
			start := i + 2
			end := start
			for end < len(r) && r[end] != '>' {
				end++
			}
			if end == len(r) {
				return "", fmt.Errorf("unterminated replacement group reference")
			}
			group, err := strconv.Atoi(string(r[start:end]))
			if err != nil || group < 0 || group >= len(match.Spans) {
				return "", fmt.Errorf("unsupported replacement group %q", string(r[start:end]))
			}
			out.WriteString(match.Group(group))
			i = end
			continue
		}
		if c >= '0' && c <= '7' {
			n := int(c - '0')
			for count := 1; count < 3 && i+1 < len(r) && r[i+1] >= '0' && r[i+1] <= '7'; count++ {
				i++
				n = n*8 + int(r[i]-'0')
			}
			out.WriteRune(rune(n))
			continue
		}
		switch c {
		case '\\':
			out.WriteRune('\\')
		case 'n':
			out.WriteRune('\n')
		case 'r':
			out.WriteRune('\r')
		case 't':
			out.WriteRune('\t')
		case 'f':
			out.WriteRune('\f')
		case 'v':
			out.WriteRune('\v')
		case 'a':
			out.WriteRune('\a')
		case 'b':
			out.WriteRune('\b')
		default:
			if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
				return "", fmt.Errorf("unsupported replacement escape \\%c", c)
			}
			out.WriteRune('\\')
			out.WriteRune(c)
		}
	}
	return out.String(), nil
}
