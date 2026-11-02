// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// Ported from Grammalecte genfrdic.py by Olivier R.

package lexicon

import (
	"bufio"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// CorpusStatistics describes the totals for one added frequency corpus.
type CorpusStatistics struct {
	ID, Name     string
	Known, Total int64
}

// Statistics accumulates corpus occurrences for a fixed set of known inflections.
type Statistics struct {
	counts  map[string][]int64
	corpora []CorpusStatistics
	words   []string
}

// NewStatistics creates empty per-word counts, eliminating repeated known words.
func NewStatistics(words []string) *Statistics {
	s := &Statistics{counts: map[string][]int64{}}
	for _, word := range words {
		if _, found := s.counts[word]; !found {
			s.words = append(s.words, word)
		}
		s.counts[word] = []int64{}
	}
	return s
}

// Write emits the original corpus metadata and per-inflection count dump.
func (s *Statistics) Write(writer io.Writer) error {
	w := bufio.NewWriter(writer)
	for _, corpus := range s.corpora {
		fmt.Fprintf(w, "(%s, %s, %d, %d)\n", pythonRepr(corpus.ID), pythonRepr(corpus.Name), corpus.Known, corpus.Total)
	}
	for _, word := range s.words {
		counts := make([]string, len(s.counts[word]))
		for i, count := range s.counts[word] {
			counts[i] = strconv.FormatInt(count, 10)
		}
		fmt.Fprintf(w, "%s - [%s]\n", word, strings.Join(counts, ", "))
	}
	return w.Flush()
}

func pythonRepr(s string) string {
	quote := '\''
	if strings.ContainsRune(s, '\'') && !strings.ContainsRune(s, '"') {
		quote = '"'
	}
	var result strings.Builder
	result.WriteRune(quote)
	for _, r := range s {
		switch r {
		case '\\':
			result.WriteString(`\\`)
		case '\n':
			result.WriteString(`\n`)
		case '\r':
			result.WriteString(`\r`)
		case '\t':
			result.WriteString(`\t`)
		default:
			if r == quote {
				result.WriteRune('\\')
				result.WriteRune(r)
			} else if unicode.IsPrint(r) {
				result.WriteRune(r)
			} else if r <= 0xff {
				fmt.Fprintf(&result, `\x%02x`, r)
			} else if r <= 0xffff {
				fmt.Fprintf(&result, `\u%04x`, r)
			} else {
				fmt.Fprintf(&result, `\U%08x`, r)
			}
		}
	}
	result.WriteRune(quote)
	return result.String()
}

// Add consumes a two-column frequency corpus. Duplicate corpus rows retain the
// reference's append behavior; they are not combined or silently overwritten.
func (s *Statistics) Add(r io.Reader, id, name string) error {
	if utf8.RuneCountInString(id) != 1 {
		return fmt.Errorf("corpus identifier must contain one character: %q", id)
	}
	metadata := CorpusStatistics{ID: id, Name: name}
	scanner := newScanner(r)
	line := 0
	for scanner.Scan() {
		line++
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 {
			return fmt.Errorf("corpus %s line %d: expected word and count", id, line)
		}
		count, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return fmt.Errorf("corpus %s line %d: %w", id, line, err)
		}
		if _, known := s.counts[fields[0]]; known {
			s.counts[fields[0]] = append(s.counts[fields[0]], count)
			metadata.Known += count
		}
		metadata.Total += count
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	s.corpora = append(s.corpora, metadata)
	for word, counts := range s.counts {
		if len(counts) < len(s.corpora) {
			s.counts[word] = append(counts, 0)
		}
	}
	return nil
}

// Counts returns the stored counts without exposing mutable internal state.
func (s *Statistics) Counts(word string) []int64 { return slices.Clone(s.counts[word]) }

// Occurrences returns the sum of the stored corpus counts for an inflection.
func (s *Statistics) Occurrences(word string) int64 {
	var total int64
	for _, count := range s.counts[word] {
		total += count
	}
	return total
}

// Corpora returns corpus metadata in insertion order.
func (s *Statistics) Corpora() []CorpusStatistics { return slices.Clone(s.corpora) }

// Info returns the original corpus summary format.
func (s *Statistics) Info() string {
	var result strings.Builder
	result.WriteString("Corpus :\n")
	var known, total int64
	for _, c := range s.corpora {
		fmt.Fprintf(&result, " * %-20s -> %18s mots reconnus / %18s\n", c.Name, commaNumber(c.Known), commaNumber(c.Total))
		known += c.Known
		total += c.Total
	}
	fmt.Fprintf(&result, "\n * %-20s -> %18s mots reconnus / %18s\n\n", "TOTAL", commaNumber(known), commaNumber(total))
	return result.String()
}

func commaNumber(value int64) string {
	s := strconv.FormatInt(value, 10)
	start := 0
	if s[0] == '-' {
		start = 1
	}
	for i := len(s) - 3; i > start; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
