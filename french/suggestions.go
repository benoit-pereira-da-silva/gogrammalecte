package french

// Grammatical suggestion algorithms ported from Grammalecte's
// gce_suggestions.py by Olivier R. (GPLv3).

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SuggestionLexicon supplies morphology and spelling validation. Both the
// embedded dictionary and the multi-dictionary spell checker implement it.
type SuggestionLexicon interface {
	Morph(string) []string
	IsValid(string) bool
}

// Suggester creates the pipe-separated replacements expected by grammar rules.
// It preserves source ordering except for Python's unordered lemma sets, which
// are traversed in the dictionary's first-seen order.
type Suggester struct{ lexicon SuggestionLexicon }

func NewSuggester(lexicon SuggestionLexicon) *Suggester { return &Suggester{lexicon: lexicon} }

type orderedSuggestions struct {
	values []string
	seen   map[string]bool
}

func (s *orderedSuggestions) add(value string) {
	if s.seen == nil {
		s.seen = make(map[string]bool)
	}
	if !s.seen[value] {
		s.seen[value] = true
		s.values = append(s.values, value)
	}
}
func (s *orderedSuggestions) join(suffix string) string {
	if suffix == "" {
		return strings.Join(s.values, "|")
	}
	out := make([]string, len(s.values))
	for i, v := range s.values {
		out[i] = JoinVerbAndSuffix(v, suffix)
	}
	return strings.Join(out, "|")
}
func (s *Suggester) lemmas(word string) []string {
	var out orderedSuggestions
	for _, m := range s.lexicon.Morph(word) {
		if end := strings.IndexByte(m, '/'); end > 0 {
			out.add(m[1:end])
		}
	}
	return out.values
}

var suggestionTense = regexp.MustCompile(`:(?:Y|I[pqsf]|S[pq]|K|P)`)
var suggestionPerson = regexp.MustCompile(`:(?:[123][sp]|P|Y)`)

// Verb changes the person while retaining the source word's possible tenses.
// Optional additional callbacks can close over any extra grammar-rule inputs.
func (s *Suggester) Verb(word, person string, compound bool, extra ...func(string) string) string {
	suffix := ""
	if compound {
		word, suffix = SplitVerb(word)
	}
	var out orderedSuggestions
	for _, stem := range s.lemmas(word) {
		tags, ok := verbTags(stem)
		if !ok {
			continue
		}
		var tenses orderedSuggestions
		pattern := regexp.MustCompile(`>` + regexp.QuoteMeta(stem) + `/.*?(:(?:Y|I[pqsf]|S[pq]|K|P|Q))`)
		for _, m := range s.lexicon.Morph(word) {
			for _, match := range pattern.FindAllStringSubmatch(m, -1) {
				switch match[1] {
				case ":Y", ":Q":
					tenses.add(":Ip")
					tenses.add(":Iq")
					tenses.add(":Is")
				case ":P":
					tenses.add(":Ip")
				default:
					tenses.add(match[1])
				}
			}
		}
		for _, tense := range tenses.values {
			if person == ":1ś" && !hasTags(tags, tense, person) {
				person = ":1s"
			}
			if hasTags(tags, tense, person) {
				out.add(mustConjugate(stem, tags, tense, person))
			}
		}
	}
	for _, fn := range extra {
		if value := fn(word); value != "" {
			out.add(value)
		}
	}
	return out.join(suffix)
}

// PastParticiple selects gender and number using a compact morphology tag.
func (s *Suggester) PastParticiple(word, pattern string) string {
	var out orderedSuggestions
	for _, stem := range s.lemmas(word) {
		tags, ok := verbTags(stem)
		if !ok {
			continue
		}
		add := func(person string) {
			if v := mustConjugate(stem, tags, ":Q", person); v != "" {
				out.add(v)
			}
		}
		fallback := func(person string) {
			if hasTags(tags, ":Q", person) {
				add(person)
			} else {
				add(":m:s")
			}
		}
		switch pattern {
		case "":
			for _, p := range []string{":m:s", ":m:p", ":f:s", ":f:p"} {
				add(p)
			}
		case ":m:p", ":f:s", ":f:p":
			fallback(pattern)
		case ":s":
			add(":m:s")
			add(":f:s")
		case ":p":
			fallback(":m:p")
			add(":f:p")
		default:
			add(":m:s")
		}
	}
	return out.join("")
}
func validateSuggestionTense(tense string) error {
	if _, ok := tenseIndex[tense]; !ok && tense != ":Y" {
		return fmt.Errorf("%w: %s", ErrUnknownTense, tense)
	}
	return nil
}

func (s *Suggester) VerbTense(word, tense, person string) (string, error) {
	if err := validateSuggestionTense(tense); err != nil {
		return "", err
	}
	var out orderedSuggestions
	for _, stem := range s.lemmas(word) {
		if tags, ok := verbTags(stem); ok && hasTags(tags, tense, person) {
			out.add(mustConjugate(stem, tags, tense, person))
		}
	}
	return out.join(""), nil
}

// VerbFrom conjugates an infinitive using the analyses of another word.
func (s *Suggester) VerbFrom(stem, word, person string) string {
	tags, ok := verbTags(stem)
	if !ok {
		return ""
	}
	var out orderedSuggestions
	for _, m := range s.lexicon.Morph(word) {
		people := []string{person}
		if person == "" {
			people = suggestionPerson.FindAllString(m, -1)
		}
		for _, tense := range suggestionTense.FindAllString(m, -1) {
			for _, p := range people {
				if hasTags(tags, tense, p) {
					out.add(mustConjugate(stem, tags, tense, p))
				}
			}
		}
	}
	return out.join("")
}

func (s *Suggester) Imperative(word string, compound bool) string {
	suffix := ""
	if compound {
		word, suffix = SplitVerb(word)
	}
	var out []string
	for _, stem := range s.lemmas(word) {
		if tags, ok := verbTags(stem); ok {
			for _, person := range []string{":2s", ":1p", ":2p"} {
				if hasTags(tags, ":E", person) {
					out = append(out, JoinVerbAndSuffix(mustConjugate(stem, tags, ":E", person), suffix))
				}
			}
		}
	}
	return strings.Join(out, "|")
}

func (s *Suggester) Infinitives(word string) string {
	var out []string
	for _, stem := range s.lemmas(word) {
		if IsVerb(stem) {
			out = append(out, stem)
		}
	}
	return strings.Join(out, "|")
}

var subjectPersons = map[string]string{"je": ":1s", "j’": ":1s", "tu": ":2s", "il": ":3s", "on": ":3s", "elle": ":3s", "iel": ":3s", "ce": ":3s", "ça": ":3s", "cela": ":3s", "ceci": ":3s", "nous": ":1p", "vous": ":2p", "ils": ":3p", "elles": ":3p", "iels": ":3p"}
var modeReplacements = map[string]string{"es": "aies", "aies": "es", "est": "ait", "ait": "est"}

func (s *Suggester) VerbMode(word, mode, subject string) (string, error) {
	var tenses []string
	switch mode {
	case ":I":
		tenses = []string{":Ip", ":Iq", ":Is", ":If"}
	case ":S":
		tenses = []string{":Sp", ":Sq"}
	default:
		if !strings.HasPrefix(mode, ":I") && !strings.HasPrefix(mode, ":S") {
			return "", nil
		}
		tenses = []string{mode}
	}
	for _, tense := range tenses {
		if err := validateSuggestionTense(tense); err != nil {
			return "", err
		}
	}
	person, ok := subjectPersons[strings.ToLower(subject)]
	if !ok {
		person = subject
	}
	var out orderedSuggestions
	for _, stem := range s.lemmas(word) {
		if tags, ok := verbTags(stem); ok {
			for _, tense := range tenses {
				if hasTags(tags, tense, person) {
					out.add(mustConjugate(stem, tags, tense, person))
				}
			}
		}
	}
	if replacement, ok := modeReplacements[word]; ok {
		out.add(replacement)
	}
	return out.join(""), nil
}

func (s *Suggester) Plural(word string, self bool) string {
	var out []string
	add := func(value string) {
		if s.lexicon.IsValid(value) {
			out = append(out, value)
		}
	}
	r := []rune(word)
	for _, ending := range []string{"al", "ail", "AL", "AIL"} {
		if strings.HasSuffix(word, ending) && len(r) > len(ending) {
			tail := "ux"
			if ending == strings.ToUpper(ending) {
				tail = "UX"
			}
			add(string(r[:len(r)-len(ending)+1]) + tail)
		}
	}
	last, _ := utf8.DecodeLastRuneInString(word)
	if unicode.IsLower(last) {
		add(word + "s")
		add(word + "x")
	} else {
		add(word + "S")
		add(word + "X")
	}
	out = append(out, IrregularPlurals(word)...)
	if len(out) == 0 && self && strings.ContainsAny(string(last), "sxSX") {
		out = append(out, word)
	}
	return strings.Join(out, "|")
}

func (s *Suggester) Singular(word string, self bool) string {
	var out []string
	add := func(value string) {
		if s.lexicon.IsValid(value) {
			out = append(out, value)
		}
	}
	r := []rune(word)
	if strings.HasSuffix(word, "ux") {
		add(string(r[:len(r)-2]) + "l")
		add(string(r[:len(r)-2]) + "il")
	}
	if strings.HasSuffix(word, "UX") {
		add(string(r[:len(r)-2]) + "L")
		add(string(r[:len(r)-2]) + "IL")
	}
	if len(r) > 0 && strings.ContainsRune("sxSX", r[len(r)-1]) {
		add(string(r[:len(r)-1]))
	}
	if len(out) == 0 && self {
		out = append(out, word)
	}
	return strings.Join(out, "|")
}

func (s *Suggester) genderForm(word string, feminine, plural, similar bool) string {
	var out orderedSuggestions
	gender, number := ":m", ":s"
	if feminine {
		gender = ":f"
	}
	if plural {
		number = ":p"
	}
	for _, m := range s.lexicon.Morph(word) {
		stem := LemmaOfMorphology(m)
		if !strings.Contains(m, ":V") {
			if strings.Contains(m, gender) || strings.Contains(m, ":e") {
				if plural {
					out.add(s.Plural(word, false))
				} else {
					out.add(s.Singular(word, true))
				}
			} else if IsMasculineForm(stem) {
				if feminine {
					for _, f := range FeminineForms(stem, plural) {
						out.add(f)
					}
				} else if plural {
					out.add(s.Plural(stem, true))
				} else {
					out.add(stem)
				}
			}
		} else if tags, ok := verbTags(stem); ok {
			person := gender + number
			if !feminine && !plural {
				if hasTags(tags, ":Q", ":m:s") && hasTags(tags, ":Q", ":f:s") {
					out.add(mustConjugate(stem, tags, ":Q", ":m:s"))
				}
			} else if hasTags(tags, ":Q", person) {
				out.add(mustConjugate(stem, tags, ":Q", person))
			} else if !feminine && hasTags(tags, ":Q", ":m:s") {
				if form := mustConjugate(stem, tags, ":Q", ":m:s"); strings.HasSuffix(form, "s") {
					out.add(form)
				}
			}
		}
	}
	if similar {
		forms, _ := SelectPhoneticSimilar(word, gender+":["+number[1:]+"i]")
		for _, f := range forms {
			out.add(f)
		}
	}
	return out.join("")
}
func (s *Suggester) MasculineSingular(word string, similar bool) string {
	return s.genderForm(word, false, false, similar)
}
func (s *Suggester) MasculinePlural(word string, similar bool) string {
	return s.genderForm(word, false, true, similar)
}
func (s *Suggester) FeminineSingular(word string, similar bool) string {
	return s.genderForm(word, true, false, similar)
}
func (s *Suggester) FemininePlural(word string, similar bool) string {
	return s.genderForm(word, true, true, similar)
}

func (s *Suggester) agreeMorphologies(word string, morphologies []string) string {
	if len(morphologies) == 0 {
		return ""
	}
	gender, number := GenderNumber(morphologies)
	switch gender {
	case ":m", ":f":
		return s.genderForm(word, gender == ":f", number == ":p", false)
	case ":e":
		if number == ":s" {
			return s.Singular(word, true)
		}
		if number == ":p" {
			return s.Plural(word, false)
		}
		return word
	default:
		return ""
	}
}
func (s *Suggester) Agree(destination, source string) string {
	return s.agreeMorphologies(destination, s.lexicon.Morph(source))
}
func (s *Suggester) AgreeTokens(destination, source MorphToken) string {
	m := source.Morphologies
	if m == nil {
		m = s.lexicon.Morph(source.Value)
	}
	return s.agreeMorphologies(destination.Value, m)
}
func (s *Suggester) hasGender(word, gender string) bool {
	for _, stem := range s.lemmas(word) {
		if IsMasculineForm(stem) {
			return true
		}
		if tags, ok := verbTags(stem); ok && hasTags(tags, ":Q", gender+":s") {
			return true
		}
	}
	ok, _ := HasPhoneticSimilar(word, gender)
	return ok
}
func (s *Suggester) HasFeminineForm(word string) bool  { return s.hasGender(word, ":f") }
func (s *Suggester) HasMasculineForm(word string) bool { return s.hasGender(word, ":m") }

// SwitchGender preserves number when plural is nil; otherwise it requests the
// explicit singular/plural choice.
func (s *Suggester) SwitchGender(word string, plural *bool) string {
	var out orderedSuggestions
	for _, m := range s.lexicon.Morph(word) {
		feminine := false
		if strings.Contains(m, ":f") {
		} else if strings.Contains(m, ":m") {
			feminine = true
		} else {
			continue
		}
		if plural != nil {
			out.add(s.genderForm(word, feminine, *plural, false))
		} else if strings.Contains(m, ":s") {
			out.add(s.genderForm(word, feminine, false, false))
		} else if strings.Contains(m, ":p") {
			out.add(s.genderForm(word, feminine, true, false))
		} else {
			out.add(s.genderForm(word, feminine, false, false))
			out.add(s.genderForm(word, feminine, true, false))
		}
	}
	return out.join("")
}
func (s *Suggester) SwitchPlural(word string) string {
	var out orderedSuggestions
	for _, m := range s.lexicon.Morph(word) {
		if strings.Contains(m, ":s") {
			out.add(s.Plural(word, false))
		} else if strings.Contains(m, ":p") {
			out.add(s.Singular(word, true))
		}
	}
	return out.join("")
}

// Similar optionally substitutes deverbal nouns when no phonetic form matches.
// Unsupported regular-expression syntax is returned as an error.
func (s *Suggester) Similar(word, pattern string, substitute, compound bool) (string, error) {
	var filter *regexp.Regexp
	if pattern != "" {
		var err error
		filter, err = regexp.Compile(pattern)
		if err != nil {
			return "", err
		}
	}
	suffix := ""
	if compound {
		word, suffix = SplitVerb(word)
	}
	forms, err := SelectPhoneticSimilar(word, pattern)
	if err != nil {
		return "", err
	}
	var out orderedSuggestions
	for _, f := range forms {
		out.add(f)
	}
	if len(out.values) == 0 && substitute {
		for _, m := range s.lexicon.Morph(word) {
			if strings.Contains(m, ":V") {
				end := strings.IndexByte(m, '/')
				if end < 1 {
					continue
				}
				for _, name := range DerivedNames(m[1:end]) {
					if filter == nil {
						out.add(name)
					} else {
						for _, m2 := range s.lexicon.Morph(name) {
							if filter.MatchString(m2) {
								out.add(name)
								break
							}
						}
					}
				}
				break
			}
		}
	}
	return out.join(suffix), nil
}
func (s *Suggester) LesLa(word string) string {
	for _, m := range s.lexicon.Morph(word) {
		if strings.Contains(m, ":p") {
			return "les|la"
		}
	}
	return "la"
}
