package rules

import (
	"fmt"
	"strings"
)

// TokenExpansion is one expanded token pattern, retaining the reference's
// nullable-position metadata. Parentheses still mark selected token groups.
type TokenExpansion struct {
	Tokens        []string `json:"tokens"`
	FirstNullable int      `json:"first_nullable"`
	LastNullable  int      `json:"last_nullable"`
}

// ExpandGraphTokens expands alternatives, definitions, suffix declensions,
// optional tokens and the visible space marker using upstream ordering.
func ExpandGraphTokens(source string, definitions, declensions []Definition) ([]TokenExpansion, error) {
	blocks := strings.Fields(source)
	if len(blocks) == 0 {
		return nil, fmt.Errorf("empty graph token pattern")
	}
	defs := map[string]string{}
	for _, d := range definitions {
		defs["{"+d.Name+"}"] = d.Value
	}
	lines := [][]string{}
	firstNullable, lastNullable := 0, 0
	for index, block := range blocks {
		position := index + 1
		block = strings.ReplaceAll(block, "␣", " ")
		optional := strings.HasPrefix(block, "?") && strings.HasSuffix(block, "¿")
		if optional {
			block = strings.TrimSuffix(strings.TrimPrefix(block, "?"), "¿")
			if firstNullable == 0 {
				firstNullable = position
			}
			lastNullable = position
		}
		selected := strings.HasPrefix(block, "(") && strings.HasSuffix(block, ")")
		if selected {
			firstNullable = -1
		}
		if strings.HasPrefix(block, "({") && strings.HasSuffix(block, "})") {
			if replacement, ok := defs[block[1:len(block)-1]]; ok {
				block = "(" + replacement + ")"
			}
		} else if replacement, ok := defs[block]; ok {
			block = replacement
		}
		multiple := strings.HasPrefix(block, "[") && strings.HasSuffix(block, "]") || strings.HasPrefix(block, "([") && strings.HasSuffix(block, "])")
		if multiple {
			selected = strings.HasPrefix(block, "(") && strings.HasSuffix(block, ")")
			if selected {
				block = block[1 : len(block)-1]
			}
			choices := []string{}
			for _, token := range strings.Split(block[1:len(block)-1], "|") {
				if strings.Contains(token, "+") && !strings.HasPrefix(token, "+") {
					matched := false
					for _, decl := range declensions {
						if strings.HasSuffix(token, decl.Name) {
							base := strings.TrimSuffix(token, decl.Name)
							choices = append(choices, base)
							for _, suffix := range strings.Fields(decl.Value) {
								choices = append(choices, base+suffix)
							}
							matched = true
							break
						}
					}
					if !matched {
						return nil, fmt.Errorf("unknown declension in token %q", token)
					}
				} else {
					choices = append(choices, token)
				}
			}
			if len(choices) == 0 {
				return nil, fmt.Errorf("empty token alternatives")
			}
			if len(lines) == 0 {
				for _, choice := range choices {
					if selected {
						choice = "(" + choice + ")"
					}
					lines = append(lines, []string{choice})
				}
				if optional {
					for i := 0; i < len(choices)+1; i++ {
						lines = append(lines, []string{})
					}
				}
			} else {
				extra := [][]string{}
				if optional {
					for _, line := range lines {
						for _, choice := range choices {
							copyLine := append([]string{}, line...)
							extra = append(extra, append(copyLine, choice))
						}
					}
				} else {
					for i, line := range lines {
						for _, choice := range choices[1:] {
							copyLine := append([]string{}, line...)
							if selected {
								choice = "(" + choice + ")"
							}
							extra = append(extra, append(copyLine, choice))
						}
						choice := choices[0]
						if selected {
							choice = "(" + choice + ")"
						}
						lines[i] = append(line, choice)
					}
				}
				lines = append(lines, extra...)
			}
		} else {
			if len(lines) == 0 {
				lines = append(lines, []string{block})
				if optional {
					lines = append(lines, []string{})
				}
			} else if optional {
				extra := make([][]string, 0, len(lines))
				for _, line := range lines {
					copyLine := append([]string{}, line...)
					extra = append(extra, append(copyLine, block))
				}
				lines = append(lines, extra...)
			} else {
				for i := range lines {
					lines[i] = append(lines[i], block)
				}
			}
		}
	}
	result := make([]TokenExpansion, 0, len(lines))
	for _, line := range lines {
		result = append(result, TokenExpansion{line, firstNullable, lastNullable - len(blocks) - 1})
	}
	return result, nil
}
