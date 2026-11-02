package french

// The ordered formatter options and transformations are ported from
// Grammalecte's textformatter.py by Olivier R. (GPLv3).

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/benoit-pereira-da-silva/gogrammalecte/internal/pyregex"
)

type formatterOption struct {
	Name    string
	Enabled bool
	Rules   [][2]string
}
type formatterRule struct {
	simple      *regexp.Regexp
	complex     *pyregex.Regexp
	replacement string
}
type formatterTable struct {
	options []formatterOption
	rules   map[string][]formatterRule
}

// TextFormatter is immutable and safe for concurrent use. Tables and patterns
// are loaded once from embedded resources, preserving the Python option order.
type TextFormatter struct{ table *formatterTable }

var defaultFormatter formatterTable
var formatterError error
var loadFormatter = sync.OnceFunc(func() {
	loadResource("textformatter", &defaultFormatter.options)
	defaultFormatter.rules = make(map[string][]formatterRule)
	for _, option := range defaultFormatter.options {
		for _, pair := range option.Rules {
			rule := formatterRule{replacement: pair[1]}
			var err error
			pattern := pair[0]
			if formatterNeedsPythonRegex(pattern) {
				rule.complex, err = pyregex.Compile(pattern)
			} else {
				rule.simple, err = regexp.Compile(pattern)
			}
			if err != nil {
				formatterError = fmt.Errorf("formatter option %s pattern %q: %w", option.Name, pair[0], err)
				return
			}
			defaultFormatter.rules[option.Name] = append(defaultFormatter.rules[option.Name], rule)
		}
	}
})

func formatterNeedsPythonRegex(pattern string) bool {
	if strings.Contains(pattern, "(?") || strings.Contains(pattern, "$") {
		return true
	}
	for _, class := range []string{`\w`, `\W`, `\s`, `\S`, `\d`, `\D`, `\b`, `\B`} {
		if strings.Contains(pattern, class) {
			return true
		}
	}
	return false
}

func NewTextFormatter() (*TextFormatter, error) {
	loadFormatter()
	if formatterError != nil {
		return nil, formatterError
	}
	return &TextFormatter{table: &defaultFormatter}, nil
}

func (f *TextFormatter) DefaultOptions() map[string]bool {
	out := make(map[string]bool, len(f.table.options))
	for _, option := range f.table.options {
		out[option.Name] = option.Enabled
	}
	return out
}

// FormatText applies the reference's default transformations in their original order.
func (f *TextFormatter) FormatText(ctx context.Context, input string) (string, error) {
	return f.FormatWithOptions(ctx, input, nil)
}

// FormatWithOptions applies overrides to the default configuration. Unknown
// option names are errors; omitted options retain their reference defaults.
func (f *TextFormatter) FormatWithOptions(ctx context.Context, input string, overrides map[string]bool) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	for name := range overrides {
		if _, ok := f.table.rules[name]; !ok {
			return "", fmt.Errorf("unknown formatter option %q", name)
		}
	}
	for _, option := range f.table.options {
		enabled := option.Enabled
		if override, ok := overrides[option.Name]; ok {
			enabled = override
		}
		if enabled {
			var err error
			input, _, err = f.Apply(ctx, input, option.Name)
			if err != nil {
				return "", err
			}
		}
	}
	return input, nil
}

// Apply runs one named transformation group and reports the replacement count.
func (f *TextFormatter) Apply(ctx context.Context, input, name string) (string, int, error) {
	rules, ok := f.table.rules[name]
	if !ok {
		return "", 0, fmt.Errorf("unknown formatter option %q", name)
	}
	count := 0
	for index := range rules {
		var n int
		var err error
		input, n, err = f.applyRule(ctx, input, name, index)
		if err != nil {
			return "", 0, err
		}
		count += n
	}
	return input, count, nil
}

func formatterReplacement(replacement string, group func(int) string) (string, error) {
	var out strings.Builder
	for i := 0; i < len(replacement); i++ {
		if replacement[i] != '\\' {
			out.WriteByte(replacement[i])
			continue
		}
		i++
		if i == len(replacement) {
			return "", fmt.Errorf("trailing backslash in formatter replacement")
		}
		if replacement[i] >= '1' && replacement[i] <= '9' {
			start := i
			for i+1 < len(replacement) && replacement[i+1] >= '0' && replacement[i+1] <= '9' {
				i++
			}
			index, _ := strconv.Atoi(replacement[start : i+1])
			out.WriteString(group(index))
			continue
		}
		switch replacement[i] {
		case '\\':
			out.WriteByte('\\')
		case 'n':
			out.WriteByte('\n')
		case 't':
			out.WriteByte('\t')
		case 'r':
			out.WriteByte('\r')
		default:
			return "", fmt.Errorf("unsupported formatter replacement escape \\%c", replacement[i])
		}
	}
	return out.String(), nil
}
func (f *TextFormatter) applyRule(ctx context.Context, input, name string, index int) (string, int, error) {
	if err := ctx.Err(); err != nil {
		return "", 0, err
	}
	rules, ok := f.table.rules[name]
	if !ok || index < 0 || index >= len(rules) {
		return "", 0, fmt.Errorf("unknown formatter rule %s[%d]", name, index)
	}
	rule := rules[index]
	var out strings.Builder
	if rule.simple != nil {
		matches := rule.simple.FindAllStringSubmatchIndex(input, -1)
		position := 0
		for _, match := range matches {
			out.WriteString(input[position:match[0]])
			replacement, err := formatterReplacement(rule.replacement, func(n int) string {
				if n*2+1 >= len(match) || match[n*2] < 0 {
					return ""
				}
				return input[match[n*2]:match[n*2+1]]
			})
			if err != nil {
				return "", 0, err
			}
			out.WriteString(replacement)
			position = match[1]
		}
		out.WriteString(input[position:])
		return out.String(), len(matches), nil
	}
	matches, err := rule.complex.FindAll(ctx, input, -1)
	if err != nil {
		return "", 0, err
	}
	r := []rune(input)
	position := 0
	for _, match := range matches {
		out.WriteString(string(r[position:match.Start(0)]))
		replacement, err := formatterReplacement(rule.replacement, match.Group)
		if err != nil {
			return "", 0, err
		}
		out.WriteString(replacement)
		position = match.End(0)
	}
	out.WriteString(string(r[position:]))
	return out.String(), len(matches), nil
}
