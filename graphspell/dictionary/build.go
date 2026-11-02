// The DAWG construction follows Olivier R.'s graphspell/dawg.py (MPL 2.0),
// which credits Steve Hanov's incremental minimal acyclic word-graph algorithm.
package dictionary

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math/bits"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/strtransform"
)

// BuildOptions controls deterministic dictionary generation. Stemming defaults
// to S (suffixes), with A (prefixes and suffixes) and N (literal stems) available.
// TagPattern uses Go's standard regular expression syntax and is optional.
// Date is emitted exactly as supplied through Metadata; no wall clock is read.
type BuildOptions struct {
	Metadata             Metadata
	Stemming, TagPattern string
}

// ReadLexicon reads the upstream three-column UTF-8 format and metadata comments.
// Invalid rows are reported with their line number rather than silently skipped.
func ReadLexicon(r io.Reader) ([]Entry, Metadata, error) {
	entries := []Entry{}
	meta := Metadata{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for line := 1; scanner.Scan(); line++ {
		s := scanner.Text()
		if !utf8.ValidString(s) {
			return nil, meta, fmt.Errorf("lexicon line %d: invalid UTF-8", line)
		}
		s = strings.TrimSpace(s)
		if strings.HasPrefix(s, "##") {
			key, value, ok := strings.Cut(strings.TrimSpace(s[2:]), ":")
			if ok {
				value = strings.TrimSpace(value)
				switch strings.TrimSpace(key) {
				case "LangCode":
					meta.LanguageCode = value
				case "LangName":
					meta.LanguageName = value
				case "DicName":
					meta.Name = value
				case "Description":
					meta.Description = value
				}
			}
			continue
		}
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		columns := strings.Split(s, "\t")
		if len(columns) != 3 {
			return nil, meta, fmt.Errorf("lexicon line %d: expected three tab-separated fields", line)
		}
		entries = append(entries, Entry{columns[0], columns[1], columns[2]})
	}
	if err := scanner.Err(); err != nil {
		return nil, meta, fmt.Errorf("read lexicon: %w", err)
	}
	return entries, meta, nil
}

type graphArc struct {
	value int
	child *graphNode
}
type graphNode struct {
	id, address int
	final       bool
	arcs        []graphArc
}
type uncheckedArc struct {
	parent   *graphNode
	arcIndex int
	child    *graphNode
}

// Build compiles entries to a minimal acyclic graph, retaining upstream value
// assignment, frequency ordering, morphology order and compact JSON layout.
// Duplicate input entries contribute to arc frequencies but appear only once
// in the graph, matching the reference generator.
func Build(entries []Entry, options BuildOptions) (*Dictionary, error) {
	mode := strings.ToUpper(options.Stemming)
	if mode == "" {
		mode = "S"
	}
	if mode != "S" && mode != "A" && mode != "N" {
		return nil, fmt.Errorf("unknown stemming mode %q", mode)
	}
	var filter *regexp.Regexp
	var err error
	if options.TagPattern != "" {
		filter, err = regexp.Compile(options.TagPattern)
		if err != nil {
			return nil, fmt.Errorf("tag filter: %w", err)
		}
	}
	chars := []string{""}
	affixes, tags := []string{}, []string{}
	charIDs, affixIDs, tagIDs := map[string]int{}, map[string]int{}, map[string]int{}
	charCounts, affixCounts, tagCounts := map[int]int{}, map[int]int{}, map[int]int{}
	type encodedEntry struct {
		word       string
		affix, tag int
	}
	encoded := []encodedEntry{}
	unique := map[string]bool{}
	bigrams := map[string]bool{}
	for i, entry := range entries {
		if !utf8.ValidString(entry.Word) || !utf8.ValidString(entry.Stem) || !utf8.ValidString(entry.Tag) {
			return nil, fmt.Errorf("entry %d: invalid UTF-8", i)
		}
		if filter != nil && !filter.MatchString(entry.Tag) {
			continue
		}
		runes := []rune(entry.Word)
		for i, c := range runes {
			s := string(c)
			id, ok := charIDs[s]
			if !ok {
				id = len(chars)
				charIDs[s] = id
				chars = append(chars, s)
			}
			charCounts[id]++
			if i > 0 {
				bigrams[string(runes[i-1:i+1])] = true
			}
		}
		var code string
		switch mode {
		case "S":
			code = strtransform.DefineSuffixCode(entry.Word, entry.Stem)
		case "A":
			code = strtransform.DefineAffixCode(entry.Word, entry.Stem)
		case "N":
			code = entry.Stem
		}
		affix, ok := affixIDs[code]
		if !ok {
			affix = len(affixes)
			affixIDs[code] = affix
			affixes = append(affixes, code)
		}
		affixCounts[affix]++
		tag, ok := tagIDs[entry.Tag]
		if !ok {
			tag = len(tags)
			tagIDs[entry.Tag] = tag
			tags = append(tags, entry.Tag)
		}
		tagCounts[tag]++
		key := entry.Word + "\x00" + strconv.Itoa(affix) + "\x00" + strconv.Itoa(tag)
		if !unique[key] {
			encoded = append(encoded, encodedEntry{entry.Word, affix, tag})
			unique[key] = true
		}
	}
	if len(encoded) == 0 {
		return nil, fmt.Errorf("empty lexicon after filtering")
	}
	nChar, nAff := len(chars), len(affixes)
	values := make([]string, 0, len(chars)+len(affixes)+len(tags))
	values = append(values, chars...)
	values = append(values, affixes...)
	values = append(values, tags...)
	counts := make([]int, len(values))
	for id, n := range charCounts {
		counts[id] = n
	}
	for id, n := range affixCounts {
		counts[id+nChar] = n
	}
	for id, n := range tagCounts {
		counts[id+nChar+nAff] = n
	}
	paths := make([][]int, 0, len(encoded))
	for _, entry := range encoded {
		path := make([]int, 0, utf8.RuneCountInString(entry.word)+2)
		for _, c := range entry.word {
			path = append(path, charIDs[string(c)])
		}
		path = append(path, entry.affix+nChar, entry.tag+nChar+nAff)
		paths = append(paths, path)
	}
	sort.Slice(paths, func(i, j int) bool {
		a, b := paths[i], paths[j]
		for n := 0; n < len(a) && n < len(b); n++ {
			if a[n] != b[n] {
				return a[n] < b[n]
			}
		}
		return len(a) < len(b)
	})
	root := &graphNode{id: 0}
	nextID := 1
	unchecked := []uncheckedArc{}
	registry := map[string]*graphNode{}
	nodes := []*graphNode{}
	keyFor := func(node *graphNode) string {
		key := make([]byte, 1+16*len(node.arcs))
		if node.final {
			key[0] = 1
		}
		for i, a := range node.arcs {
			binary.LittleEndian.PutUint64(key[1+16*i:], uint64(a.value))
			binary.LittleEndian.PutUint64(key[9+16*i:], uint64(a.child.id))
		}
		return string(key)
	}
	minimize := func(downTo int) {
		for i := len(unchecked) - 1; i >= downTo; i-- {
			edge := unchecked[i]
			key := keyFor(edge.child)
			if existing, ok := registry[key]; ok {
				edge.parent.arcs[edge.arcIndex].child = existing
			} else {
				registry[key] = edge.child
				nodes = append(nodes, edge.child)
			}
		}
		unchecked = unchecked[:downTo]
	}
	previous := []int{}
	for _, path := range paths {
		common := 0
		for common < len(path) && common < len(previous) && path[common] == previous[common] {
			common++
		}
		minimize(common)
		node := root
		if len(unchecked) > 0 {
			node = unchecked[len(unchecked)-1].child
		}
		for i := common; i < len(path); i++ {
			child := &graphNode{id: nextID}
			nextID++
			node.arcs = append(node.arcs, graphArc{path[i], child})
			unchecked = append(unchecked, uncheckedArc{node, len(node.arcs) - 1, child})
			if i == len(path)-2 {
				node.final = true
			}
			node = child
		}
		node.final = true
		previous = path
	}
	minimize(0)
	arcs := len(root.arcs)
	for _, n := range nodes {
		arcs += len(n.arcs)
	}
	allNodes := append([]*graphNode{root}, nodes...)
	address := 0
	for _, node := range allNodes {
		sort.SliceStable(node.arcs, func(i, j int) bool { return counts[node.arcs[i].value] > counts[node.arcs[j].value] })
		node.address = address
		length := len(node.arcs)
		if length == 0 {
			length = 1
		}
		address += length * 2
	}
	arcBytes := (bits.Len(uint(len(values)))+2)/8 + 1
	addressBytes := 1
	for uint64(arcBytes+addressBytes)*uint64(arcs) > uint64(1)<<uint(addressBytes*8) {
		addressBytes++
		if addressBytes > 8 {
			return nil, fmt.Errorf("dictionary exceeds address capacity")
		}
	}
	finalMask, lastMask := uint64(1)<<uint(arcBytes*8-1), uint64(1)<<uint(arcBytes*8-2)
	graph := make([]uint64, 0, address)
	for _, node := range allNodes {
		if len(node.arcs) == 0 {
			graph = append(graph, finalMask|lastMask, 0)
			continue
		}
		for i, a := range node.arcs {
			v := uint64(a.value)
			if i == 0 && node.final {
				v |= finalMask
			}
			if i == len(node.arcs)-1 {
				v |= lastMask
			}
			graph = append(graph, v, uint64(a.child.address))
		}
	}
	grams := make([]string, 0, len(bigrams))
	for gram := range bigrams {
		grams = append(grams, gram)
	}
	sort.Strings(grams)
	meta := options.Metadata
	if meta.SourceFile == "" {
		meta.SourceFile = "[None]"
	}
	d := &Dictionary{data: wireData{Header: "/grammalecte-fsa/", LanguageCode: meta.LanguageCode, LanguageName: meta.LanguageName, Name: meta.Name, Description: meta.Description, SourceFile: meta.SourceFile, Date: meta.Date, Entries: len(paths), Characters: nChar - 1, Affixes: nAff, Tags: len(tags) + 1, Stemming: mode, CharacterValues: charIDs, Nodes: len(nodes), Arcs: arcs, ArcValues: len(values), Values: values, ArcBytes: arcBytes, AddressBytes: addressBytes, Graph: graph, Bigrams: grams}}
	if err := d.validate(); err != nil {
		return nil, fmt.Errorf("generated dictionary: %w", err)
	}
	return d, nil
}
