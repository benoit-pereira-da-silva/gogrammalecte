package french

// Native resource generation follows gc_lang/fr/build_data.py by Olivier R.
// The upstream builder is distributed under MPL 2.0. Linguistic inputs retain
// their upstream content; see SOURCES.md for provenance and scope.

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/strtransform"
)

//go:embed sources/*.gz
var sourceResources embed.FS

// MorphologyLexicon supplies spelling and morphology for linguistic generators.
// Its results must use the pinned dictionary's morphology format and ordering.
type MorphologyLexicon interface {
	Lookup(string) bool
	Morph(string) []string
}

// Source opens an embedded original linguistic input by its base filename,
// such as "thes_fr.dat" or "synsets_fr.dat". The caller must close the reader.
func Source(name string) (io.ReadCloser, error) { return sourceReader(name) }

func sourceReader(name string) (io.ReadCloser, error) {
	b, e := sourceResources.ReadFile("sources/" + name + ".gz")
	if e != nil {
		return nil, e
	}
	return gzip.NewReader(bytes.NewReader(b))
}
func readSource(r io.Reader, consume func(int, string) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		s := strings.TrimSpace(scanner.Text())
		if s == "__END__" {
			break
		}
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		if e := consume(line, s); e != nil {
			return fmt.Errorf("line %d: %w", line, e)
		}
	}
	return scanner.Err()
}

// GenerateInflectionData rebuilds the embedded declension tables using only Go.
func GenerateInflectionData() ([]byte, error) {
	r, e := sourceReader("dictDecl.txt")
	if e != nil {
		return nil, e
	}
	defer r.Close()
	return CompileInflection(r)
}

// CompileInflection compiles a dictDecl-format stream into the compatible JSON table.
func CompileInflection(r io.Reader) ([]byte, error) {
	pluralTags, feminineTags := []string{}, []string{}
	plurals, masculine := map[string]int{}, map[string]int{}
	tagIndex := map[string]int{}
	lemma, kind := "", ""
	tags, femPlural := []string{}, []string{}
	active := false
	err := readSource(r, func(_ int, line string) error {
		fields := strings.Split(line, "\t")
		switch len(fields) {
		case 2:
			if active {
				return fmt.Errorf("entry %q is missing its terminator", lemma)
			}
			lemma = fields[0]
			if fields[1] == "" {
				return fmt.Errorf("empty inflection flags")
			}
			active = true
			tags = []string{}
			femPlural = []string{}
			switch fields[1][0] {
			case 'S':
				kind = "singular"
			case 'X', 'A', 'I':
				kind = "plural"
			case 'F', 'W':
				kind = "gender"
			default:
				return fmt.Errorf("unknown inflection flags: %s", fields[1])
			}
		case 3:
			if !active {
				return fmt.Errorf("form outside an entry")
			}
			if kind == "plural" && strings.HasSuffix(fields[1], "pl") {
				tags = append(tags, strtransform.DefineSuffixCode(lemma, fields[2]))
			}
			if kind == "gender" {
				if strings.HasSuffix(fields[1], "fem sg") {
					tags = append(tags, strtransform.DefineSuffixCode(lemma, fields[2]))
				}
				if strings.HasSuffix(fields[1], "fem pl") {
					femPlural = append(femPlural, strtransform.DefineSuffixCode(lemma, fields[2]))
				}
			}
		case 1:
			if line != "$" || !active {
				return fmt.Errorf("unexpected source record: %s", line)
			}
			active = false
			if kind == "plural" {
				tag := strings.Join(tags, "|")
				index, ok := tagIndex[tag]
				if !ok {
					index = len(pluralTags)
					tagIndex[tag] = index
					pluralTags = append(pluralTags, tag)
				}
				plurals[lemma] = index
			}
			if kind == "gender" {
				tag := strings.Join(tags, "|") + "/" + strings.Join(femPlural, "|")
				index, ok := tagIndex[tag]
				if !ok {
					index = len(feminineTags)
					tagIndex[tag] = index
					feminineTags = append(feminineTags, tag)
				}
				masculine[lemma] = index
			}
		default:
			return fmt.Errorf("invalid field count")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if active {
		return nil, fmt.Errorf("entry %q is missing its terminator", lemma)
	}
	return json.Marshal(map[string]any{"lTagMiscPlur": pluralTags, "lTagFemForm": feminineTags, "dMiscPlur": plurals, "dMasForm": masculine})
}

var generatorTenses = []string{":P", ":Q", ":Ip", ":Iq", ":Is", ":If", ":K", ":Sp", ":Sq", ":E"}
var generatorPeople = map[string][]string{":P": {":P"}, ":Q": {":m:s", ":f:s", ":m:p", ":f:p"}, ":Ip": {":1s", ":2s", ":3s", ":1p", ":2p", ":3p", ":1ś"}, ":Iq": {":1s", ":2s", ":3s", ":1p", ":2p", ":3p"}, ":Is": {":1s", ":2s", ":3s", ":1p", ":2p", ":3p"}, ":If": {":1s", ":2s", ":3s", ":1p", ":2p", ":3p"}, ":K": {":1s", ":2s", ":3s", ":1p", ":2p", ":3p"}, ":Sp": {":1s", ":2s", ":3s", ":1p", ":2p", ":3p", ":1ś"}, ":Sq": {":1s", ":2s", ":3s", ":1p", ":2p", ":3p", ":1ś"}, ":E": {":2s", ":1p", ":2p"}}
var sourceTag = map[string]string{"infi": ":Y", "ppre": ":P", "ppas": ":Q", "ipre": ":Ip", "iimp": ":Iq", "ipsi": ":Is", "ifut": ":If", "spre": ":Sp", "simp": ":Sq", "cond": ":K", "impe": ":E", "1sg": ":1s", "2sg": ":2s", "3sg": ":3s", "1pl": ":1p", "2pl": ":2p", "3pl": ":3p", "1isg": ":1ś", "mas sg": ":m:s", "mas pl": ":m:p", "mas inv": ":m:s", "fem sg": ":f:s", "fem pl": ":f:p", "epi inv": ":m:s"}

// GenerateConjugationData rebuilds patterns from embedded conjugation sources.
// Derived nouns require a lexicon. Their order is sorted for reproducibility;
// the upstream generator used unordered Python sets for that field.
func GenerateConjugationData(lexicon MorphologyLexicon) ([]byte, error) {
	r, e := sourceReader("dictConj.txt")
	if e != nil {
		return nil, e
	}
	defer r.Close()
	return CompileConjugation(r, lexicon)
}

// CompileConjugation compiles a dictConj stream into compatible table JSON.
func CompileConjugation(r io.Reader, lexicon MorphologyLexicon) ([]byte, error) {
	if lexicon == nil {
		return nil, fmt.Errorf("conjugation generation requires a lexicon")
	}
	types := []string{}
	typeIndex := map[string]int{}
	tags := [][]int{}
	tagIndex := map[string]int{}
	verbs := map[string][2]int{}
	names := map[string]map[string]bool{}
	patterns := map[string][]map[string]string{}
	patternIndex := map[string]map[string]int{}
	for _, t := range generatorTenses {
		patterns[t] = []map[string]string{}
		patternIndex[t] = map[string]int{}
	}
	lemma, info := "", ""
	forms := map[string]map[string]string{}
	active := false
	addNames := func(word string, initial bool) {
		for _, m := range lexicon.Morph(word) {
			if !strings.Contains(m, ":N") {
				continue
			}
			if names[lemma] == nil {
				names[lemma] = map[string]bool{}
			}
			names[lemma][word] = true
			if initial {
				break
			}
			if end := strings.IndexByte(m, '/'); end > 0 {
				names[lemma][m[1:end]] = true
			}
		}
	}
	err := readSource(r, func(_ int, line string) error {
		fields := strings.Split(line, "\t")
		switch len(fields) {
		case 2:
			if active {
				return fmt.Errorf("entry %q is missing its terminator", lemma)
			}
			lemma, info = fields[0], fields[1]
			active = true
			forms = map[string]map[string]string{}
			for _, t := range generatorTenses {
				forms[t] = map[string]string{}
				for _, p := range generatorPeople[t] {
					forms[t][p] = ""
				}
			}
			if _, ok := typeIndex[info]; !ok {
				typeIndex[info] = len(types)
				types = append(types, info)
			}
			addNames(lemma, true)
		case 3:
			if !active {
				return fmt.Errorf("form outside an entry")
			}
			tag, word := fields[1], fields[2]
			if tag == "ppre" {
				forms[":P"][":P"] = strtransform.DefineSuffixCode(lemma, word)
			} else if space := strings.IndexByte(tag, ' '); space >= 0 {
				tense, person := sourceTag[tag[:space]], sourceTag[tag[space+1:]]
				if tense == "" || person == "" {
					return fmt.Errorf("unknown conjugation tag: %s", tag)
				}
				if _, ok := forms[tense][person]; !ok {
					return fmt.Errorf("invalid conjugation tag: %s", tag)
				}
				if forms[tense][person] == "" {
					forms[tense][person] = strtransform.DefineSuffixCode(lemma, word)
				}
			} else if tag != "infi" {
				return fmt.Errorf("unknown conjugation tag: %s", tag)
			}
			addNames(word, false)
		case 1:
			if line != "$" || !active {
				return fmt.Errorf("unexpected source record: %s", line)
			}
			active = false
			if forms[":Ip"][":1ś"] == "2è" {
				forms[":Ip"][":1ś"] = "2é"
			} else if lemma == "pouvoir" {
				forms[":Ip"][":1ś"] = "6uis"
			}
			indexes := make([]int, 0, 10)
			for _, t := range generatorTenses {
				encoded, _ := json.Marshal(forms[t])
				key := string(encoded)
				i, ok := patternIndex[t][key]
				if !ok {
					i = len(patterns[t])
					patternIndex[t][key] = i
					patterns[t] = append(patterns[t], forms[t])
				}
				indexes = append(indexes, i)
			}
			encoded, _ := json.Marshal(indexes)
			key := string(encoded)
			i, ok := tagIndex[key]
			if !ok {
				i = len(tags)
				tagIndex[key] = i
				tags = append(tags, indexes)
			}
			verbs[lemma] = [2]int{typeIndex[info], i}
		default:
			return fmt.Errorf("invalid field count")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if active {
		return nil, fmt.Errorf("entry %q is missing its terminator", lemma)
	}
	nameLists := map[string][]string{}
	for word, set := range names {
		list := make([]string, 0, len(set))
		for name := range set {
			list = append(list, name)
		}
		sort.Strings(list)
		nameLists[word] = list
	}
	return json.Marshal(map[string]any{"lVtyp": types, "lTags": tags, "dPatternConj": patterns, "dVerb": verbs, "dVerbNames": nameLists})
}
