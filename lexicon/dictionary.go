// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// Ported from Grammalecte genfrdic.py by Olivier R.

package lexicon

import (
	"bufio"
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/strtransform"
)

//go:embed data/FRANCAIS.dic.gz data/FRANCAIS_7.aff.gz
var dictionaryFiles embed.FS

// Dictionary contains source entries and ordered affixes, followed by generated inflections.
type Dictionary struct {
	Entries               []*Entry
	Flags                 []*Flag
	Settings              string
	Flexions              []*Flexion
	RecognizedOccurrences int64
	flagsByName           map[string]*Flag
}

// EmbeddedDictionary parses the embedded pinned French source dictionary and affixes.
func EmbeddedDictionary() (*Dictionary, error) {
	entryFile, err := dictionaryFiles.Open("data/FRANCAIS.dic.gz")
	if err != nil {
		return nil, err
	}
	defer entryFile.Close()
	entries, err := gzip.NewReader(entryFile)
	if err != nil {
		return nil, err
	}
	defer entries.Close()
	affixFile, err := dictionaryFiles.Open("data/FRANCAIS_7.aff.gz")
	if err != nil {
		return nil, err
	}
	defer affixFile.Close()
	affixes, err := gzip.NewReader(affixFile)
	if err != nil {
		return nil, err
	}
	defer affixes.Close()
	return ReadDictionary(entries, affixes)
}

// ReadDictionary reads the source dictionary and affix formats from UTF-8 streams.
func ReadDictionary(entries, affixes io.Reader) (*Dictionary, error) {
	d := &Dictionary{flagsByName: map[string]*Flag{}}
	scanner := newScanner(entries)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strtransform.IsDigit(line) && !strings.HasPrefix(line, "#") {
			d.Entries = append(d.Entries, ParseEntry(line))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	reader := bufio.NewReader(newUniversalReader(affixes))
	settings := true
	var flag *Flag
	lineNumber := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}
		if line == "" && err == io.EOF {
			break
		}
		lineNumber++
		switch {
		case strings.HasPrefix(line, "# END"):
			settings = false
		case strings.HasPrefix(line, "#"):
		case strings.HasPrefix(line, "PFX"), strings.HasPrefix(line, "SFX"):
			line = strings.TrimRight(line, " \n")
			if before, _, ok := strings.Cut(line, "#"); ok {
				line = strings.TrimRight(before, " ")
			}
			parts := strings.Fields(line)
			if len(parts) < 4 {
				return nil, fmt.Errorf("affix line %d: expected four or more fields", lineNumber)
			}
			if _, ok := d.flagsByName[parts[1]]; !ok {
				flag = &Flag{Name: parts[1], Suffix: parts[0] == "SFX", Mix: parts[2] == "Y"}
				d.flagsByName[flag.Name] = flag
				d.Flags = append(d.Flags, flag)
			} else {
				// The source appends to the most recently introduced flag.
				rule, parseErr := ParseAffixRule(line)
				if parseErr != nil {
					return nil, fmt.Errorf("affix line %d: %w", lineNumber, parseErr)
				}
				flag.Rules = append(flag.Rules, rule)
			}
		default:
			if settings {
				d.Settings += line
			}
		}
		if err == io.EOF {
			break
		}
	}
	return d, nil
}

func newScanner(r io.Reader) *bufio.Scanner {
	s := bufio.NewScanner(newUniversalReader(r))
	s.Buffer(make([]byte, 64*1024), 16*1024*1024)
	return s
}

// SortNatural stably sorts entries by the source's three-part natural key.
func (d *Dictionary) SortNatural() {
	slices.SortStableFunc(d.Entries, func(a, b *Entry) int {
		ka, kb := a.NaturalKey(), b.NaturalKey()
		for i := range ka {
			if c := strings.Compare(ka[i], kb[i]); c != 0 {
				return c
			}
		}
		return 0
	})
}

// Generate replaces the inflections using the exact source order and duplicate counts.
func (d *Dictionary) Generate() error {
	d.Flexions = nil
	for _, entry := range d.Entries {
		forms, err := d.inflect(entry, "", 0)
		if err != nil {
			return err
		}
		entry.Flexions = nil
		for _, form := range forms {
			if strings.Contains(form.morph, "+") {
				continue
			}
			morph := strings.Trim(strings.ReplaceAll(form.morph, "  ", " "), " ")
			if strings.HasSuffix(morph, " mas") || strings.HasSuffix(morph, " fem") || strings.HasSuffix(morph, " epi") {
				continue
			}
			entry.Flexions = append(entry.Flexions, &Flexion{Entry: entry, Text: form.text, Morphology: morph, Dictionary: form.dictionary, Index: len(entry.Flexions) + 1})
		}
		entry.Stem = entry.Fields["st"]
		if entry.Stem == "" {
			entry.Stem = entry.Lemma
		}
		duplicates := map[string]int{}
		for _, f := range entry.Flexions {
			duplicates[f.Text]++
		}
		for _, f := range entry.Flexions {
			f.Duplicates = duplicates[f.Text]
		}
		d.Flexions = append(d.Flexions, entry.Flexions...)
	}
	owners := map[string][]*Entry{}
	for _, f := range d.Flexions {
		if !slices.Contains(owners[f.Text], f.Entry) {
			owners[f.Text] = append(owners[f.Text], f.Entry)
		}
	}
	for _, f := range d.Flexions {
		for _, entry := range owners[f.Text] {
			if entry != f.Entry {
				f.OtherEntries = append(f.OtherEntries, entry)
			}
		}
	}
	return nil
}

type inflection struct{ text, morph, dictionary string }

func (d *Dictionary) inflect(entry *Entry, morph string, depth int) ([]inflection, error) {
	if depth == 2 {
		return nil, nil
	}
	if depth == 0 {
		morph = entry.LexicalMorphology()
	}
	var forms, prefixes, suffixes []inflection
	if depth == 0 && !strings.HasSuffix(entry.Flags, "()") {
		forms = append(forms, inflection{entry.Lemma, morph, entry.Fields["di"]})
	}
	for _, name := range LongFlags(entry.Flags) {
		flag, ok := d.flagsByName[name]
		if !ok {
			if name != "**" && name != "()" && name != "||" && name != "--" {
				forms = append(forms, inflection{entry.Lemma, "[unknown flag: " + name + "]", entry.Fields["di"]})
			}
			continue
		}
		for _, rule := range flag.Rules {
			if !rule.Matches(entry.Lemma) {
				continue
			}
			ruleMorph := rule.LexicalMorphology()
			form := inflection{dictionary: rule.Fields["di"]}
			if !flag.Suffix {
				form.morph = ruleMorph + morph
				if rule.Cut == "0" {
					form.text = rule.Add + entry.Lemma
				} else {
					form.text = strings.Replace(entry.Lemma, rule.Cut, rule.Add, 1)
				}
			} else {
				form.morph = morph + ruleMorph
				if rule.Cut == "0" {
					form.text = entry.Lemma + rule.Add
				} else {
					runes := []rune(entry.Lemma)
					keep := len(runes) - len([]rune(rule.Cut))
					if keep < 0 {
						keep = 0
					}
					form.text = string(runes[:keep]) + rule.Add
				}
			}
			circumfix := flag.Suffix && strings.HasSuffix(rule.Flags, "**") && rule.Flags != "**"
			if !circumfix {
				if flag.Mix {
					if (!flag.Suffix && len(suffixes) > 0) || (flag.Suffix && len(prefixes) > 0) {
						return nil, fmt.Errorf("lemma %q: source cross-product rule produces an invalid two-field inflection", entry.Lemma)
					}
					if flag.Suffix {
						suffixes = append(suffixes, form)
					} else {
						prefixes = append(prefixes, form)
					}
				} else {
					forms = append(forms, form)
				}
			}
			if circumfix || (rule.Flags != "" && rule.Flags != "**") {
				more, err := d.inflect(ParseEntry(form.text+"/"+rule.Flags), form.morph, depth+1)
				if err != nil {
					return nil, err
				}
				forms = append(forms, more...)
			}
		}
	}
	forms = append(forms, prefixes...)
	forms = append(forms, suffixes...)
	return forms, nil
}
