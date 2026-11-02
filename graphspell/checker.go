// Package graphspell provides the French spelling checker.
// Ported from Grammalecte by Olivier R.; GNU GPL version 3.
package graphspell

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/dictionary"
	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/strtransform"
	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/tokenizer"
)

//go:embed data/default_suggestions.json
var suggestionData []byte

var defaultSuggestions = func() map[string]string {
	var result map[string]string
	if err := json.Unmarshal(suggestionData, &result); err != nil {
		panic(err)
	}
	return result
}()

// Checker combines a main dictionary with optional community and personal
// dictionaries. Its immutable configuration can be shared between goroutines.
type Checker struct{ dictionaries []*dictionary.Dictionary }

// New creates a checker. The main dictionary is required; nil supplementary
// dictionaries are disabled. Dictionary order determines morphology order.
func New(main, community, personal *dictionary.Dictionary) (*Checker, error) {
	if main == nil {
		return nil, errors.New("main dictionary is required")
	}
	c := &Checker{dictionaries: []*dictionary.Dictionary{main}}
	for _, d := range []*dictionary.Dictionary{community, personal} {
		if d != nil {
			c.dictionaries = append(c.dictionaries, d)
		}
	}
	return c, nil
}

// Default uses the embedded French dictionary containing every spelling variant.
func Default() (*Checker, error) {
	d, err := dictionary.Default()
	if err != nil {
		return nil, err
	}
	return New(d, nil, nil)
}

// Lookup checks the exact spelling without case or token normalization.
func (c *Checker) Lookup(word string) bool {
	for _, d := range c.dictionaries {
		if d.Lookup(word) {
			return true
		}
	}
	return false
}

// IsValid checks normalized spelling and accepted case variants.
func (c *Checker) IsValid(word string) bool {
	for _, d := range c.dictionaries {
		if d.IsValid(word) {
			return true
		}
	}
	return false
}

// IsValidToken also checks hyphenated words. As in Python, all parts of a
// compound must be accepted by one of the dictionaries.
func (c *Checker) IsValidToken(word string) bool {
	for _, d := range c.dictionaries {
		if d.IsValidToken(word) {
			return true
		}
	}
	return false
}

// Morph returns all morphology entries in dictionary order, including duplicates.
func (c *Checker) Morph(word string) []string {
	out := []string{}
	for _, d := range c.dictionaries {
		out = append(out, d.Morph(word)...)
	}
	return out
}

// Lemmas returns unique lemmas in first-seen order. Python exposes an unordered set.
func (c *Checker) Lemmas(word string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, m := range c.Morph(word) {
		if end := strings.IndexByte(m, '/'); end > 0 {
			lemma := m[1:end]
			if !seen[lemma] {
				seen[lemma] = true
				out = append(out, lemma)
			}
		}
	}
	return out
}

// ParseParagraph returns misspelled WORD tokens, with Unicode code-point offsets.
func (c *Checker) ParseParagraph(input string) []tokenizer.Token {
	out := []tokenizer.Token{}
	for _, t := range tokenizer.Tokenize(input) {
		if t.Type == "WORD" && !c.IsValidToken(t.Value) {
			out = append(out, t)
		}
	}
	return out
}

// CountWords counts word tokens or their distinct lemmas. When onlyUnknown is
// true, literal unknown tokens are counted regardless of byLemma.
func (c *Checker) CountWords(input string, byLemma, onlyUnknown bool) map[string]int {
	out := map[string]int{}
	for _, t := range tokenizer.Tokenize(input) {
		if !strings.HasPrefix(t.Type, "WORD") {
			continue
		}
		if onlyUnknown {
			if !c.IsValidToken(t.Value) {
				out[t.Value]++
			}
		} else if byLemma {
			for _, lemma := range c.Lemmas(t.Value) {
				out[lemma]++
			}
		} else {
			out[t.Value]++
		}
	}
	return out
}

var elisionMorphology = map[string]*regexp.Regexp{
	"d": regexp.MustCompile(`:[YNAW]`), "n": regexp.MustCompile(`:V`), "m": regexp.MustCompile(`:V`), "t": regexp.MustCompile(`:V`), "s": regexp.MustCompile(`:V`), "j": regexp.MustCompile(`:(?:Y|[123][sp])`), "c": regexp.MustCompile(`:3[sp]`),
}

func (c *Checker) validSuggestion(word string) bool {
	if strings.HasSuffix(word, "è") || strings.HasSuffix(word, "È") {
		return false
	}
	r := []rune(word)
	if len(r) >= 2 && r[1] == '’' {
		if pattern := elisionMorphology[strtransform.Lower(string(r[0]))]; pattern != nil {
			for _, morph := range c.Morph(string(r[2:])) {
				if pattern.MatchString(morph) {
					return true
				}
			}
			return false
		}
	}
	return true
}

// Suggest returns one suggestion group per active dictionary. The main group
// applies French abbreviation and elision rules; supplementary groups use the
// smaller limit of the reference implementation. Cancellation is propagated.
func (c *Checker) Suggest(ctx context.Context, word string, limit int) ([][]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	groups := [][]string{}
	suggestions, exact := defaultSuggestions[word]
	lowerSuggestions, lowerFound := defaultSuggestions[strtransform.Lower(word)]
	var main []string
	switch {
	case exact:
		main = strings.Split(suggestions, "|")
	case lowerFound && strtransform.IsTitle(word):
		main = strings.Split(lowerSuggestions, "|")
		for i, s := range main {
			r := []rune(s)
			if len(r) > 0 {
				main[i] = strtransform.Upper(string(r[0])) + string(r[1:])
			}
		}
	case lowerFound && strtransform.IsUpper(word):
		main = strings.Split(lowerSuggestions, "|")
		for i, s := range main {
			main[i] = strtransform.Upper(s)
		}
	default:
		candidates, err := c.dictionaries[0].SuggestContext(ctx, word, limit, true)
		if err != nil {
			return nil, err
		}
		main = []string{}
		for _, s := range candidates {
			if c.validSuggestion(s) {
				main = append(main, s)
			}
		}
	}
	groups = append(groups, main)
	for _, d := range c.dictionaries[1:] {
		suggestions, err := d.SuggestContext(ctx, word, limit/2+1, false)
		if err != nil {
			return nil, err
		}
		groups = append(groups, suggestions)
	}
	return groups, nil
}
