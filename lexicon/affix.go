// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// Ported from Grammalecte genfrdic.py by Olivier R.

package lexicon

import (
	"fmt"
	"regexp"
	"strings"
)

// Flag groups ordered prefix or suffix rules with a shared Hunspell flag.
type Flag struct {
	Name        string
	Suffix, Mix bool
	Rules       []*AffixRule
}

// AffixRule is a parsed source rule and its compiled standard-library condition.
type AffixRule struct {
	Name                                string
	Suffix                              bool
	Cut, Add, Flags, Condition, Comment string
	Fields                              map[string]string
	pattern                             *regexp.Regexp
}

// ParseAffixRule rejects malformed rules and unsupported regular expressions.
func ParseAffixRule(line string) (*AffixRule, error) {
	r := &AffixRule{Fields: map[string]string{"di": "*"}}
	line = strings.TrimRight(line, " \n")
	if before, comment, ok := strings.Cut(line, "#"); ok {
		line = before
		r.Comment = strings.TrimSpace(comment)
	}
	parts := strings.Fields(line)
	if len(parts) < 5 {
		return nil, fmt.Errorf("affix rule requires at least five fields: %q", line)
	}
	r.Suffix = parts[0] == "SFX"
	r.Name = parts[1]
	r.Cut = parts[2]
	r.Add, r.Flags, _ = strings.Cut(parts[3], "/")
	if strings.Contains(r.Flags, "/") {
		return nil, fmt.Errorf("affix rule has multiple continuation separators: %q", line)
	}
	if r.Add == "0" {
		r.Add = ""
	}
	r.Condition = parts[4]
	pattern := "^" + r.Condition
	if r.Suffix {
		pattern = r.Condition + "$"
	}
	var err error
	r.pattern, err = regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("affix %s condition %q: %w", r.Name, r.Condition, err)
	}
	for _, field := range parts[5:] {
		if len(field) < 4 || field[2] != ':' {
			return nil, fmt.Errorf("invalid affix attribute %q", field)
		}
		name, value, _ := strings.Cut(field, ":")
		switch name {
		case "po", "is", "ds", "ts", "ip", "dp", "tp", "sp", "pa", "lx":
			if r.Fields[name] != "" {
				r.Fields[name] += " "
			}
			r.Fields[name] += value
		case "ph":
			// Preserve the reference's historical use of pa instead of ph here.
			if r.Fields["pa"] == "" {
				r.Fields[name] = value
			} else {
				r.Fields[name] = r.Fields["pa"] + " " + value
			}
		case "di":
			r.Fields[name] = value
		}
	}
	return r, nil
}

// Matches reports whether the rule's anchored condition matches the lemma.
func (r *AffixRule) Matches(lemma string) bool { return r.pattern.MatchString(lemma) }

// IsReplication identifies the source's virtual-lemma replication rules.
func (r *AffixRule) IsReplication() bool {
	return r.Flags == "" && ((r.Cut == "0" && r.Add == "") || r.Cut == r.Add)
}

// LexicalMorphology returns the expanded fields used by the inflector.
func (r *AffixRule) LexicalMorphology() string { return lexicalMorphology(r.Fields) }

// Morphology formats the rule's Hunspell morphology and optional metadata.
func (r *AffixRule) Morphology(mode int) string {
	var result strings.Builder
	for _, name := range []string{"po", "is", "ds", "ts", "ip", "dp", "tp", "sp", "pa", "ph"} {
		if value := r.Fields[name]; value != "" {
			result.WriteString(FieldToHunspell(name, value))
		}
	}
	if mode > 1 {
		if value := r.Fields["lx"]; value != "" {
			result.WriteString(FieldToHunspell("lx", value))
		}
		if r.Fields["di"] != "*" {
			result.WriteString(" di:" + r.Fields["di"])
		}
	}
	return result.String()
}

// HunspellLine formats a rule without optional alias compression.
func (r *AffixRule) HunspellLine(mode int, simplified bool) string {
	kind := "PFX"
	if r.Suffix {
		kind = "SFX"
	}
	addition := r.Add
	if addition == "" {
		addition = "0"
	}
	line := kind + " " + r.Name + " " + r.Cut + " " + addition
	if r.Flags != "" {
		line += "/" + r.Flags
		if simplified {
			line = strings.ReplaceAll(line, "()", "")
		}
	}
	line += " " + r.Condition
	if !simplified && mode > 0 {
		line += r.Morphology(mode)
	}
	return line + "\n"
}
