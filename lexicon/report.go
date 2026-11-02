// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// Ported from Grammalecte genfrdic.py by Olivier R.

package lexicon

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

var reportVerbFlags = regexp.MustCompile(`(v[0-3])[itnpqrmaezx_]+`)
var subdictionaryNames = map[string]string{"*": "Commun", "R": "Réforme1990", "M": "Moderne", "C": "Classique", "A": "Annexe", "X": "Contributeurs"}

// CalculateFrequenciesReport calculates frequencies and writes the complete
// original statistical report, including every ambiguity-allocation decision.
func (d *Dictionary) CalculateFrequenciesReport(stats *Statistics, w io.Writer) error {
	report := bufio.NewWriter(w)
	io.WriteString(report, stats.Info())
	io.WriteString(report, "Report des occurrences des formes fléchies multiples :\n  Légende :\n    >>   le nombre d’occurrences de la flexion est ramené à la moyenne.\n    +>   le nombre d’occurrences de la flexion est augmenté avec le surplus d’occurrences des flexions ramenées à la moyenne.\n    %>   le nombre d’occurrences de la flexion est pondéré avec le poids de la moyenne de l’entrée.\n\n")
	d.calculateFrequencies(stats, report)
	io.WriteString(report, "\n\nNatures grammaticales :\n")
	parts := map[string]int64{}
	for _, e := range d.Entries {
		parts[reportVerbFlags.ReplaceAllString(e.Fields["po"], "${1}")]++
	}
	for _, key := range descendingCounts(parts) {
		fmt.Fprintf(report, " * %-15d : %s\n", parts[key], key)
	}
	io.WriteString(report, "\n\nVentilation des entrées par indice de fréquence :\n")
	counts, sums := map[string]int{}, map[string]float64{}
	for _, e := range d.Entries {
		counts[e.Fields["fq"]]++
		sums[e.Fields["fq"]] += e.Frequency
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		fmt.Fprintf(report, " * %s : %d entrées (%.2f %%)  → %.9f %%\n", key, counts[key], float64(counts[key]*100)/float64(len(d.Entries)), sums[key])
	}
	io.WriteString(report, "\n\nRépartition des entrées par sous-dictionnaire :\n")
	counts = map[string]int{}
	keys = nil
	for _, e := range d.Entries {
		key := e.Fields["di"]
		if _, found := counts[key]; !found {
			keys = append(keys, key)
		}
		counts[key]++
	}
	for _, key := range keys {
		fmt.Fprintf(report, " * %-15s : %d entrées (%.2f %%)\n", subdictionaryNames[key], counts[key], float64(counts[key]*100)/float64(len(d.Entries)))
	}
	letters := map[string]int64{}
	var total int64
	spellings := map[string]bool{}
	for _, f := range d.Flexions {
		spellings[f.Text] = true
		for _, r := range f.Text {
			letters[string(r)] += f.Occurrences
			total += f.Occurrences
		}
	}
	if total != 0 {
		io.WriteString(report, "\n\nOccurrences des lettres dans le corpus :\n")
		for _, letter := range descendingCounts(letters) {
			fmt.Fprintf(report, "   %s : %16s  /  %.8f %%\n", letter, commaNumber(letters[letter]), float64(letters[letter]*100)/float64(total))
		}
	}
	lengths := map[int]int{}
	for word := range spellings {
		lengths[utf8.RuneCountInString(word)]++
	}
	sizes := make([]int, 0, len(lengths))
	for size := range lengths {
		sizes = append(sizes, size)
	}
	slices.Sort(sizes)
	io.WriteString(report, "\n\nNombre de lettres dans les graphies :\n")
	for _, size := range sizes {
		fmt.Fprintf(report, "   %2d lettres : %8d graphies\n", size, lengths[size])
	}
	fmt.Fprintf(report, "\n\nNombre de formes fléchies : %d\n\n\nNombre de graphies : %d\n", len(d.Flexions), len(spellings))
	return report.Flush()
}

func descendingCounts(counts map[string]int64) []string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b string) int {
		if counts[a] > counts[b] {
			return -1
		}
		if counts[a] < counts[b] {
			return 1
		}
		return -strings.Compare(a, b)
	})
	return keys
}
