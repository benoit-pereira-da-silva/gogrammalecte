// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// Ported from Grammalecte genfrdic.py by Olivier R.

package lexicon

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/strtransform"
)

// VerbMultiMorph combines tense tags with subsequent person tags in source order.
func VerbMultiMorph(morphology string) []string {
	tags := strings.Fields(morphology)
	result := []string{}
	for i, tag := range tags {
		if firstIsDigit(tag) {
			break
		}
		for _, person := range tags[i+1:] {
			if firstIsDigit(person) {
				result = append(result, tag+" "+person)
			}
		}
	}
	return result
}
func firstIsDigit(s string) bool {
	for _, r := range s {
		return strtransform.IsDigit(string(r))
	}
	return false
}
func runeSlice(s string, start, end int) string {
	r := []rune(s)
	if end < 0 || end > len(r) {
		end = len(r)
	}
	if start > len(r) {
		start = len(r)
	}
	return string(r[start:end])
}

// Conjugation formats the source conjugation-generator input for this entry.
func (e *Entry) Conjugation() string {
	var result strings.Builder
	result.WriteString(e.Lemma + "\t" + runeSlice(e.Fields["po"], 1, 10) + "\n")
	for _, f := range e.Flexions {
		morph := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(strings.TrimRight(runeSlice(f.Morphology, 11, -1), "!"), "ppas adj", "ppas"), "1jsg", ""))
		if !strings.HasPrefix(morph, "ppas") && strings.Index(morph, " ") > 1 {
			for _, part := range VerbMultiMorph(morph) {
				result.WriteString("_\t" + part + "\t" + f.Text + "\n")
			}
		} else {
			result.WriteString("_\t" + morph + "\t" + f.Text + "\n")
		}
	}
	result.WriteString("$\n")
	return result.String()
}

// Declension formats the source declension-generator input for this entry.
func (e *Entry) Declension() string {
	var result strings.Builder
	result.WriteString(e.Lemma + "\t" + e.Flags + "\n")
	for _, f := range e.Flexions {
		if strings.Contains(f.Morphology, "adj") || strings.Contains(f.Morphology, "nom") {
			result.WriteString("_\t" + f.Morphology + "\t" + f.Text + "\n")
		}
	}
	result.WriteString("$\n")
	return result.String()
}

// WriteConjugations emits the complete dictConj.txt generator source.
func (d *Dictionary) WriteConjugations(w io.Writer) error {
	for _, e := range d.Entries {
		if strings.HasPrefix(e.Fields["po"], "v") {
			if _, err := io.WriteString(w, e.Conjugation()); err != nil {
				return err
			}
		}
	}
	return nil
}

// WriteDeclensions emits the complete dictDecl.txt generator source.
func (d *Dictionary) WriteDeclensions(w io.Writer) error {
	for _, e := range d.Entries {
		if e.Flags != "" && strings.ContainsRune("SXFWIA", []rune(e.Flags)[0]) && (strings.HasPrefix(e.Fields["po"], "nom") || strings.HasPrefix(e.Fields["po"], "adj")) {
			if _, err := io.WriteString(w, e.Declension()); err != nil {
				return err
			}
		}
	}
	return nil
}

const mplHeader = "# This Source Code Form is subject to the terms of the Mozilla Public\n# License, v. 2.0. If a copy of the MPL was not distributed with this\n# file, You can obtain one at http://mozilla.org/MPL/2.0/.\n\n"

// WriteGrammarLexicon writes a French.lex-compatible source lexicon in current order.
func (d *Dictionary) WriteGrammarLexicon(w io.Writer, version string) error {
	if _, err := fmt.Fprintf(w, "%s# Lexique simplifié pour Grammalecte v%s\n# Licence : MPL v2.0\n\n# :POS ;LEX ~SEM =FQ /DIC\n", mplHeader, version); err != nil {
		return err
	}
	for _, f := range d.Flexions {
		if _, err := io.WriteString(w, f.GrammarCheckerLine()); err != nil {
			return err
		}
	}
	return nil
}

// LexiconHeader returns the source lexical table header, including corpus columns.
func LexiconHeader(stats *Statistics) string {
	var corpora strings.Builder
	for _, corpus := range stats.corpora {
		corpora.WriteString(corpus.Name + "\t")
	}
	return "id\tfid\tFlexion\tLemme\tÉtiquettes\tMétagraphe (β)\tMetaphone2\tNotes\tSémantique\tÉtymologie\tSous-dictionnaire\t" + corpora.String() + "Total occurrences\tDoublons\tMultiples\tFréquence\tIndice de fréquence\n"
}

// LexiconLine formats every source lexical-table column, including frequencies.
func (f *Flexion) LexiconLine(stats *Statistics) string {
	var counts strings.Builder
	for _, count := range stats.counts[f.Text] {
		fmt.Fprintf(&counts, "%d\t", count)
	}
	dictionary := f.Entry.Fields["di"]
	if f.Dictionary != "*" {
		dictionary += "/" + f.Dictionary
	}
	return fmt.Sprintf("%s\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s%d\t%d\t%d\t%.15f\t%s\n", f.Entry.Fields["id"], f.Index, f.Text, f.Entry.Stem, f.Morphology, f.MetagrapheCode, f.MetaphoneCode, f.Entry.Fields["lx"], f.Entry.Fields["se"], f.Entry.Fields["et"], dictionary, counts.String(), f.Occurrences, f.Duplicates, len(f.OtherEntries), f.Frequency, f.FrequencyIndex)
}

// WriteLexicon emits the full lexical table with its source attribution and corpus summary.
// Generate, CalculatePhonetics and CalculateFrequencies populate its data columns.
func (d *Dictionary) WriteLexicon(w io.Writer, version string, stats *Statistics) error {
	buffer := bufio.NewWriter(w)
	fmt.Fprintf(buffer, "%s# Lexique des formes fléchies du français - Grammalecte v%s\n# Licence : MPL v2.0\n\n", mplHeader, version)
	io.WriteString(buffer, stats.Info())
	io.WriteString(buffer, LexiconHeader(stats))
	for _, f := range d.Flexions {
		io.WriteString(buffer, f.LexiconLine(stats))
	}
	return buffer.Flush()
}

// WriteFrequencyChanges emits the original database index changes and readable notes.
func (d *Dictionary) WriteFrequencyChanges(changes, notes io.Writer) error {
	diff, description := bufio.NewWriter(changes), bufio.NewWriter(notes)
	for _, e := range d.Entries {
		if e.Fields["fq"] != e.OldFrequencyIndex {
			fmt.Fprintf(diff, "%s\t%s\n", e.Fields["id"], e.Fields["fq"])
			fmt.Fprintf(description, "%s/%s\t%s > %s\n", e.Lemma, e.Flags, e.OldFrequencyIndex, e.Fields["fq"])
		}
	}
	if err := diff.Flush(); err != nil {
		return err
	}
	return description.Flush()
}
