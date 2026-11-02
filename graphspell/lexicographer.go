package graphspell

// French lexical labels and token decomposition are ported from
// graphspell/lexgraph_fr.py and spellchecker.py by Olivier R. (GPLv3).

import (
	"context"
	_ "embed"
	"encoding/json"
	"regexp"
	"strings"
	"sync"

	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/strtransform"
	"github.com/benoit-pereira-da-silva/gogrammalecte/internal/pyregex"
)

//go:embed data/lexicographer.json
var lexicalData []byte
var lexical struct {
	Tags     map[string][2]string
	Values   map[string]string
	Patterns map[string]string
}
var lexicalPatterns map[string]*pyregex.Regexp
var loadLexical = sync.OnceFunc(func() {
	if err := json.Unmarshal(lexicalData, &lexical); err != nil {
		panic(err)
	}
	lexicalPatterns = make(map[string]*pyregex.Regexp, len(lexical.Patterns))
	for name, pattern := range lexical.Patterns {
		r, err := pyregex.Compile(pattern)
		if err != nil {
			panic(err)
		}
		lexicalPatterns[name] = r
	}
})

func lexicalMatch(name, input string, anchored bool) *pyregex.Match {
	loadLexical()
	m, err := lexicalPatterns[name].Search(context.Background(), input)
	if err != nil {
		panic(err)
	}
	if m != nil && anchored && m.Start(0) != 0 {
		return nil
	}
	return m
}

// SplitWord returns an elided prefix, a stem and a pronominal suffix. The
// reference normalizes recognized prefix apostrophes to the typographic mark.
func SplitWord(word string) (prefix, stem, suffix string) {
	stem = word
	if m := lexicalMatch("_zElidedPrefix", stem, true); m != nil {
		prefix = m.Group(1) + "’"
		stem = m.Group(2)
	}
	if m := lexicalMatch("_zCompoundWord", stem, true); m != nil {
		stem = m.Group(1)
		suffix = m.Group(2)
	}
	return
}
func AnalyzeWord(word string) string { loadLexical(); return lexical.Values[strtransform.Lower(word)] }

var lexicalVerbFlags = regexp.MustCompile(`(V[0123][ea_])[itpqnmr_eaxz]+`)
var lexicalTag = regexp.MustCompile(`[:;/#][\p{L}\p{N}_@*!][^:;/#]*`)

func readableTag(tag string) string {
	if labels, ok := lexical.Tags[tag]; ok {
		return labels[0]
	}
	return " [" + tag + "]?"
}

// ReadableMorphology preserves the reference's human-readable French labels,
// including its legacy past-participle slicing behavior.
func ReadableMorphology(morphology string) string {
	if morphology == "" {
		return "mot inconnu"
	}
	loadLexical()
	verbType := ""
	if strings.Contains(morphology, ":V") {
		morphology = lexicalVerbFlags.ReplaceAllString(morphology, "${1}")
	}
	if strings.Contains(morphology, ":Q") {
		r := []rune(morphology)
		index := -1
		if pos := strings.Index(morphology, ":V"); pos >= 0 {
			index = len([]rune(morphology[:pos]))
		}
		start, end := index, index+4
		if start < 0 {
			start += len(r)
		}
		if end < 0 {
			end += len(r)
		}
		if start < 0 {
			start = 0
		}
		if end > len(r) {
			end = len(r)
		}
		if end >= start {
			verbType = string(r[start:end])
		}
		if len(r) > 4 {
			morphology = string(r[4:])
		} else {
			morphology = ""
		}
		morphology = strings.ReplaceAll(strings.ReplaceAll(morphology, ":1ŝ", ""), ":1ś", "")
	}
	var result strings.Builder
	for _, tag := range lexicalTag.FindAllString(morphology, -1) {
		result.WriteString(readableTag(tag))
	}
	text := result.String()
	if (strings.HasPrefix(text, " verbe") || strings.HasPrefix(text, " participe")) && !strings.HasSuffix(text, "infinitif") {
		r := []rune(morphology)
		end := -1
		for i, c := range r {
			if c == '/' {
				end = i
				break
			}
		}
		if end < 0 {
			end = len(r) - 1
		}
		stem := ""
		if end > 1 {
			stem = string(r[1:end])
		}
		if verbType != "" {
			text += " [" + stem + " : " + strings.TrimRight(readableTag(verbType), ",") + "]"
		} else {
			text += " [" + stem + "]"
		}
	}
	if text == "" {
		return " [" + morphology + "]: étiquettes inconnues"
	}
	return strings.TrimRight(text, ",")
}

type MorphologyLabel struct {
	Morphology string `json:"morphology"`
	Label      string `json:"label"`
}
type WordAnalysis struct {
	Word         string            `json:"word"`
	Morphologies []MorphologyLabel `json:"morphologies"`
}

// Analyze returns known components and their dictionary analyses.
func (c *Checker) Analyze(word string) []WordAnalysis {
	prefix, stem, suffix := SplitWord(word)
	out := []WordAnalysis{}
	for _, part := range []string{prefix, stem, suffix} {
		if part == "" {
			continue
		}
		morphologies := c.Morph(part)
		var labels []MorphologyLabel
		if meaning := AnalyzeWord(part); meaning != "" {
			labels = []MorphologyLabel{{strings.Join(morphologies, " | "), meaning}}
		} else {
			for _, m := range morphologies {
				labels = append(labels, MorphologyLabel{m, ReadableMorphology(m)})
			}
		}
		if len(labels) > 0 {
			out = append(out, WordAnalysis{part, labels})
		}
	}
	return out
}
func (c *Checker) ReadableMorphology(morphology string) string { return ReadableMorphology(morphology) }

type LabelLexicon interface {
	Morph(string) []string
	IsValidToken(string) bool
}
type LabeledPart struct {
	Type       string   `json:"sType"`
	Value      string   `json:"sValue"`
	Morphology []string `json:"lMorph"`
	Labels     []string `json:"aLabels,omitempty"`
}
type TokenLabels struct {
	Morphology []string      `json:"lMorph"`
	Labels     []string      `json:"aLabels"`
	ValidToken *bool         `json:"bValidToken,omitempty"`
	SubTokens  []LabeledPart `json:"lSubTokens,omitempty"`
}

func copiedMorphology(values []string) []string {
	out := make([]string, len(values))
	copy(out, values)
	return out
}

var simpleTokenLabels = map[string]string{
	"SYMBOL": "symbole", "EMOJI": "émoji", "NUM": "nombre", "LINK": "hyperlien", "TAG": "étiquette (hashtag)", "HTML": "balise HTML", "PSEUDOHTML": "balise pseudo-HTML", "HTMLENTITY": "entité caractère XML/HTML", "HOUR": "heure", "WORDORD": "nombre ordinal", "FOLDERUNIX": "dossier UNIX (et dérivés)", "FOLDERWIN": "dossier Windows", "WORD_ACRONYM": "sigle ou acronyme",
}

// LabelToken returns independent descriptive fields. A supplied morphology
// slice, even empty, marks word tokens as already validated, matching Python.
func LabelToken(kind, value string, morphologies []string, checker LabelLexicon) TokenLabels {
	loadLexical()
	out := TokenLabels{}
	if strings.HasPrefix(kind, "WORD") {
		valid := morphologies != nil || checker.IsValidToken(value)
		out.ValidToken = &valid
	}
	if morphologies == nil {
		morphologies = checker.Morph(value)
	}
	out.Morphology = copiedMorphology(morphologies)
	if strings.HasPrefix(kind, "WORD") {
		prefix, stem, suffix := SplitWord(value)
		if stem != value {
			for _, part := range []string{prefix, stem, suffix} {
				out.SubTokens = append(out.SubTokens, LabeledPart{Type: "WORD", Value: part, Morphology: copiedMorphology(checker.Morph(part))})
			}
		}
	}
	switch kind {
	case "PUNC", "SIGN":
		label, ok := lexical.Values[value]
		if !ok {
			label = "signe de ponctuation divers"
		}
		out.Labels = []string{label}
	case "WORD", "WORDELD":
		if len(out.Morphology) > 0 {
			out.Labels = make([]string, 0, len(out.Morphology))
			for _, m := range out.Morphology {
				out.Labels = append(out.Labels, ReadableMorphology(m))
			}
		} else {
			label := "mot inconnu du dictionnaire"
			if strings.Count(value, "-") > 4 {
				label = "élément complexe indéterminé"
			} else if lexicalMatch("_zPartDemForm", value, false) != nil {
				label = "mot avec particule démonstrative"
			} else if lexicalMatch("_zImperatifVerb", value, false) != nil {
				label = "forme verbale impérative"
			} else if lexicalMatch("_zInterroVerb", value, false) != nil {
				label = "forme verbale interrogative"
			}
			out.Labels = []string{label}
		}
		for i := range out.SubTokens {
			part := &out.SubTokens[i]
			if part.Value != "" {
				if label, ok := lexical.Values[part.Value]; ok {
					part.Morphology = []string{""}
					part.Labels = []string{label}
				} else {
					part.Labels = make([]string, 0, len(part.Morphology))
					for _, m := range part.Morphology {
						part.Labels = append(part.Labels, ReadableMorphology(m))
					}
				}
			}
		}
	default:
		label, ok := simpleTokenLabels[kind]
		if !ok {
			label = "token de nature inconnue"
		}
		out.Labels = []string{label}
	}
	return out
}
