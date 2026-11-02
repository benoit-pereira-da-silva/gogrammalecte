// Package french provides French inflection, conjugation, phonetic and
// morphological helpers ported from Grammalecte by Olivier R. (GPLv3).
// Linguistic tables are embedded; no resource files are needed at runtime.
package french

import (
	"embed"
	"encoding/json"
	"sync"
)

//go:embed data/*.json
var resources embed.FS

var conjugation struct {
	Types    []string                       `json:"lVtyp"`
	Tags     [][]int                        `json:"lTags"`
	Patterns map[string][]map[string]string `json:"dPatternConj"`
	Verbs    map[string][2]int              `json:"dVerb"`
	Names    map[string][]string            `json:"dVerbNames"`
}
var inflection struct {
	PluralTags   []string       `json:"lTagMiscPlur"`
	FeminineTags []string       `json:"lTagFemForm"`
	Plurals      map[string]int `json:"dMiscPlur"`
	Masculine    map[string]int `json:"dMasForm"`
}
var phonetic struct {
	Words        map[string]int      `json:"dWord"`
	Sets         [][]string          `json:"lSet"`
	Morphologies map[string][]string `json:"dMorph"`
}
var thesaurus map[string][]SynonymSet
var loadConjugation = sync.OnceFunc(func() { loadResource("conj", &conjugation) })
var loadInflection = sync.OnceFunc(func() { loadResource("mfsp", &inflection) })
var loadPhonetic = sync.OnceFunc(func() { loadResource("phonet", &phonetic) })
var loadThesaurus = sync.OnceFunc(func() { loadResource("thesaurus", &thesaurus) })

func loadResource(name string, dest any) {
	b, e := resources.ReadFile("data/" + name + ".json")
	if e != nil {
		panic(e)
	}
	if e = json.Unmarshal(b, dest); e != nil {
		panic(e)
	}
}
func copyStrings(values []string) []string {
	out := make([]string, len(values))
	copy(out, values)
	return out
}
