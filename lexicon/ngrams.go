// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// Ported from Grammalecte genfrdic.py by Olivier R.

package lexicon

import (
	"fmt"
	"slices"
)

// LexicalNgrams preserves the lexical generator's historical len(word)-n-1
// range, which intentionally differs from graphspell's general ngram helper.
func LexicalNgrams(word string, n int) []string {
	runes := []rune(word)
	result := []string{}
	for i := 0; i < len(runes)-n-1; i++ {
		start, end := i, i+n
		if end < 0 {
			end += len(runes)
		}
		if end < 0 {
			end = 0
		}
		if end > len(runes) {
			end = len(runes)
		}
		if start > len(runes) {
			start = len(runes)
		}
		if end < start {
			result = append(result, "")
		} else {
			result = append(result, string(runes[start:end]))
		}
	}
	return result
}

// Ngram records occurrence counts and the distinct words containing the sequence.
type Ngram struct {
	Text        string
	Count       int
	WordIndices []int
}

// NgramIndex retains the source's first-appearance ordering for words and sequences.
type NgramIndex struct {
	Words []string
	Grams []Ngram
}

// BuildNgrams builds the source index; word references are stored in ascending
// insertion order instead of exposing Python's implementation-specific set order.
func (d *Dictionary) BuildNgrams(n int) (*NgramIndex, error) {
	if n < 2 {
		return nil, fmt.Errorf("lexical ngram length must be at least two")
	}
	result := &NgramIndex{Words: []string{}, Grams: []Ngram{}}
	words, grams := map[string]int{}, map[string]int{}
	for _, flexion := range d.Flexions {
		for _, text := range LexicalNgrams(flexion.Text, n) {
			wordIndex, found := words[flexion.Text]
			if !found {
				wordIndex = len(result.Words)
				words[flexion.Text] = wordIndex
				result.Words = append(result.Words, flexion.Text)
			}
			gramIndex, found := grams[text]
			if !found {
				gramIndex = len(result.Grams)
				grams[text] = gramIndex
				result.Grams = append(result.Grams, Ngram{Text: text, WordIndices: []int{}})
			}
			gram := &result.Grams[gramIndex]
			gram.Count++
			if !slices.Contains(gram.WordIndices, wordIndex) {
				gram.WordIndices = append(gram.WordIndices, wordIndex)
			}
		}
	}
	return result, nil
}
