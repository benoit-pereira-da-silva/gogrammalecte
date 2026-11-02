package french

// This is the experimental conjugation generator from Grammalecte's
// conj_generator.py (GPLv3). Its first- and second-group rules support novel
// infinitives; the source intentionally emits only an infinitive for other groups.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const DefaultVerbTag = "i_____a"

// ConjugationRule removes Cut Unicode characters before appending Addition.
// Pattern is an optional regular expression on the original infinitive.
type ConjugationRule struct {
	Cut      int
	Addition string
	Tags     string
	Pattern  string
}

func (r *ConjugationRule) UnmarshalJSON(data []byte) error {
	var fields []json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if len(fields) != 4 {
		return fmt.Errorf("invalid conjugation rule: expected 4 fields, got %d", len(fields))
	}
	for i, dest := range []any{&r.Cut, &r.Addition, &r.Tags} {
		if err := json.Unmarshal(fields[i], dest); err != nil {
			return err
		}
	}
	if r.Cut < 0 {
		return fmt.Errorf("negative conjugation cut: %d", r.Cut)
	}
	r.Pattern = ""
	if !bytes.Equal(fields[3], []byte("false")) {
		return json.Unmarshal(fields[3], &r.Pattern)
	}
	return nil
}
func (r ConjugationRule) MarshalJSON() ([]byte, error) {
	var pattern any = false
	if r.Pattern != "" {
		pattern = r.Pattern
	}
	return json.Marshal([]any{r.Cut, r.Addition, r.Tags, pattern})
}

type ConjugatedForm struct {
	Word string
	Tags string
}

var generationData struct {
	First             map[string]json.RawMessage   `json:"V1"`
	Second            []ConjugationRule            `json:"V2"`
	FirstParticiples  map[string][]ConjugationRule `json:"V1_ppas"`
	SecondParticiples map[string][]ConjugationRule `json:"V2_ppas"`
}
var firstGenerationRules map[string][]ConjugationRule
var generationPatterns map[string]*regexp.Regexp
var loadConjugationGeneration = sync.OnceFunc(func() {
	loadResource("conj_generator", &generationData)
	firstGenerationRules = make(map[string][]ConjugationRule)
	generationPatterns = make(map[string]*regexp.Regexp)
	for ending, raw := range generationData.First {
		if bytes.HasPrefix(bytes.TrimSpace(raw), []byte("{")) {
			var variants map[string]json.RawMessage
			if err := json.Unmarshal(raw, &variants); err != nil {
				panic(err)
			}
			raw = variants["1"]
		}
		var rules []ConjugationRule
		if err := json.Unmarshal(raw, &rules); err != nil {
			panic(err)
		}
		firstGenerationRules[ending] = rules
		for _, r := range rules {
			if r.Pattern != "" {
				generationPatterns[r.Pattern] = regexp.MustCompile(r.Pattern)
			}
		}
	}
})

// GuessVerbGroup reproduces the reference suffix heuristic, which does not
// distinguish irregular -ir verbs from regular second-group verbs.
func GuessVerbGroup(verb string) string {
	verb = strings.ToLower(verb)
	if strings.HasSuffix(verb, "er") {
		return "1"
	}
	if strings.HasSuffix(verb, "ir") {
		return "2"
	}
	if verb == "être" || verb == "avoir" {
		return "0"
	}
	if strings.HasSuffix(verb, "re") {
		return "3"
	}
	return "4"
}

// ConjugationRules returns an independent rule slice. The reference default
// groupLimit is 2; values above 2 suppress the regular -ir template.
func ConjugationRules(verb string, variableParticiple bool, groupLimit int) []ConjugationRule {
	loadConjugationGeneration()
	var out []ConjugationRule
	if strings.HasSuffix(verb, "er") {
		r := []rune(verb)
		chosen := firstGenerationRules["er"]
		for _, length := range []int{5, 4, 3} {
			start := len(r) - length
			if start < 0 {
				start = 0
			}
			if rules, ok := firstGenerationRules[string(r[start:])]; ok {
				chosen = rules
				break
			}
		}
		out = append(out, chosen...)
		out = append(out, generationData.FirstParticiples[strconv.FormatBool(variableParticiple)]...)
	} else if strings.HasSuffix(verb, "ir") && groupLimit <= 2 {
		out = append(out, generationData.Second...)
		out = append(out, generationData.SecondParticiples[strconv.FormatBool(variableParticiple)]...)
	} else {
		out = []ConjugationRule{{Tags: ":Y/*"}}
	}
	return out
}

// GenerateConjugations generates tagged forms for an infinitive, including
// novel words. Use DefaultVerbTag for the reference default classification.
func GenerateConjugations(verb, verbTag string, variableParticiple bool) []ConjugatedForm {
	rules := ConjugationRules(verb, variableParticiple, 2)
	group := GuessVerbGroup(verb)
	r := []rune(verb)
	out := make([]ConjugatedForm, 0, len(rules))
	for _, rule := range rules {
		if rule.Pattern != "" && !generationPatterns[rule.Pattern].MatchString(verb) {
			continue
		}
		end := len(r) - rule.Cut
		if end < 0 {
			end = 0
		}
		out = append(out, ConjugatedForm{string(r[:end]) + rule.Addition, ":V" + group + "_" + verbTag + rule.Tags})
	}
	return out
}
