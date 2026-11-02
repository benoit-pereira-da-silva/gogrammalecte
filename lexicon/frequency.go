// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// Ported from Grammalecte genfrdic.py by Olivier R.

package lexicon

import (
	"fmt"
	"io"
	"math"
	"slices"
	"strings"
)

// EmbeddedStatistics reads the four original corpora in the source generator's order.
func (d *Dictionary) EmbeddedStatistics() (*Statistics, error) {
	words := make([]string, len(d.Flexions))
	for i, f := range d.Flexions {
		words[i] = f.Text
	}
	stats := NewStatistics(words)
	for _, corpus := range []struct{ file, id, name string }{{"stats_google_ngram_1.txt", "G", "Google 1-grams"}, {"stats_frwiki.txt", "W", "Wikipédia"}, {"stats_frwikisource.txt", "S", "Wikisource"}, {"stats_litterature.txt", "L", "Littérature"}} {
		r, err := Corpus(corpus.file)
		if err != nil {
			return nil, err
		}
		err = stats.Add(r, corpus.id, corpus.name)
		closeErr := r.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
	}
	return stats, nil
}

// FrequencyIndex returns the original one-character logarithmic frequency band.
func FrequencyIndex(frequency float64) string {
	if frequency == 0 {
		return "0"
	}
	for i, threshold := range []float64{1e-8, 1e-7, 1e-6, 1e-5, 1e-4, 1e-3, 1e-2, 0.1} {
		if frequency < threshold {
			return string(rune('1' + i))
		}
	}
	return "9"
}

// CalculateFrequencies applies the reference's ordered ambiguity allocation.
// Entry order matters; call SortNatural first to reproduce the upstream build.
func (d *Dictionary) CalculateFrequencies(stats *Statistics) {
	d.calculateFrequencies(stats, nil)
}

func (d *Dictionary) calculateFrequencies(stats *Statistics, report io.Writer) {
	d.RecognizedOccurrences = 0
	for _, e := range d.Entries {
		e.AverageKnownOccurrences = -1
	}
	for _, f := range d.Flexions {
		f.Occurrences = int64(math.Ceil(float64(stats.Occurrences(f.Text)) / float64(len(f.OtherEntries)+1) / float64(f.Duplicates)))
		f.Blocked = false
		d.RecognizedOccurrences += f.Occurrences
	}
	for _, e := range d.Entries {
		e.sumOccurrences()
		var known, count int64
		for _, f := range e.Flexions {
			if len(f.OtherEntries) == 0 {
				known += f.Occurrences
				count++
			}
		}
		if count > 0 {
			e.AverageKnownOccurrences = int64(math.Ceil(float64(known) / float64(count)))
		}
		e.allocateAmbiguousOccurrences(stats, report)
		e.sumOccurrences()
	}
	for _, f := range d.Flexions {
		f.Frequency = frequency(f.Occurrences, d.RecognizedOccurrences)
		f.FrequencyIndex = FrequencyIndex(f.Frequency)
	}
	for _, e := range d.Entries {
		e.Frequency = frequency(e.Occurrences, d.RecognizedOccurrences)
		e.OldFrequencyIndex = e.Fields["fq"]
		e.Fields["fq"] = FrequencyIndex(e.Frequency)
	}
}

func frequency(count, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(count*100) / float64(total)
}
func (e *Entry) sumOccurrences() {
	e.Occurrences = 0
	for _, f := range e.Flexions {
		e.Occurrences += f.Occurrences
	}
}
func (f *Flexion) blockOccurrences(count int64) { f.Occurrences = count; f.Blocked = true }

func (e *Entry) allocateAmbiguousOccurrences(stats *Statistics, report io.Writer) {
	if e.AverageKnownOccurrences < 0 {
		return
	}
	for _, f := range e.Flexions {
		if len(f.OtherEntries) == 0 || f.Blocked {
			continue
		}
		var withAverage, withoutAverage []*Entry
		for _, other := range f.OtherEntries {
			if other.AverageKnownOccurrences >= 0 {
				withAverage = append(withAverage, other)
			} else {
				withoutAverage = append(withoutAverage, other)
			}
		}
		if len(withoutAverage) > 0 {
			difference := (f.Occurrences - e.AverageKnownOccurrences) * int64(f.Duplicates)
			for _, other := range withAverage {
				for _, matching := range other.Flexions {
					if matching.Text == f.Text {
						difference += matching.Occurrences - other.AverageKnownOccurrences
					}
				}
			}
			if difference > 0 {
				if report != nil {
					fmt.Fprintf(report, " * %s\n       moyenne connue\n", f.Text)
				}
				for _, matching := range e.Flexions {
					if matching.Text == f.Text {
						occurrenceReport(report, matching, e, ">>", e.AverageKnownOccurrences)
						matching.blockOccurrences(e.AverageKnownOccurrences)
					}
				}
				for _, other := range withAverage {
					if report != nil {
						io.WriteString(report, "       moyenne connue\n")
					}
					for _, matching := range other.Flexions {
						if matching.Text == f.Text {
							occurrenceReport(report, matching, other, ">>", other.AverageKnownOccurrences)
							matching.blockOccurrences(other.AverageKnownOccurrences)
						}
					}
				}
				for _, other := range withoutAverage {
					if report != nil {
						io.WriteString(report, "       sans moyenne connue\n")
					}
					for _, matching := range other.Flexions {
						if matching.Text == f.Text {
							count := matching.Occurrences + int64(math.Ceil(float64(difference)/float64(len(withoutAverage))/float64(matching.Duplicates)))
							occurrenceReport(report, matching, other, "+>", count)
							matching.blockOccurrences(count)
						}
					}
				}
			}
		} else {
			if report != nil {
				fmt.Fprintf(report, " = %s\n       moyennes connues\n", f.Text)
			}
			occurrences := stats.Occurrences(f.Text)
			totalAverage := e.AverageKnownOccurrences
			for _, other := range f.OtherEntries {
				totalAverage += other.AverageKnownOccurrences
			}
			allocate := func(entry *Entry) {
				for _, matching := range entry.Flexions {
					if matching.Text == f.Text {
						var count int64
						if totalAverage != 0 {
							count = int64(math.Ceil((float64(occurrences) * (float64(entry.AverageKnownOccurrences) / float64(totalAverage))) / float64(matching.Duplicates)))
						}
						occurrenceReport(report, matching, entry, "%>", count)
						matching.blockOccurrences(count)
					}
				}
			}
			allocate(e)
			for _, other := range f.OtherEntries {
				allocate(other)
			}
		}
	}
}

func occurrenceReport(w io.Writer, f *Flexion, e *Entry, indicator string, count int64) {
	if w != nil {
		fmt.Fprintf(w, "           %-30s %-30s  %10d  %s %10d\n", e.ShortDescription(), f.Morphology, f.Occurrences, indicator, count)
	}
}

// SortByFrequency stably applies the reference's frequency, stem and spelling order.
func (d *Dictionary) SortByFrequency() {
	slices.SortStableFunc(d.Flexions, func(a, b *Flexion) int {
		ka, kb := 100-a.Frequency, 100-b.Frequency
		if ka < kb {
			return -1
		}
		if ka > kb {
			return 1
		}
		if c := strings.Compare(a.Entry.Stem, b.Entry.Stem); c != 0 {
			return c
		}
		return strings.Compare(a.Text, b.Text)
	})
}
