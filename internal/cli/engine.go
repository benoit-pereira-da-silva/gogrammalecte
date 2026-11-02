package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/benoit-pereira-da-silva/gogrammalecte/french"
	"github.com/benoit-pereira-da-silva/gogrammalecte/grammar"
	"github.com/spf13/cobra"
)

func readUTF8(cmd *cobra.Command) (string, error) {
	data, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return "", fmt.Errorf("read standard input: %w", err)
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("standard input must contain valid UTF-8")
	}
	return string(data), nil
}
func parseOptionFlags(enabled, disabled []string) map[string]bool {
	options := map[string]bool{}
	for _, name := range enabled {
		options[name] = true
	}
	for _, name := range disabled {
		options[name] = false
	}
	return options
}
func checkCommand() *cobra.Command {
	var country, app string
	var withContext, full bool
	var enable, disable, ignore []string
	cmd := &cobra.Command{Use: "check", Short: "Check every paragraph of UTF-8 standard input and emit diagnostic JSON", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		input, err := readUTF8(cmd)
		if err != nil {
			return err
		}
		engine, err := grammar.Default()
		if err != nil {
			return err
		}
		ignored := map[string]bool{}
		for _, rule := range ignore {
			ignored[rule] = true
		}
		settings := grammar.ParseOptions{Country: country, AppContext: app, Context: withContext, Options: parseOptionFlags(enable, disable), IgnoredRules: ignored}
		result, err := checkDocument(cmd.Context(), engine, input, settings, full)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}}
	cmd.Flags().StringVar(&country, "country", "FR", "Country code used by grammar rules")
	cmd.Flags().StringVar(&app, "context", "Python", "Option defaults: Python, Server or Writer")
	cmd.Flags().BoolVar(&withContext, "include-context", false, "Include the original text around each diagnostic")
	cmd.Flags().BoolVar(&full, "full", false, "Include sentence analysis and labeled tokens")
	cmd.Flags().StringSliceVar(&enable, "enable", nil, "Comma-separated grammar options to enable")
	cmd.Flags().StringSliceVar(&disable, "disable", nil, "Comma-separated grammar options to disable")
	cmd.Flags().StringSliceVar(&ignore, "ignore", nil, "Comma-separated rule identifiers to ignore")
	return cmd
}
func formatCommand() *cobra.Command {
	var enable, disable []string
	var rule string
	cmd := &cobra.Command{Use: "format", Short: "Format French typography in UTF-8 standard input", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		input, err := readUTF8(cmd)
		if err != nil {
			return err
		}
		formatter, err := french.NewTextFormatter()
		if err != nil {
			return err
		}
		var output string
		if rule != "" {
			if len(enable)+len(disable) > 0 {
				return fmt.Errorf("--rule cannot be combined with --enable or --disable")
			}
			output, _, err = formatter.Apply(cmd.Context(), input, rule)
		} else {
			output, err = formatter.FormatWithOptions(cmd.Context(), input, parseOptionFlags(enable, disable))
		}
		if err != nil {
			return err
		}
		_, err = io.Copy(cmd.OutOrStdout(), strings.NewReader(output))
		return err
	}}
	cmd.Flags().StringVar(&rule, "rule", "", "Apply one named formatter option")
	cmd.Flags().StringSliceVar(&enable, "enable", nil, "Comma-separated formatter options to enable")
	cmd.Flags().StringSliceVar(&disable, "disable", nil, "Comma-separated formatter options to disable")
	return cmd
}

// checkDocument keeps the grammar engine's per-paragraph state isolated while
// reporting positions in the original input, including CRLF's two code points.
func checkDocument(ctx context.Context, engine *grammar.Engine, input string, options grammar.ParseOptions, full bool) (any, error) {
	source := []rune(input)
	paragraphs := [][2]int{}
	start := 0
	for i := 0; i < len(source); i++ {
		if source[i] != '\n' && source[i] != '\r' {
			continue
		}
		paragraphs = append(paragraphs, [2]int{start, i})
		if source[i] == '\r' && i+1 < len(source) && source[i+1] == '\n' {
			i++
		}
		start = i + 1
	}
	paragraphs = append(paragraphs, [2]int{start, len(source)})
	diagnostics := []grammar.Error{}
	analysis := &grammar.Analysis{ParagraphErrors: []grammar.Error{}, Sentences: []grammar.SentenceAnalysis{}}
	for _, span := range paragraphs {
		paragraph := string(source[span[0]:span[1]])
		if full {
			part, err := engine.Analyze(ctx, paragraph, options)
			if err != nil {
				return nil, err
			}
			globalizeDiagnostics(part.ParagraphErrors, span[0], source, options.Context)
			analysis.ParagraphErrors = append(analysis.ParagraphErrors, part.ParagraphErrors...)
			for _, sentence := range part.Sentences {
				sentence.Start += span[0]
				sentence.End += span[0]
				globalizeDiagnostics(sentence.Errors, span[0], source, options.Context)
				analysis.Sentences = append(analysis.Sentences, sentence)
			}
		} else {
			part, err := engine.Check(ctx, paragraph, options)
			if err != nil {
				return nil, err
			}
			globalizeDiagnostics(part, span[0], source, options.Context)
			diagnostics = append(diagnostics, part...)
		}
	}
	if full {
		return analysis, nil
	}
	return diagnostics, nil
}
func globalizeDiagnostics(diagnostics []grammar.Error, offset int, source []rune, withContext bool) {
	for i := range diagnostics {
		diagnostic := &diagnostics[i]
		diagnostic.Start += offset
		diagnostic.End += offset
		if withContext {
			start := max(0, min(diagnostic.Start, len(source)))
			end := max(start, min(diagnostic.End, len(source)))
			diagnostic.Underlined = string(source[start:end])
			diagnostic.Before = string(source[max(0, start-80):start])
			diagnostic.After = string(source[end:min(len(source), end+80)])
		}
	}
}
