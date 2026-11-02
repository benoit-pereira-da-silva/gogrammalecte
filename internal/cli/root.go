// Package cli implements the gogrammalecte command line interface.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/benoit-pereira-da-silva/gogrammalecte/text"
	"github.com/spf13/cobra"
)

// NewCommand constructs an independent command tree for the supplied version.
func NewCommand(version string) *cobra.Command {
	root := &cobra.Command{
		Use:           "gogrammalecte",
		Short:         "Go port of Grammalecte's French language tools",
		Long:          "Go port of Grammalecte's French language tools with embedded grammar, dictionaries and resource generators.",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(
		textCommand("paragraphs", "Split UTF-8 standard input into paragraphs and emit a JSON array", text.Paragraphs),
		textCommand("sentences", "Split UTF-8 standard input into sentences and emit a JSON array", text.Sentences),
	)
	root.AddCommand(languageCommands()...)
	root.AddCommand(checkCommand(), formatCommand(), generationCommand(), corporaCommand())
	return root
}

func textCommand(name, description string, split func(string) []string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: description,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			input, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("read standard input: %w", err)
			}
			if !utf8.Valid(input) {
				return fmt.Errorf("standard input must contain valid UTF-8")
			}
			if err := json.NewEncoder(cmd.OutOrStdout()).Encode(split(string(input))); err != nil {
				return fmt.Errorf("write JSON output: %w", err)
			}
			return nil
		},
	}
}
