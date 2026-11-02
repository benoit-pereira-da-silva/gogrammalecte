// Package dictionary reads and builds Grammalecte's compact word graphs.
// Algorithms and format are based on graphspell/dawg.py (Olivier R., MPL 2.0)
// and graphspell/ibdawg.py from the pinned Grammalecte reference.
package dictionary

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/strtransform"
)

// Entry is one inflected word, lemma and morphological tag sequence.
type Entry struct {
	Word string `json:"word"`
	Stem string `json:"stem"`
	Tag  string `json:"tag"`
}

// Metadata describes a dictionary without exposing mutable graph storage.
type Metadata struct {
	LanguageCode string `json:"language_code"`
	LanguageName string `json:"language_name"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	SourceFile   string `json:"source_file"`
	Date         string `json:"date"`
}

// Stats describes the encoded graph. Nodes excludes the root. Tags includes
// the reserved value slot, preserving the historical upstream nTag field.
type Stats struct{ Entries, Characters, Affixes, Tags, Nodes, Arcs int }
type wireData struct {
	Header          string         `json:"sHeader"`
	LanguageCode    string         `json:"sLangCode"`
	LanguageName    string         `json:"sLangName"`
	Name            string         `json:"sDicName"`
	Description     string         `json:"sDescription"`
	SourceFile      string         `json:"sFileName"`
	Date            string         `json:"sDate"`
	Entries         int            `json:"nEntry"`
	Characters      int            `json:"nChar"`
	Affixes         int            `json:"nAff"`
	Tags            int            `json:"nTag"`
	Stemming        string         `json:"cStemming"`
	CharacterValues map[string]int `json:"dChar"`
	Nodes           int            `json:"nNode"`
	Arcs            int            `json:"nArc"`
	ArcValues       int            `json:"nArcVal"`
	Values          []string       `json:"lArcVal"`
	ArcBytes        int            `json:"nBytesArc"`
	AddressBytes    int            `json:"nBytesNodeAddress"`
	OffsetBytes     int            `json:"nBytesOffset"`
	Graph           []uint64       `json:"lByDic"`
	LegacyGraph     string         `json:"sByDic,omitempty"`
	Bigrams         []string       `json:"l2grams"`
}

// Dictionary is an immutable validated graph and can be shared by goroutines.
// Construct it with Read or Build. Text inputs must be valid UTF-8.
type Dictionary struct {
	ngrams                       map[string]bool
	data                         wireData
	arcMask, finalMask, lastMask uint64
	allowAcronyms                bool
}

// Read accepts the current JSON integer graph and legacy hexadecimal graph.
// Malformed data is rejected before it can be queried.
func Read(r io.Reader) (*Dictionary, error) {
	var data wireData
	dec := json.NewDecoder(r)
	if err := dec.Decode(&data); err != nil {
		return nil, fmt.Errorf("read dictionary: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("unexpected trailing JSON value")
		}
		return nil, fmt.Errorf("read dictionary trailer: %w", err)
	}
	if data.Graph == nil && data.LegacyGraph != "" {
		if data.ArcBytes < 1 || data.ArcBytes > 8 || data.AddressBytes < 1 || data.AddressBytes > 8 {
			return nil, fmt.Errorf("invalid legacy record widths")
		}
		binary, err := hex.DecodeString(data.LegacyGraph)
		if err != nil {
			return nil, fmt.Errorf("decode legacy graph: %w", err)
		}
		width := data.ArcBytes + data.AddressBytes
		if len(binary)%width != 0 {
			return nil, fmt.Errorf("truncated legacy graph")
		}
		data.Graph = make([]uint64, 0, 2*len(binary)/width)
		for i := 0; i < len(binary); i += width {
			arc, address := uint64(0), uint64(0)
			for _, b := range binary[i : i+data.ArcBytes] {
				arc = arc<<8 | uint64(b)
			}
			for _, b := range binary[i+data.ArcBytes : i+width] {
				address = address<<8 | uint64(b)
			}
			if address%uint64(width) != 0 {
				return nil, fmt.Errorf("unaligned legacy node address")
			}
			data.Graph = append(data.Graph, arc, address/uint64(width)*2)
		}
		data.LegacyGraph = ""
	}
	d := &Dictionary{data: data}
	if err := d.validate(); err != nil {
		return nil, fmt.Errorf("invalid dictionary: %w", err)
	}
	return d, nil
}

// Load reads a dictionary JSON file.
func Load(path string) (*Dictionary, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Read(f)
}

// Write emits the portable JSON format with deterministic map ordering.
func (d *Dictionary) Write(w io.Writer) error { return json.NewEncoder(w).Encode(d.data) }

// Metadata returns the dictionary's descriptive fields.
func (d *Dictionary) Metadata() Metadata {
	return Metadata{d.data.LanguageCode, d.data.LanguageName, d.data.Name, d.data.Description, d.data.SourceFile, d.data.Date}
}

// Stats returns the stored graph counts.
func (d *Dictionary) Stats() Stats {
	return Stats{d.data.Entries, d.data.Characters, d.data.Affixes, d.data.Tags, d.data.Nodes, d.data.Arcs}
}

// WithAcronyms returns an independent view with uppercase acronyms accepted.
func (d *Dictionary) WithAcronyms(accept bool) *Dictionary {
	copy := *d
	copy.allowAcronyms = accept
	return &copy
}

func (d *Dictionary) validate() error {
	w := &d.data
	if w.Header != "/grammalecte-fsa/" {
		return fmt.Errorf("unsupported header %q", w.Header)
	}
	if w.Stemming != "S" && w.Stemming != "A" && w.Stemming != "N" {
		return fmt.Errorf("unknown stemming mode %q", w.Stemming)
	}
	if w.ArcBytes < 1 || w.ArcBytes > 8 || w.AddressBytes < 1 || w.AddressBytes > 8 {
		return fmt.Errorf("invalid record widths")
	}
	if w.Characters < 0 || w.Affixes < 1 || w.Tags < 1 || w.ArcValues != len(w.Values) || w.Characters+w.Affixes+w.Tags != len(w.Values) {
		return fmt.Errorf("invalid value counts")
	}
	if len(w.Values) == 0 || w.Values[0] != "" || len(w.CharacterValues) != w.Characters {
		return fmt.Errorf("invalid character table")
	}
	d.arcMask = (uint64(1) << uint(w.ArcBytes*8-3)) - 1
	d.finalMask = uint64(1) << uint(w.ArcBytes*8-1)
	d.lastMask = uint64(1) << uint(w.ArcBytes*8-2)
	if uint64(len(w.Values)-1) > d.arcMask {
		return fmt.Errorf("arc width too small")
	}
	seenChars := make(map[int]bool, len(w.CharacterValues))
	for c, v := range w.CharacterValues {
		if !utf8.ValidString(c) || utf8.RuneCountInString(c) != 1 || v < 1 || v > w.Characters || w.Values[v] != c || seenChars[v] {
			return fmt.Errorf("invalid character mapping %q", c)
		}
		seenChars[v] = true
	}
	for i, v := range w.Values {
		if !utf8.ValidString(v) {
			return fmt.Errorf("invalid UTF-8 at value %d", i)
		}
		if i > w.Characters && i <= w.Characters+w.Affixes {
			if _, err := d.decodeStem("", v); err != nil {
				return fmt.Errorf("invalid affix at value %d: %w", i, err)
			}
		}
	}
	if len(w.Graph) < 2 || len(w.Graph)%2 != 0 {
		return fmt.Errorf("invalid graph length")
	}
	starts := map[int]bool{0: true}
	nodeStart := 0
	last := false
	for i := 0; i < len(w.Graph); i += 2 {
		raw := w.Graph[i]
		value := raw & d.arcMask
		if raw&^(d.arcMask|d.finalMask|d.lastMask) != 0 || value >= uint64(len(w.Values)) {
			return fmt.Errorf("invalid arc at %d", i)
		}
		if i != nodeStart && raw&d.finalMask != 0 {
			return fmt.Errorf("final flag outside node start at %d", i)
		}
		last = raw&d.lastMask != 0
		if value == 0 && (!last || i != nodeStart || raw&d.finalMask == 0 || w.Graph[i+1] != 0) {
			return fmt.Errorf("invalid terminal node at %d", i)
		}
		if last && i+2 < len(w.Graph) {
			nodeStart = i + 2
			starts[nodeStart] = true
		}
	}
	if !last {
		return fmt.Errorf("unterminated node")
	}
	for i := 0; i < len(w.Graph); i += 2 {
		target := w.Graph[i+1]
		if target >= uint64(len(w.Graph)) || !starts[int(target)] {
			return fmt.Errorf("invalid node address at %d", i)
		}
	}
	state := make(map[int]uint8, len(starts))
	var visit func(int) error
	visit = func(addr int) error {
		if state[addr] == 1 {
			return fmt.Errorf("cyclic graph at node %d", addr)
		}
		if state[addr] == 2 {
			return nil
		}
		state[addr] = 1
		for i := addr; ; i += 2 {
			if w.Graph[i]&d.arcMask != 0 {
				if err := visit(int(w.Graph[i+1])); err != nil {
					return err
				}
			}
			if w.Graph[i]&d.lastMask != 0 {
				break
			}
		}
		state[addr] = 2
		return nil
	}
	if err := visit(0); err != nil {
		return err
	}
	if len(state) != len(starts) {
		return fmt.Errorf("unreachable graph nodes")
	}
	// Validate paths by layer. A word's outgoing affix must lead to tags, whose
	// destinations are terminal nodes. This makes all query decodes total.
	visited := make(map[int]bool)
	var checkWords func(int) error
	checkWords = func(addr int) error {
		if visited[addr] {
			return nil
		}
		visited[addr] = true
		for i := addr; ; i += 2 {
			value := int(w.Graph[i] & d.arcMask)
			next := int(w.Graph[i+1])
			if value >= 1 && value <= w.Characters {
				if err := checkWords(next); err != nil {
					return err
				}
			} else if value > w.Characters && value <= w.Characters+w.Affixes {
				if w.Graph[addr]&d.finalMask == 0 {
					return fmt.Errorf("affix on nonfinal word node")
				}
				for j := next; ; j += 2 {
					tag := int(w.Graph[j] & d.arcMask)
					leaf := int(w.Graph[j+1])
					if tag <= w.Characters+w.Affixes || tag >= len(w.Values) || w.Graph[leaf] != (d.finalMask|d.lastMask) {
						return fmt.Errorf("invalid morphology path")
					}
					if w.Graph[j]&d.lastMask != 0 {
						break
					}
				}
			} else {
				return fmt.Errorf("unexpected arc value %d in word graph", value)
			}
			if w.Graph[i]&d.lastMask != 0 {
				break
			}
		}
		return nil
	}
	if err := checkWords(0); err != nil {
		return err
	}
	d.ngrams = make(map[string]bool, len(w.Bigrams))
	for _, gram := range w.Bigrams {
		d.ngrams[gram] = true
	}
	return nil
}
func (d *Dictionary) next(addr, value int) (int, bool) {
	for i := addr; ; i += 2 {
		raw := d.data.Graph[i]
		if int(raw&d.arcMask) == value {
			return int(d.data.Graph[i+1]), true
		}
		if raw&d.lastMask != 0 {
			return 0, false
		}
	}
}
func (d *Dictionary) wordNode(word string) (int, bool) {
	if !utf8.ValidString(word) {
		return 0, false
	}
	addr := 0
	for _, c := range word {
		v, ok := d.data.CharacterValues[string(c)]
		if !ok {
			return 0, false
		}
		addr, ok = d.next(addr, v)
		if !ok {
			return 0, false
		}
	}
	return addr, d.data.Graph[addr]&d.finalMask != 0
}

// Lookup tests exact spelling and case without normalization.
func (d *Dictionary) Lookup(word string) bool { _, ok := d.wordNode(word); return ok }
func (d *Dictionary) decodeStem(word, code string) (string, error) {
	switch d.data.Stemming {
	case "S":
		return strtransform.ChangeWordWithSuffixCode(word, code)
	case "A":
		return strtransform.ChangeWordWithAffixCode(word, code)
	default:
		return code, nil
	}
}
func (d *Dictionary) rawMorph(word string, tags bool) []string {
	out := []string{}
	addr, ok := d.wordNode(word)
	if !ok {
		return out
	}
	for i := addr; ; i += 2 {
		raw := d.data.Graph[i]
		v := int(raw & d.arcMask)
		if v > d.data.Characters {
			stem, _ := d.decodeStem(word, d.data.Values[v])
			if !tags {
				out = append(out, stem)
			} else {
				for j := int(d.data.Graph[i+1]); ; j += 2 {
					tag := d.data.Graph[j]
					out = append(out, ">"+stem+"/"+d.data.Values[tag&d.arcMask])
					if tag&d.lastMask != 0 {
						break
					}
				}
			}
		}
		if raw&d.lastMask != 0 {
			break
		}
	}
	return out
}

// MorphExact returns morphology strings for the exact input spelling and case,
// without spelling normalization or additional case variants.
func (d *Dictionary) MorphExact(word string) []string { return d.rawMorph(word, true) }

// Morph returns Python-compatible morphology strings, including its casing
// variants and ordering. The result may contain duplicates present upstream.
func (d *Dictionary) Morph(word string) []string {
	if word == "" || !utf8.ValidString(word) {
		return []string{}
	}
	word = strtransform.SpellingNormalization(word)
	out := d.rawMorph(word, true)
	chars := []rune(word)
	if strtransform.IsUpper(string(chars[0])) {
		out = append(out, d.rawMorph(strtransform.Lower(word), true)...)
		if len(chars) > 1 && strtransform.IsUpper(word) {
			out = append(out, d.rawMorph(strtransform.Capitalize(word), true)...)
		}
	}
	return out
}

// Stem returns the exact-case stems from the word's outgoing affix arcs.
func (d *Dictionary) Stem(word string) []string { return d.rawMorph(word, false) }

// IsValid follows Python's casing and leading-digit acceptance rules.
func (d *Dictionary) IsValid(word string) bool {
	if word == "" {
		return true
	}
	if !utf8.ValidString(word) {
		return false
	}
	if d.Lookup(word) {
		return true
	}
	chars := []rune(word)
	first := string(chars[0])
	if strtransform.IsUpper(first) {
		if len(chars) > 1 {
			if strtransform.IsTitle(word) {
				return d.Lookup(strtransform.Lower(word))
			}
			if strtransform.IsUpper(word) {
				return d.allowAcronyms || d.Lookup(strtransform.Lower(word)) || d.Lookup(strtransform.Capitalize(word))
			}
			return d.Lookup(strtransform.Lower(first) + string(chars[1:]))
		}
		return d.Lookup(strtransform.Lower(word))
	}
	return strtransform.IsDigit(first)
}

// IsValidToken normalizes spelling, then applies upstream compound-token rules.
func (d *Dictionary) IsValidToken(token string) bool {
	if !utf8.ValidString(token) {
		return false
	}
	token = strtransform.SpellingNormalization(token)
	if d.IsValid(token) {
		return true
	}
	if strings.Contains(token, "-") {
		if strings.Count(token, "-") > 4 {
			return true
		}
		for _, word := range strings.Split(token, "-") {
			if !d.IsValid(word) {
				return false
			}
		}
		return true
	}
	return strings.ContainsAny(token, ".·")
}

// Entries returns entries in stored arc order, preserving upstream enumeration.
func (d *Dictionary) Entries() []Entry { entries, _ := d.Select("", ""); return entries }

// Select enumerates entries matching Go-standard word and tag regexes. Invalid
// or unsupported regex syntax returns an error instead of disabling a filter.
func (d *Dictionary) Select(wordPattern, tagPattern string) ([]Entry, error) {
	var wordFilter, tagFilter *regexp.Regexp
	var err error
	if wordPattern != "" {
		wordFilter, err = regexp.Compile(wordPattern)
		if err != nil {
			return nil, fmt.Errorf("word pattern: %w", err)
		}
	}
	if tagPattern != "" {
		tagFilter, err = regexp.Compile(tagPattern)
		if err != nil {
			return nil, fmt.Errorf("tag pattern: %w", err)
		}
	}
	entries := []Entry{}
	var walk func(int, string)
	walk = func(addr int, word string) {
		for i := addr; ; i += 2 {
			raw := d.data.Graph[i]
			value := int(raw & d.arcMask)
			next := int(d.data.Graph[i+1])
			if value <= d.data.Characters {
				walk(next, word+d.data.Values[value])
			} else if wordFilter == nil || wordFilter.MatchString(word) {
				stem, _ := d.decodeStem(word, d.data.Values[value])
				for j := next; ; j += 2 {
					tagRaw := d.data.Graph[j]
					tag := d.data.Values[tagRaw&d.arcMask]
					if tagFilter == nil || tagFilter.MatchString(tag) {
						entries = append(entries, Entry{word, stem, tag})
					}
					if tagRaw&d.lastMask != 0 {
						break
					}
				}
			}
			if raw&d.lastMask != 0 {
				break
			}
		}
	}
	walk(0, "")
	return entries, nil
}
