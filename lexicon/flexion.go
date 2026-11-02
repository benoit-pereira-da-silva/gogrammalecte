// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// Ported from Grammalecte genfrdic.py by Olivier R.

package lexicon

import "strings"

// Flexion is one inflected form, retaining its source entry and ambiguity counts.
type Flexion struct {
	Entry                        *Entry
	Text, Morphology, Dictionary string
	Index, Duplicates            int
	OtherEntries                 []*Entry
	Occurrences                  int64
	Blocked                      bool
	Frequency                    float64
	FrequencyIndex               string
	MetagrapheCode               string
	MetaphoneCode                string
}

// CalculatePhonetics fills both phonetic columns using native implementations.
func (d *Dictionary) CalculatePhonetics() {
	for _, f := range d.Flexions {
		metagraphe, metaphone := Metagraphe(f.Text, f.Morphology), DoubleMetaphone(f.Text)
		f.MetagrapheCode = phoneticCode(metagraphe)
		f.MetaphoneCode = phoneticCode(metaphone)
	}
}

func phoneticCode(codes [2]string) string {
	if codes[1] == "" {
		return codes[0]
	}
	return codes[0] + "/" + codes[1]
}

// SimpleTags converts the lexical morphology to grammar-checker tags.
func (f *Flexion) SimpleTags() string {
	var result strings.Builder
	for _, tag := range strings.Fields(f.Morphology) {
		if strings.HasPrefix(tag, "v") {
			result.WriteString(":V" + tag[1:])
		} else {
			result.WriteString(tagReplacement[tag])
		}
	}
	for _, field := range []string{"lx", "se"} {
		for _, tag := range strings.Fields(f.Entry.Fields[field]) {
			result.WriteString(tagReplacement[tag])
		}
	}
	dictionary := f.Entry.Fields["di"]
	if dictionary == "*" && f.Dictionary != "*" {
		dictionary = f.Dictionary
	}
	result.WriteString("/" + dictionary)
	return result.String()
}

// GrammarCheckerLine formats a tab-separated word, lemma and tag row.
func (f *Flexion) GrammarCheckerLine() string {
	return f.Text + "\t" + f.Entry.Lemma + "\t" + f.SimpleTags() + "\n"
}
