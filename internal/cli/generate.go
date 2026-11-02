package cli

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/benoit-pereira-da-silva/gogrammalecte/french"
	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/dictionary"
	"github.com/benoit-pereira-da-silva/gogrammalecte/lexicon"
	"github.com/benoit-pereira-da-silva/gogrammalecte/rules"
	"github.com/spf13/cobra"
)

func generationCommand() *cobra.Command {
	var version, variant string
	var mode, ngramLength int
	var aliases, simplified bool
	cmd := &cobra.Command{Use: "generate target", Short: "Regenerate resources entirely in Go using embedded sources", Long: "Targets: rules, inflections, conjugations, phonetics, thesaurus, thesaurus-source, mythes, lexicon, lexical-table, verb-source, declension-source, hunspell, statistics, validation, ngrams, corpus-counts, frequency-changes, aliases. Output is written to standard output. Hunspell and MyThes emit ZIP archives.", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := cmd.Context().Err(); err != nil {
			return err
		}
		write := func(data []byte, err error) error {
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		}
		switch args[0] {
		case "rules":
			source, err := rules.FrenchSource()
			if err != nil {
				return err
			}
			doc, err := rules.Parse(source)
			if err != nil {
				return err
			}
			graphs, err := rules.CompileGraphs(doc)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(struct {
				Source *rules.Document
				Graphs []rules.CompiledGraph
			}{doc, graphs})
		case "inflections":
			return write(french.GenerateInflectionData())
		case "conjugations", "phonetics":
			d, err := dictionary.Default()
			if err != nil {
				return err
			}
			if args[0] == "conjugations" {
				return write(french.GenerateConjugationData(d))
			}
			return write(french.GeneratePhoneticData(d))
		case "thesaurus":
			return write(french.GenerateThesaurusData())
		case "thesaurus-source", "mythes":
			t, err := lexicon.EmbeddedThesaurus()
			if err != nil {
				return err
			}
			if args[0] == "thesaurus-source" {
				return t.WriteJSON(cmd.OutOrStdout())
			}
			var data, index bytes.Buffer
			if err := t.WriteMyThes(&data, &index); err != nil {
				return err
			}
			return writeResourceZIP(cmd.OutOrStdout(), []archiveResource{{"thes_fr.dat", data.Bytes()}, {"thes_fr.idx", index.Bytes()}})
		case "lexicon", "lexical-table", "verb-source", "declension-source", "hunspell", "statistics", "validation", "ngrams", "corpus-counts", "frequency-changes", "aliases":
			d, err := lexicon.EmbeddedDictionary()
			if err != nil {
				return err
			}
			d.SortNatural()
			if args[0] == "validation" {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(d.ValidationMessages())
			}
			if err := d.Generate(); err != nil {
				return err
			}
			if args[0] == "ngrams" {
				index, err := d.BuildNgrams(ngramLength)
				if err != nil {
					return err
				}
				return json.NewEncoder(cmd.OutOrStdout()).Encode(index)
			}
			if args[0] == "verb-source" {
				return d.WriteConjugations(cmd.OutOrStdout())
			}
			if args[0] == "declension-source" {
				return d.WriteDeclensions(cmd.OutOrStdout())
			}
			stats, err := d.EmbeddedStatistics()
			if err != nil {
				return err
			}
			if args[0] == "statistics" {
				return d.CalculateFrequenciesReport(stats, cmd.OutOrStdout())
			}
			if args[0] == "corpus-counts" {
				return stats.Write(cmd.OutOrStdout())
			}
			d.CalculateFrequencies(stats)
			if args[0] == "aliases" {
				return d.WriteAliasReport(cmd.OutOrStdout(), mode)
			}
			if args[0] == "frequency-changes" {
				var changes, notes bytes.Buffer
				if err := d.WriteFrequencyChanges(&changes, &notes); err != nil {
					return err
				}
				return writeResourceZIP(cmd.OutOrStdout(), []archiveResource{{"frequency-changes.diff", changes.Bytes()}, {"frequency-notes.txt", notes.Bytes()}})
			}
			if args[0] == "hunspell" {
				var dic, aff bytes.Buffer
				if err := d.WriteHunspell(&dic, &aff, variant, version, lexicon.HunspellOptions{Mode: mode, Aliases: aliases, Simplified: simplified}); err != nil {
					return err
				}
				return writeHunspellZIP(cmd.OutOrStdout(), variant, dic.Bytes(), aff.Bytes())
			}
			d.SortByFrequency()
			if args[0] == "lexical-table" {
				d.CalculatePhonetics()
				return d.WriteLexicon(cmd.OutOrStdout(), version, stats)
			}
			return d.WriteGrammarLexicon(cmd.OutOrStdout(), version)
		default:
			return fmt.Errorf("unknown generation target %q", args[0])
		}
	}}
	cmd.Flags().StringVar(&version, "resource-version", "7.7", "Version written into generated lexical resources")
	cmd.Flags().StringVar(&variant, "variant", "fr-toutesvariantes", "Hunspell variant: fr-toutesvariantes, fr-classique or fr-reforme1990")
	cmd.Flags().IntVar(&mode, "morphology", 0, "Hunspell morphology mode: 0, 1 or 2")
	cmd.Flags().BoolVar(&aliases, "aliases", false, "Use Hunspell morphology and flag aliases")
	cmd.Flags().BoolVar(&simplified, "simplified", false, "Generate a simplified Hunspell dictionary")
	cmd.Flags().IntVar(&ngramLength, "ngram-length", 3, "Sequence length for lexical ngram generation")
	return cmd
}
func writeHunspellZIP(output io.Writer, name string, dic, aff []byte) error {
	return writeResourceZIP(output, []archiveResource{{name + ".dic", dic}, {name + ".aff", aff}})
}

type archiveResource struct {
	name string
	data []byte
}

func writeResourceZIP(output io.Writer, resources []archiveResource) error {
	archive := zip.NewWriter(output)
	for _, item := range resources {
		entry, err := archive.Create(item.name)
		if err != nil {
			return err
		}
		if _, err = entry.Write(item.data); err != nil {
			return err
		}
	}
	return archive.Close()
}
func corporaCommand() *cobra.Command {
	return &cobra.Command{Use: "corpora", Short: "List embedded original frequency corpora", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(lexicon.CorpusNames())
	}}
}
