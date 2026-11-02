// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// Ported from Grammalecte genfrdic.py by Olivier R.

package lexicon

import (
	"strings"
	"unicode/utf8"
)

var morphologyFields = []string{"po", "is", "ds", "ts", "ip", "dp", "tp", "sp", "pa", "al", "st", "ph"}
var lexicalFields = []string{"po", "is", "ds", "ts", "ip", "dp", "tp", "sp"}
var entryFields = map[string]bool{"po": true, "is": true, "ds": true, "ts": true, "ip": true, "dp": true, "tp": true, "sp": true, "pa": true, "st": true, "al": true, "ph": true, "lx": true, "se": true, "et": true, "di": true, "fq": true, "id": true}

// Entry is a source dictionary lemma, its continuation flags and original fields.
type Entry struct {
	Lemma                   string
	Flags                   string
	Fields                  map[string]string
	Comment                 string
	InvalidFields           string
	Flexions                []*Flexion
	Stem                    string
	Occurrences             int64
	AverageKnownOccurrences int64
	Frequency               float64
	OldFrequencyIndex       string
}

// ParseEntry preserves the source format's repeated fields and malformed-field text.
func ParseEntry(line string) *Entry {
	e := &Entry{Fields: map[string]string{"di": "*", "id": "0"}, AverageKnownOccurrences: -1}
	line = strings.TrimRight(line, " \n")
	if before, comment, found := strings.Cut(line, "#"); found {
		line = before
		e.Comment = strings.TrimSpace(comment)
	}
	parts := strings.Split(line, "\t")
	if parts[0] == "/" {
		e.Lemma = "/"
	} else {
		first := strings.Split(parts[0], "/")
		e.Lemma = first[0]
		if len(first) > 1 {
			e.Flags = first[1]
		}
	}
	for _, field := range parts[1:] {
		if utf8.RuneCountInString(field) > 3 && len(field) > 2 && field[2] == ':' {
			name, value, _ := strings.Cut(field, ":")
			if !entryFields[name] {
				continue
			}
			if name != "di" && name != "fq" && name != "id" {
				value = e.Fields[name] + " " + value
			}
			e.Fields[name] = strings.TrimSpace(value)
		} else {
			e.InvalidFields += field
		}
	}
	return e
}

// Attribute returns a source attribute using its original spelling (is, id).
func (e *Entry) Attribute(name string) string {
	switch name {
	case "lemma":
		return e.Lemma
	case "flags":
		return e.Flags
	case "comment":
		return e.Comment
	case "err":
		return e.InvalidFields
	}
	return e.Fields[name]
}

// FieldToHunspell formats a space-separated source attribute.
func FieldToHunspell(name, value string) string {
	separator := " " + name + ":"
	return separator + strings.ReplaceAll(value, " ", separator)
}

// Morphology returns the source Hunspell field representation. Mode 2 includes metadata.
func (e *Entry) Morphology(mode int) string {
	var result strings.Builder
	for _, name := range morphologyFields {
		if value := e.Fields[name]; value != "" {
			result.WriteString(FieldToHunspell(name, value))
		}
	}
	if mode > 1 {
		for _, name := range []string{"lx", "se", "et"} {
			if value := e.Fields[name]; value != "" {
				result.WriteString(FieldToHunspell(name, value))
			}
		}
		if e.Fields["fq"] != "" {
			result.WriteString(" fq:" + e.Fields["fq"])
		}
		if e.Fields["di"] != "*" {
			result.WriteString(" di:" + e.Fields["di"])
		}
	}
	return result.String()
}

// LexicalMorphology returns the expanded morphology used during inflection.
func (e *Entry) LexicalMorphology() string { return lexicalMorphology(e.Fields) }

func lexicalMorphology(fields map[string]string) string {
	var result strings.Builder
	result.WriteByte(' ')
	for _, name := range lexicalFields {
		if value := fields[name]; value != "" {
			result.WriteString(value)
			result.WriteByte(' ')
		}
	}
	return result.String()
}

// HunspellLine formats an entry without optional Hunspell alias compression.
func (e *Entry) HunspellLine(mode int, simplified bool) string {
	line := strings.ReplaceAll(e.Lemma, "’", "'")
	if e.Flags != "" {
		line += "/" + e.Flags
	}
	if simplified {
		return strings.ReplaceAll(line, "()", "") + "\n"
	}
	if mode > 0 {
		line += e.Morphology(mode)
	}
	return line + "\n"
}

// NaturalKey is the exact three-part accent-folded lexical sort key.
func (e *Entry) NaturalKey() [3]string {
	return [3]string{NaturalSortText(e.Lemma), e.Flags, e.Fields["po"]}
}

// LongFlags splits two-code-point Hunspell flags, padding an incomplete final flag.
func LongFlags(flags string) []string {
	runes := []rune(flags)
	if len(runes)%2 != 0 {
		runes = append(runes, ' ')
	}
	result := make([]string, 0, len(runes)/2)
	for i := 0; i < len(runes); i += 2 {
		result = append(result, string(runes[i:i+2]))
	}
	return result
}

// NaturalSortText reproduces the source's intentionally limited accent folding.
func NaturalSortText(s string) string {
	return naturalSortReplacer.Replace(s)
}

var naturalSortReplacer = strings.NewReplacer(
	"à", "a", "À", "A", "â", "a", "Â", "A", "ä", "a", "Ä", "A", "å", "a", "Å", "A", "ā", "a", "Ā", "A",
	"ç", "c", "Ç", "C", "é", "e", "É", "E", "è", "e", "È", "E", "ê", "e", "Ê", "E", "ë", "e", "Ë", "E", "ē", "e", "Ē", "E",
	"î", "i", "Î", "I", "ï", "i", "Ï", "I", "ī", "i", "Ī", "I", "ñ", "n", "ô", "o", "Ô", "O", "ö", "o", "Ö", "O", "ō", "o", "Ō", "O",
	"ù", "u", "Ù", "U", "û", "u", "Û", "U", "ü", "u", "Ü", "U", "ū", "u", "Ū", "U", "ÿ", "y", "æ", "ae", "Æ", "AE", "œ", "oe", "Œ", "OE", "-", "", ".", "", "'", "")
