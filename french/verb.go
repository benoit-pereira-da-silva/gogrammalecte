package french

import (
	"fmt"
	"strings"
)

// VerbOptions selects the grammatical variants used in a conjugation table.
type VerbOptions struct {
	Pronominal    bool
	Negative      bool
	Compound      bool
	Interrogative bool
	Feminine      bool
}

// Verb contains an infinitive, its classification and its inflection pattern.
// Methods do not mutate the verb and may be called concurrently.
type Verb struct {
	InfinitiveWord  string
	Auxiliary       string
	RawInfo         string
	Info            string
	Group           string
	Incomplete      bool
	PronominalLabel string
	Pronominal      int
	withEn          bool
	tags, auxTags   []int
	forms           map[string]map[string]string
}

var subjects = map[string]string{":1s": "je", ":1ś": "je", ":2s": "tu", ":3s": "il", ":1p": "nous", ":2p": "vous", ":3p": "ils"}
var objects = map[string]string{":1s": "me ", ":1ś": "me ", ":2s": "te ", ":3s": "se ", ":1p": "nous ", ":2p": "vous ", ":3p": "se "}
var elidedObjects = map[string]string{":1s": "m’", ":1ś": "m’", ":2s": "t’", ":3s": "s’", ":1p": "nous ", ":2p": "vous ", ":3p": "s’"}
var imperativePronouns = map[string]string{":2s": "-toi", ":1p": "-nous", ":2p": "-vous"}
var imperativeNegative = map[string]string{":2s": "ne te ", ":1p": "ne nous ", ":2p": "ne vous "}
var imperativeEn = map[string]string{":2s": "-t’en", ":1p": "-nous-en", ":2p": "-vous-en"}
var imperativeNegativeEn = map[string]string{":2s": "ne t’en ", ":1p": "ne nous en ", ":2p": "ne vous en "}
var groupLabels = map[byte]string{'0': "auxiliaire", '1': "1ᵉʳ groupe", '2': "2ᵉ groupe", '3': "3ᵉ groupe"}

// NewVerb uses verb's own pattern unless pattern names another reference verb.
// Supplying a pattern supports conjugating a novel infinitive with known rules.
func NewVerb(verb, pattern string) (*Verb, error) {
	if verb == "" {
		return nil, fmt.Errorf("%w: empty infinitive", ErrUnknownVerb)
	}
	if pattern == "" {
		pattern = verb
	}
	raw, ok := VerbType(pattern)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownVerb, pattern)
	}
	tags, _ := verbTags(pattern)
	v := &Verb{InfinitiveWord: verb, RawInfo: raw, Group: raw[:1], Incomplete: strings.HasSuffix(raw, "zz"), PronominalLabel: "pronominal", Pronominal: -1, tags: tags, forms: map[string]map[string]string{}}
	v.Auxiliary = "avoir"
	if raw[7] == 'e' {
		v.Auxiliary = "être"
	}
	v.auxTags, _ = verbTags(v.Auxiliary)
	v.withEn = raw[5] == 'e'
	switch raw[5] {
	case 'q', 'u', 'v', 'e':
		v.Pronominal = 0
	case 'p', 'r':
		v.Pronominal = 1
	case 'x':
		v.Pronominal = 2
		v.PronominalLabel = "cas particuliers"
	case '_':
	default:
		v.PronominalLabel = "# erreur #"
	}
	info := ""
	switch {
	case raw[3] == 't':
		info = "transitif"
	case raw[4] == 'n':
		info = "transitif indirect"
	case raw[2] == 'i':
		info = "intransitif"
	case raw[5] == 'r':
		info = "pronominal réciproque"
	case raw[5] == 'p':
		info = "pronominal"
	}
	if strings.ContainsRune("quve", rune(raw[5])) {
		info += " (+ usage pronominal)"
	}
	if raw[6] == 'm' {
		info += " impersonnel"
	}
	if info == "" {
		info = "# erreur - code : " + raw
	}
	v.Info = groupLabels[raw[0]] + " · " + info
	for tense := range tenseIndex {
		v.forms[tense] = map[string]string{}
		for person := range conjugation.Patterns[tense][tags[tenseIndex[tense]]] {
			v.forms[tense][person] = mustConjugate(verb, tags, tense, person)
		}
	}
	return v, nil
}

func startsVowel(s string) bool {
	for _, r := range s {
		return strings.ContainsRune("aeéiouœê", r)
	}
	return false
}
func (v *Verb) participle(o VerbOptions, plural bool) string {
	base := v.forms[":Q"][":m:s"]
	if !o.Pronominal && v.Auxiliary == "avoir" {
		return base
	}
	person := ":m:s"
	if o.Feminine && plural {
		person = ":f:p"
	} else if o.Feminine {
		person = ":f:s"
	} else if plural {
		person = ":m:p"
	}
	if form := v.forms[":Q"][person]; form != "" {
		return form
	}
	return base
}

// Infinitive formats a simple or compound infinitive with the selected variants.
func (v *Verb) Infinitive(o VerbOptions) string {
	s := v.InfinitiveWord
	if o.Compound {
		s = v.Auxiliary
		if o.Pronominal {
			s = "être"
		}
	}
	if o.Pronominal {
		if v.withEn {
			s = "s’en " + s
		} else if startsVowel(s) {
			s = "s’" + s
		} else {
			s = "se " + s
		}
	}
	if o.Negative {
		s = "ne pas " + s
	}
	if o.Compound {
		s += " " + v.participle(o, v.RawInfo[5] == 'r')
	}
	if o.Interrogative {
		s += " … ?"
	}
	return s
}

// PastParticiple returns the bare participle for a gender/number identifier.
func (v *Verb) PastParticiple(person string) (string, error) {
	form, ok := v.forms[":Q"][person]
	if !ok {
		return "", fmt.Errorf("unknown participle person: %s", person)
	}
	return form, nil
}

// PresentParticiple formats the present participle and its compound variants.
func (v *Verb) PresentParticiple(o VerbOptions) string {
	s := v.forms[":P"][":P"]
	if s == "" {
		return ""
	}
	if o.Compound {
		if o.Pronominal {
			s, _ = GetConjugation("être", ":P", ":P")
		} else {
			s = mustConjugate(v.Auxiliary, v.auxTags, ":P", ":P")
		}
	}
	if s == "" {
		return ""
	}
	elide := startsVowel(s)
	if o.Pronominal {
		if v.withEn {
			s = "s’en " + s
		} else if elide {
			s = "s’" + s
		} else {
			s = "se " + s
		}
	}
	if o.Negative {
		if elide && !o.Pronominal {
			s = "n’" + s + " pas"
		} else {
			s = "ne " + s + " pas"
		}
	}
	if o.Compound {
		s += " " + v.participle(o, v.RawInfo[5] == 'r')
	}
	if o.Interrogative {
		s += " … ?"
	}
	return s
}
func (v *Verb) subject(person string, feminine bool) string {
	if person == ":3s" {
		if v.RawInfo[5] == 'r' {
			return "on"
		}
		if feminine {
			return "elle"
		}
	}
	if person == ":3p" && feminine {
		return "elles"
	}
	return subjects[person]
}

// Conjugate formats a finite form, including pronouns, negation and inversion.
func (v *Verb) Conjugate(tense, person string, o VerbOptions) (string, error) {
	forms, ok := v.forms[tense]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrUnknownTense, tense)
	}
	if _, ok = subjects[person]; !ok {
		return "", fmt.Errorf("unknown person: %s", person)
	}
	s, ok := forms[person]
	if !ok {
		return "", fmt.Errorf("person %s is not defined for %s", person, tense)
	}
	if s == "" {
		return "", nil
	}
	if !o.Compound && o.Interrogative && person == ":1s" && forms[":1ś"] != "" {
		person = ":1ś"
	}
	if o.Compound {
		if o.Pronominal {
			s, _ = GetConjugation("être", tense, person)
		} else {
			s = mustConjugate(v.Auxiliary, v.auxTags, tense, person)
		}
	} else {
		s = forms[person]
	}
	if s == "" {
		return "", nil
	}
	elide := startsVowel(s)
	if o.Pronominal {
		if v.withEn {
			s = elidedObjects[person] + "en " + s
		} else if elide {
			s = elidedObjects[person] + s
		} else {
			s = objects[person] + s
		}
	}
	if o.Negative {
		if elide && !o.Pronominal {
			s = "n’" + s
		} else {
			s = "ne " + s
		}
	}
	if o.Interrogative {
		if person == ":3s" && !strings.HasSuffix(s, "t") && !strings.HasSuffix(s, "d") && !strings.HasSuffix(s, "c") {
			s += "-t"
		}
		s += "-" + v.subject(person, o.Feminine)
	} else if person == ":1s" && elide && !o.Negative && !o.Pronominal {
		s = "j’" + s
	} else {
		s = v.subject(person, o.Feminine) + " " + s
	}
	if o.Negative {
		s += " pas"
	}
	if o.Compound {
		s += " " + v.participle(o, strings.HasSuffix(person, "p") || v.RawInfo[5] == 'r')
	}
	if o.Interrogative {
		s += " … ?"
	}
	return s, nil
}

// Imperative formats the second-person singular or first/second-person plural imperative.
func (v *Verb) Imperative(person string, o VerbOptions) (string, error) {
	if _, ok := imperativePronouns[person]; !ok {
		return "", fmt.Errorf("unknown imperative person: %s", person)
	}
	s := v.forms[":E"][person]
	if s == "" {
		return "", nil
	}
	if o.Compound {
		if o.Pronominal {
			s, _ = GetConjugation("être", ":E", person)
		} else {
			s = mustConjugate(v.Auxiliary, v.auxTags, ":E", person)
		}
	}
	if s == "" {
		return "", nil
	}
	elide := startsVowel(s)
	if o.Negative {
		if o.Pronominal {
			if v.withEn {
				s = imperativeNegativeEn[person] + s + " pas"
			} else if elide && person == ":2s" {
				s = "ne t’" + s + " pas"
			} else {
				s = imperativeNegative[person] + s + " pas"
			}
		} else if elide {
			s = "n’" + s + " pas"
		} else {
			s = "ne " + s + " pas"
		}
	} else if o.Pronominal {
		if v.withEn {
			s += imperativeEn[person]
		} else {
			s += imperativePronouns[person]
		}
	}
	if o.Compound {
		s += " " + v.participle(o, strings.HasSuffix(person, "p") || v.RawInfo[5] == 'r')
	}
	return s, nil
}

// Table returns the complete upstream display table. Labels and keys retain
// the reference format; returned maps are independent of the verb's state.
func (v *Verb) Table(o VerbOptions) map[string]string {
	table := map[string]string{"t_infi": "Infinitif", "infi": v.Infinitive(o), "t_ppre": "Participe présent", "ppre": v.PresentParticiple(o), "t_ppas": "Participes passés", "t_imp": "Impératif", "t_indi": "Indicatif", "t_cond": "Conditionnel", "t_subj": "Subjonctif"}
	for i, p := range []string{":m:s", ":f:s", ":m:p", ":f:p"} {
		s, e := v.PastParticiple(p)
		if e != nil {
			s = "# erreur"
		}
		table[fmt.Sprintf("ppas%d", i+1)] = s
	}
	table["t_impe"] = "Présent"
	if o.Compound {
		table["t_impe"] = "Passé"
	}
	if o.Interrogative {
		table["t_impe"] = ""
	}
	for i, p := range []string{":2s", ":1p", ":2p"} {
		s := ""
		if !o.Interrogative {
			s, _ = v.Imperative(p, o)
		}
		table[fmt.Sprintf("impe%d", i+1)] = s
	}
	groups := [][5]string{{"ipre", ":Ip", "Présent", "Passé composé", ""}, {"iimp", ":Iq", "Imparfait", "Plus-que-parfait", ""}, {"ipsi", ":Is", "Passé simple", "Passé antérieur", ""}, {"ifut", ":If", "Futur", "Futur antérieur", ""}, {"conda", ":K", "Présent", "Passé (1ʳᵉ forme)", ""}, {"condb", ":Sq", "", "Passé (2ᵉ forme)", "compound"}, {"spre", ":Sp", "Présent", "Passé", "subj"}, {"simp", ":Sq", "Imparfait", "Plus-que-parfait", "subj"}}
	for _, g := range groups {
		enabled := !(g[4] == "compound" && !o.Compound || g[4] == "subj" && o.Interrogative)
		label := g[2]
		if o.Compound {
			label = g[3]
		}
		if !enabled {
			label = ""
		}
		table["t_"+g[0]] = label
		for i, p := range []string{":1s", ":2s", ":3s", ":1p", ":2p", ":3p"} {
			s := ""
			if enabled {
				var e error
				s, e = v.Conjugate(g[1], p, o)
				if e != nil {
					s = "# erreur"
				}
			}
			table[fmt.Sprintf("%s%d", g[0], i+1)] = s
		}
	}
	return table
}
