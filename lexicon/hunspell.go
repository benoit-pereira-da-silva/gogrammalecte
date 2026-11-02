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
)

// HunspellOptions controls the original morphology, compression and simplification modes.
type HunspellOptions struct {
	// Mode is 0 (no morphology), 1 (Hunspell morphology), or 2 (all metadata).
	Mode int
	// Aliases compress repeated morphology and continuation flags using AM and AF.
	Aliases bool
	// Simplified removes virtual lemmas and replication rules; aliases are ignored.
	Simplified bool
}

type dictionaryVariant struct{ name, subdictionaries string }

var hunspellVariants = map[string]dictionaryVariant{
	"fr-classique":       {"DICTIONNAIRE ORTHOGRAPHIQUE FRANÇAIS “CLASSIQUE”", "*MCX"},
	"fr-reforme1990":     {"DICTIONNAIRE ORTHOGRAPHIQUE FRANÇAIS “RÉFORME 1990”", "*RX"},
	"fr-toutesvariantes": {"DICTIONNAIRE ORTHOGRAPHIQUE FRANÇAIS “TOUTES VARIANTES”", "*MCRAX"},
}

// HunspellVariants returns the three original dictionary identifiers in build order.
func HunspellVariants() []string {
	return []string{"fr-toutesvariantes", "fr-classique", "fr-reforme1990"}
}

type hunspellAliases struct {
	flags, morphology            map[string]string
	flagOrder, morphologyOrder   []string
	flagCounts, morphologyCounts map[string]int
}

func (d *Dictionary) aliases(mode int) hunspellAliases {
	flagCounts, morphCounts := map[string]int{}, map[string]int{}
	add := func(flags, morph string) {
		if flags != "" {
			flagCounts[flags]++
		}
		morph = strings.TrimSpace(morph)
		if morph != "" {
			morphCounts[morph]++
		}
	}
	for _, flag := range d.Flags {
		for _, rule := range flag.Rules {
			add(rule.Flags, rule.Morphology(mode))
		}
	}
	for _, entry := range d.Entries {
		add(entry.Flags, entry.Morphology(mode))
	}
	flags, flagOrder := rankAliases(flagCounts)
	morphology, morphologyOrder := rankAliases(morphCounts)
	return hunspellAliases{flags, morphology, flagOrder, morphologyOrder, flagCounts, morphCounts}
}

// WriteAliasReport appends the original compression-frequency diagnostic tables.
func (d *Dictionary) WriteAliasReport(writer io.Writer, mode int) error {
	if mode < 0 || mode > 2 {
		return fmt.Errorf("Hunspell morphology mode must be 0, 1 or 2")
	}
	aliases := d.aliases(mode)
	w := bufio.NewWriter(writer)
	io.WriteString(w, "\n\nDrapeaux :\n")
	for _, flag := range aliases.flagOrder {
		fmt.Fprintf(w, "  > %8d : %s\n", aliases.flagCounts[flag], flag)
	}
	io.WriteString(w, "\n\nMorphologies :\n")
	for _, morph := range aliases.morphologyOrder {
		fmt.Fprintf(w, "  > %8d : %s\n", aliases.morphologyCounts[morph], morph)
	}
	return w.Flush()
}

func rankAliases(counts map[string]int) (map[string]string, []string) {
	order := make([]string, 0, len(counts))
	for key := range counts {
		order = append(order, key)
	}
	slices.SortFunc(order, func(a, b string) int {
		if counts[a] > counts[b] {
			return -1
		}
		if counts[a] < counts[b] {
			return 1
		}
		return -strings.Compare(a, b)
	})
	aliases := make(map[string]string, len(order))
	for i, key := range order {
		aliases[key] = strconv.Itoa(i + 1)
	}
	return aliases, order
}

// WriteHunspell emits one original .dic/.aff pair, preserving source ordering and
// alias ranking. The dictionary header retains the reference's pre-filter count.
func (d *Dictionary) WriteHunspell(dictionary, affixes io.Writer, variant, version string, options HunspellOptions) error {
	variantData, ok := hunspellVariants[variant]
	if !ok {
		return fmt.Errorf("unknown Hunspell variant %q", variant)
	}
	if options.Mode < 0 || options.Mode > 2 {
		return fmt.Errorf("Hunspell morphology mode must be 0, 1 or 2")
	}
	if options.Simplified {
		options.Aliases = false
	}
	var aliases hunspellAliases
	if options.Aliases {
		aliases = d.aliases(options.Mode)
	}
	dic, aff := bufio.NewWriter(dictionary), bufio.NewWriter(affixes)
	count := 0
	for _, entry := range d.Entries {
		if strings.Contains(variantData.subdictionaries, entry.Fields["di"]) && !strings.Contains(entry.Lemma, " ") {
			count++
		}
	}
	fmt.Fprintln(dic, count)
	for _, entry := range d.Entries {
		if !strings.Contains(variantData.subdictionaries, entry.Fields["di"]) || !validHunspellLemma(entry.Lemma) {
			continue
		}
		if !options.Aliases {
			io.WriteString(dic, entry.HunspellLine(options.Mode, options.Simplified))
			continue
		}
		line := strings.ReplaceAll(entry.Lemma, "’", "'")
		if entry.Flags != "" {
			line += "/" + aliases.flags[entry.Flags]
		}
		if options.Mode > 0 {
			if morph := entry.Morphology(options.Mode); morph != "" {
				line += "\t" + aliases.morphology[strings.TrimSpace(morph)]
			}
		}
		fmt.Fprintln(dic, line)
	}
	fmt.Fprintf(aff, "%s# AFFIXES DU %s v%s\n# par Olivier R. -- licence MPL 2.0\n# Pour améliorer le dictionnaire, allez sur https://grammalecte.net/\n\n", mplHeader, variantData.name, version)
	fmt.Fprintln(aff, d.Settings)
	if options.Aliases {
		fmt.Fprintf(aff, "AM %d\n", len(aliases.morphologyOrder))
		for _, morph := range aliases.morphologyOrder {
			fmt.Fprintln(aff, "AM "+morph)
		}
		fmt.Fprintln(aff)
		fmt.Fprintf(aff, "AF %d\n", len(aliases.flagOrder))
		for _, flag := range aliases.flagOrder {
			fmt.Fprintln(aff, "AF "+flag)
		}
		fmt.Fprintln(aff)
	}
	for _, flag := range d.Flags {
		var lines []string
		for _, rule := range flag.Rules {
			if !strings.Contains(variantData.subdictionaries, rule.Fields["di"]) || (options.Simplified && rule.IsReplication()) {
				continue
			}
			if !options.Aliases {
				lines = append(lines, rule.HunspellLine(options.Mode, options.Simplified))
				continue
			}
			kind := "PFX"
			if rule.Suffix {
				kind = "SFX"
			}
			addition := rule.Add
			if addition == "" {
				addition = "0"
			}
			line := kind + " " + rule.Name + " " + rule.Cut + " " + addition
			if rule.Flags != "" {
				line += "/" + aliases.flags[rule.Flags]
			}
			line += " " + rule.Condition
			if options.Mode > 0 {
				if morph := rule.Morphology(options.Mode); morph != "" {
					line += " " + aliases.morphology[strings.TrimSpace(morph)]
				}
			}
			lines = append(lines, line+"\n")
		}
		if len(lines) > 0 {
			kind, mix := "PFX", "N"
			if flag.Suffix {
				kind = "SFX"
			}
			if flag.Mix {
				mix = "Y"
			}
			fmt.Fprintf(aff, "\n%s %s %s %d\n", kind, flag.Name, mix, len(lines))
			for _, line := range lines {
				io.WriteString(aff, line)
			}
		}
	}
	if err := dic.Flush(); err != nil {
		return err
	}
	return aff.Flush()
}

func validHunspellLemma(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '_' && r != '’' && r != '\'' && r != '-' {
			return false
		}
	}
	return true
}
