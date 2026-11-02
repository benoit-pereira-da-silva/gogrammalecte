package text

import (
	"fmt"
	"strings"
	"unicode"
)

// Wrap follows the reference paragraph wrapper. It preserves spaces and counts
// Unicode code points; only trailing CR/LF characters are removed.
func Wrap(input string, width int) ([]string, error) {
	if width <= 0 {
		return nil, fmt.Errorf("invalid wrapping width %d", width)
	}
	r := []rune(strings.TrimRight(input, "\r\n"))
	lines := []string{}
	for len(r) > width {
		end := 0
		for i := 0; i < width; i++ {
			if r[i] == ' ' {
				end = i + 1
			}
		}
		if end == 0 {
			end = width
		}
		lines = append(lines, string(r[:end]))
		r = r[end:]
	}
	return append(lines, string(r)), nil
}
func asciiWhitespace(r rune) bool    { return strings.ContainsRune(" \t\n\r\v\f", r) }
func pythonWhitespace(r rune) bool   { return unicode.IsSpace(r) || r >= 0x1c && r <= 0x1f }
func wordCharacter(r rune) bool      { return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' }
func wordNonDigit(r rune) bool       { return wordCharacter(r) && !unicode.IsDigit(r) }
func dashBoundaryBefore(r rune) bool { return wordCharacter(r) || strings.ContainsRune("!\"'&.,?", r) }
func blankChunk(s string) bool       { return strings.TrimFunc(s, pythonWhitespace) == "" }

// messageChunks implements the default textwrap word boundaries, including
// multi-hyphen separators and the two permitted intra-word hyphen boundaries.
// This avoids depending on a second regular-expression dialect for wrapping.
func messageChunks(input string) []string {
	var expanded strings.Builder
	column := 0
	for _, r := range input {
		if r == '\t' {
			n := 8 - column%8
			expanded.WriteString(strings.Repeat(" ", n))
			column += n
			continue
		}
		if r == '\r' || r == '\n' {
			column = 0
		} else {
			column++
		}
		if asciiWhitespace(r) {
			r = ' '
		}
		expanded.WriteRune(r)
	}
	r := []rune(expanded.String())
	chunks := []string{}
	emDashEnd := func(index int) int {
		if index == 0 || !dashBoundaryBefore(r[index-1]) || r[index] != '-' {
			return -1
		}
		end := index
		for end < len(r) && r[end] == '-' {
			end++
		}
		if end-index >= 2 && end < len(r) && wordCharacter(r[end]) {
			return end
		}
		return -1
	}
	hyphenBoundary := func(index int) bool {
		if r[index] != '-' {
			return false
		}
		before := index >= 2 && wordNonDigit(r[index-1]) && wordNonDigit(r[index-2]) || index >= 3 && wordNonDigit(r[index-1]) && r[index-2] == '-' && wordNonDigit(r[index-3])
		if !before || index+1 >= len(r) || !wordNonDigit(r[index+1]) {
			return false
		}
		end := index + 2
		if end < len(r) && r[end] == '-' {
			end++
		}
		return end < len(r) && wordNonDigit(r[end])
	}
	for start := 0; start < len(r); {
		end := start + 1
		if asciiWhitespace(r[start]) {
			for end < len(r) && asciiWhitespace(r[end]) {
				end++
			}
		} else if dashEnd := emDashEnd(start); dashEnd >= 0 {
			end = dashEnd
		} else {
			for end <= len(r) {
				if hyphenBoundary(end-1) || end == len(r) || asciiWhitespace(r[end]) || emDashEnd(end) >= 0 {
					break
				}
				end++
			}
		}
		chunks = append(chunks, string(r[start:end]))
		start = end
	}
	return chunks
}

// wrapMessage mirrors Python textwrap's defaults used by readable diagnostics.
// The final indent may exceed width; in that case each pass still consumes at
// least one character, preserving the reference behavior for very narrow output.
func wrapMessage(input string, width int, subsequentIndent string) ([]string, error) {
	if width <= 0 {
		return nil, fmt.Errorf("invalid wrapping width %d", width)
	}
	chunks := messageChunks(input)
	lines := []string{}
	position := 0
	for position < len(chunks) {
		indent := ""
		if len(lines) > 0 {
			indent = subsequentIndent
		}
		available := width - len([]rune(indent))
		if len(lines) > 0 && blankChunk(chunks[position]) {
			position++
		}
		current := []string{}
		length := 0
		for position < len(chunks) {
			n := len([]rune(chunks[position]))
			if length+n > available {
				break
			}
			current = append(current, chunks[position])
			length += n
			position++
		}
		if position < len(chunks) && len([]rune(chunks[position])) > available {
			space := available - length
			if available < 1 {
				space = 1
			}
			if space > 0 {
				r := []rune(chunks[position])
				end := space
				if end > len(r) {
					end = len(r)
				}
				if len(r) > space {
					hyphen := -1
					for i := 0; i < space; i++ {
						if r[i] == '-' {
							hyphen = i
						}
					}
					if hyphen > 0 {
						for _, c := range r[:hyphen] {
							if c != '-' {
								end = hyphen + 1
								break
							}
						}
					}
				}
				current = append(current, string(r[:end]))
				chunks[position] = string(r[end:])
			} else if len(current) == 0 {
				current = append(current, chunks[position])
				position++
			}
		}
		if len(current) > 0 && blankChunk(current[len(current)-1]) {
			current = current[:len(current)-1]
		}
		if len(current) > 0 {
			lines = append(lines, indent+strings.Join(current, ""))
		}
	}
	return lines, nil
}
