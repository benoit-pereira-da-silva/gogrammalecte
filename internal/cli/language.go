package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/benoit-pereira-da-silva/gogrammalecte/french"
	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell"
	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/dictionary"
	"github.com/benoit-pereira-da-silva/gogrammalecte/rules"
	"github.com/benoit-pereira-da-silva/gogrammalecte/subtitle"
	"github.com/spf13/cobra"
)

func languageCommands() []*cobra.Command {
	lookup := &cobra.Command{Use: "lookup word", Short: "Look up a word in the embedded French dictionary", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		d, err := dictionary.Default()
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(struct {
			Word         string   `json:"word"`
			Valid        bool     `json:"valid"`
			Morphologies []string `json:"morphologies"`
		}{args[0], d.IsValidToken(args[0]), d.Morph(args[0])})
	}}
	conjugate := &cobra.Command{Use: "conjugate verb tense person", Short: "Return a French conjugated form from embedded resources", Args: cobra.ExactArgs(3), RunE: func(cmd *cobra.Command, args []string) error {
		form, err := french.GetConjugation(args[0], args[1], args[2])
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(form)
	}}
	inventory := &cobra.Command{Use: "rules", Short: "Inspect the embedded grammar source inventory", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		source, err := rules.FrenchSource()
		if err != nil {
			return err
		}
		doc, err := rules.Parse(source)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(struct {
			Tests, RegexRules, Graphs int
			Status                    string
		}{len(doc.Tests), len(doc.RegexRules), len(doc.Graphs), "Native grammar engine and resource generators available."})
	}}
	options := &cobra.Command{Use: "options", Short: "Show grammar option defaults from the embedded source", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		source, err := rules.FrenchSource()
		if err != nil {
			return err
		}
		doc, err := rules.Parse(source)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(doc.Defaults)
	}}
	var format string
	spell := &cobra.Command{Use: "spell [file]", Short: "Find spelling errors in UTF-8 prose or subtitle payloads", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		input, kind, err := readDocument(cmd, args, format)
		if err != nil {
			return err
		}
		if kind != "" {
			result, err := subtitle.Spell(cmd.Context(), input, kind)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		}
		checker, err := graphspell.Default()
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(checker.ParseParagraph(input))
	}}
	documentFormatFlag(spell, &format)

	var limit int
	suggest := &cobra.Command{Use: "suggest word", Short: "Suggest spellings using embedded French resources", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if limit < 1 {
			return fmt.Errorf("limit must be positive")
		}
		checker, err := graphspell.Default()
		if err != nil {
			return err
		}
		groups, err := checker.Suggest(cmd.Context(), args[0], limit)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(groups)
	}}
	suggest.Flags().IntVar(&limit, "limit", 10, "Maximum generated suggestions per main dictionary group")
	return []*cobra.Command{lookup, conjugate, inventory, options, spell, suggest, dictionaryBuildCommand()}
}

func dictionaryBuildCommand() *cobra.Command {
	var source, stemming, filter string
	cmd := &cobra.Command{Use: "build-dictionary", Short: "Generate a dictionary from the embedded lexicon or an explicit source", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		var input io.Reader
		switch source {
		case "":
			r, err := dictionary.BundledLexicon()
			if err != nil {
				return err
			}
			defer r.Close()
			input = r
		case "-":
			input = cmd.InOrStdin()
		default:
			f, err := os.Open(source)
			if err != nil {
				return fmt.Errorf("open lexicon: %w", err)
			}
			defer f.Close()
			input = f
		}
		entries, meta, err := dictionary.ReadLexicon(input)
		if err != nil {
			return err
		}
		built, err := dictionary.Build(entries, dictionary.BuildOptions{Metadata: meta, Stemming: stemming, TagPattern: filter})
		if err != nil {
			return err
		}
		return built.Write(cmd.OutOrStdout())
	}}
	cmd.Flags().StringVar(&source, "source", "", "Optional three-column lexicon path, or - for standard input; defaults to the embedded lexicon")
	cmd.Flags().StringVar(&stemming, "stemming", "S", "Stemming format: S (suffix), A (affix), N (literal)")
	cmd.Flags().StringVar(&filter, "tag-pattern", "", "Optional standard-library regular expression filtering morphological tags")
	return cmd
}
