// Portions ported from Grammalecte's graphspell/str_transform.py by Olivier R.
// Distributed under GNU GPL version 3; see the repository LICENSE.

// Package strtransform implements dictionary word transformations using Unicode
// code points. Text is expected to be well-formed UTF-8 and is not normalized.
package strtransform

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidCode identifies a malformed transformation code. It represents the
// IndexError or ValueError raised by the corresponding Python decoder.
var ErrInvalidCode = errors.New("invalid transformation code")

// DefineSuffixCode encodes how to derive stem from flexion. The first character
// encodes the number of trailing code points to remove, offset from '0'; the
// remaining characters are appended. Counts greater than nine are not decimal
// strings: ':' encodes ten, for example.
func DefineSuffixCode(flexion, stem string) string {
	if flexion == stem {
		return "0"
	}
	word, base := []rune(flexion), []rune(stem)
	shared := 0
	for shared < len(word) && shared < len(base) && word[shared] == base[shared] {
		shared++
	}
	return string(rune(len(word)-shared)+'0') + string(base[shared:])
}

// ChangeWordWithSuffixCode applies a suffix code to word. Removal counts use
// Python's clamped slice semantics, including codes below '0'. An empty code
// returns ErrInvalidCode.
func ChangeWordWithSuffixCode(word, code string) (string, error) {
	if code == "0" {
		return word, nil
	}
	transform := []rune(code)
	if len(transform) == 0 {
		return "", fmt.Errorf("%w: empty suffix", ErrInvalidCode)
	}
	if transform[0] == '0' {
		return word + string(transform[1:]), nil
	}
	text := []rune(word)
	end := sliceIndex(-int(transform[0]-'0'), len(text))
	return string(text[:end]) + string(transform[1:]), nil
}

// DefineAffixCode encodes a prefix and suffix transformation. It returns "0"
// for identical words, a literal stem when no reusable substring is found, or
// two transformation codes separated by '/'. Longest-substring ties follow the
// Python implementation's order, selecting the first ending position in flexion.
func DefineAffixCode(flexion, stem string) string {
	if flexion == stem {
		return "0"
	}
	word, base := []rune(flexion), []rune(stem)
	if start := indexRunes(word, base); start >= 0 {
		return string(rune(start)+'0') + "/" + string(rune(len(word)-len(base)-start)+'0')
	}
	common := longestCommonSubstring(word, base)
	if len(common) > 1 {
		baseStart := indexRunes(base, common)
		wordStart := indexRunes(word, common)
		return string(rune(wordStart)+'0') + string(base[:baseStart]) + "/" +
			string(rune(len(word)-len(common)-wordStart)+'0') + string(base[baseStart+len(common):])
	}
	return stem
}

// ChangeWordWithAffixCode applies an affix code to word. Codes without '/' are
// literal replacements, except "0", which leaves word unchanged. A separated
// code must contain exactly one slash and a nonempty code on either side.
func ChangeWordWithAffixCode(word, code string) (string, error) {
	if code == "0" {
		return word, nil
	}
	if !strings.ContainsRune(code, '/') {
		return code, nil
	}
	parts := strings.Split(code, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("%w: expected nonempty prefix/suffix", ErrInvalidCode)
	}
	prefix, text := []rune(parts[0]), []rune(word)
	start := sliceIndex(int(prefix[0]-'0'), len(text))
	withPrefix := string(prefix[1:]) + string(text[start:])
	return ChangeWordWithSuffixCode(withPrefix, parts[1])
}

// sliceIndex converts a Python slice boundary to a clamped nonnegative index.
func sliceIndex(index, length int) int {
	if index < 0 {
		index += length
	}
	if index < 0 {
		return 0
	}
	if index > length {
		return length
	}
	return index
}

func indexRunes(text, part []rune) int {
	for start := 0; start <= len(text)-len(part); start++ {
		matched := true
		for offset, character := range part {
			if text[start+offset] != character {
				matched = false
				break
			}
		}
		if matched {
			return start
		}
	}
	return -1
}

func longestCommonSubstring(first, second []rune) []rune {
	// A single row preserves the upstream traversal and tie-breaking while
	// avoiding its full quadratic matrix allocation.
	row := make([]int, len(second)+1)
	longest, end := 0, 0
	for x, left := range first {
		diagonal := 0
		for y, right := range second {
			previous := row[y+1]
			if left == right {
				row[y+1] = diagonal + 1
				if row[y+1] > longest {
					longest, end = row[y+1], x+1
				}
			} else {
				row[y+1] = 0
			}
			diagonal = previous
		}
	}
	return first[end-longest : end]
}
