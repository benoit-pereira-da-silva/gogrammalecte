// Ported from Grammalecte metagraphe.py, by Olivier R., 2013.
// Distributed with the French lexical generator under the Mozilla Public License 2.0.

package lexicon

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/strtransform"
)

type phoneticRule struct {
	pattern, before, after *regexp.Regexp
	negativeBefore         bool
	replacement            string
	endAnchor              bool
}

// phoneticRewrite handles the source's limited lookarounds explicitly. All
// consuming matches and context checks use the standard regexp implementation.
func phoneticRewrite(pattern, replacement string) phoneticRule {
	rule := phoneticRule{replacement: replacement}
	if strings.HasPrefix(pattern, "(?<=") || strings.HasPrefix(pattern, "(?<!") {
		rule.negativeBefore = strings.HasPrefix(pattern, "(?<!")
		end := strings.IndexByte(pattern, ')')
		rule.before = regexp.MustCompile("(?:" + pattern[4:end] + ")$")
		pattern = pattern[end+1:]
	}
	if start := strings.Index(pattern, "(?="); start >= 0 {
		rule.after = regexp.MustCompile("^(?:" + pattern[start+3:len(pattern)-1] + ")")
		pattern = pattern[:start]
	}
	rule.endAnchor = strings.HasSuffix(pattern, "$")
	rule.pattern = regexp.MustCompile(pattern)
	return rule
}

func (r phoneticRule) apply(word string) string {
	text := word
	// Python's dollar anchor also matches before a final newline.
	if r.endAnchor {
		text = strings.TrimSuffix(text, "\n")
	}
	indices := r.pattern.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		return word
	}
	var result strings.Builder
	last := 0
	for _, bounds := range indices {
		if r.before != nil {
			match := r.before.MatchString(text[:bounds[0]])
			if match == r.negativeBefore {
				continue
			}
		}
		if r.after != nil && !r.after.MatchString(text[bounds[1]:]) {
			continue
		}
		result.WriteString(word[last:bounds[0]])
		result.WriteString(r.replacement)
		last = bounds[1]
	}
	if last == 0 {
		return word
	}
	result.WriteString(word[last:])
	return result.String()
}

func phoneticRules(pairs ...string) []phoneticRule {
	rules := make([]phoneticRule, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		rules = append(rules, phoneticRewrite(pairs[i], pairs[i+1]))
	}
	return rules
}

var phoneticCharacters = strings.NewReplacer("à", "a", "â", "a", "ä", "a", "å", "a", "ā", "a", "ç", "S", "è", "é", "ê", "é", "ë", "é", "ē", "é", "î", "i", "ï", "i", "ī", "i", "ñ", "ni", "ô", "o", "ö", "o", "ō", "o", "ù", "u", "û", "u", "ü", "u", "ū", "u", "ÿ", "i", "æ", "é")
var phoneticAbbreviation = regexp.MustCompile("^[A-Z0-9]{2,}$")
var earlyPhoneticRules = phoneticRules(
	"sc(?=[eéiy])", "S", "x[cs](?=[eéiy])", "kS", "c(?=[eéiy])", "S", "c(?=[auoœ])", "k",
	"ge(?=[ao])", "j", "g(?=[ieéy])", "j", "gue", "ge", "am(?=[bp])", "â", "om(?=[bpt])", "ô", "omp(?=t)", "ô",
	"[iy]m(?=[bp])", "1", "eill", "éY", "aill", "aY", "[ae]y(?=[aeéi])", "éY", "^s", "S", "(?<=[aeioéu])s(?=[aeioéu])", "z")
var verbPhoneticRules = phoneticRules(
	"aient$", "é", "ent$", "", "ant$", "â", "a[st]$", "a", "ai[ets]$", "é", "ez$", "é", "on[st]?$", "ô", "es?$", "",
	"ée?s?$", "é", "ie?s?$", "i", "ue?s?$", "u", "ui[ts]$", "ui", "aut$", "o", "ut$", "u", "it?s?$", "i")
var pluralPhoneticRule = phoneticRewrite("[sx]$", "")
var latePhoneticRules = phoneticRules(
	"tion$", "Siô", "ent$", "â", "eux$", "2", "eaux$", "o", "e[rt]$", "é", "[ea]ine$", "én", "at$", "a",
	"on(?=[bcdfghjklmpqrsStvwxz])", "ô", "an[st]?$", "â", "an(?=[bcdfghjklmpqrsStvwxz])", "â", "iens?$", "i1", "(?<!i)ens?$", "â",
	"en(?=[bcdfghjklmpqrsStvwxz])", "â", "uns?$", "1", "un(?=[bcdfghjklmpqrsStvwxz])", "1", "ins?$", "1", "a?in(?=[bcdfghjklmpqrsStvwxz])", "1",
	"œu?", "2", "eux?$", "2", "eu(?=[bcdfghjklmpqrsStvwxz])", "2", "e(?=ss|tt|rr|ll|ff)", "é", "enne$", "én", "e?au", "o", "che$", "ç", "[ae]i", "é")
var finalPhoneticRule = phoneticRewrite("[qc]u?", "k")
var finalES = phoneticRewrite("es$", "")
var finalE = phoneticRewrite("e$", "")

// Metagraphe returns the source's simplified phonetic spelling and its currently
// empty secondary spelling. Morphology distinguishes verb endings and plurals.
func Metagraphe(word, morphology string) [2]string {
	if phoneticAbbreviation.MatchString(strings.TrimSuffix(word, "\n")) {
		return [2]string{word, ""}
	}
	length := utf8.RuneCountInString(word)
	word = phoneticCharacters.Replace(strtransform.Lower(word))
	if length == 1 {
		return [2]string{word, ""}
	}
	for _, rule := range earlyPhoneticRules {
		word = rule.apply(word)
	}
	if strings.HasPrefix(morphology, "v") {
		for _, rule := range verbPhoneticRules {
			word = rule.apply(word)
		}
	}
	if (strings.HasPrefix(morphology, "nom") || strings.HasPrefix(morphology, "adj")) && strings.Contains(morphology, "pl") {
		word = pluralPhoneticRule.apply(word)
	}
	for _, rule := range latePhoneticRules {
		word = rule.apply(word)
	}
	for _, pair := range [][2]string{{"ou", "û"}, {"ss", "S"}, {"dj", "j"}, {"th", "t"}} {
		word = strings.ReplaceAll(word, pair[0], pair[1])
	}
	word = finalPhoneticRule.apply(word)
	for _, pair := range [][2]string{{"ch", "k"}, {"ph", "f"}, {"h", ""}, {"y", "i"}, {"æ", "é"}, {"x", "ks"}} {
		word = strings.ReplaceAll(word, pair[0], pair[1])
	}
	if utf8.RuneCountInString(word) > 3 {
		word = finalES.apply(word)
	}
	if utf8.RuneCountInString(word) > 2 {
		word = finalE.apply(word)
	}
	runes := []rune(word)
	var result strings.Builder
	for i := 0; i < len(runes); i++ {
		result.WriteRune(runes[i])
		if i+1 < len(runes) && runes[i] == runes[i+1] && (runes[i] == '_' || unicode.IsLetter(runes[i]) || unicode.IsNumber(runes[i])) {
			i++
		}
	}
	return [2]string{result.String(), ""}
}
