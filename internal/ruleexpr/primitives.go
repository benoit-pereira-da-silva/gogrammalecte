package ruleexpr

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

func integerValue(value any) (int, bool) {
	switch n := value.(type) {
	case int:
		return n, true
	case bool:
		if n {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}
func pythonEqual(a, b any) bool {
	if x, ok := integerValue(a); ok {
		if y, ok := integerValue(b); ok {
			return x == y
		}
	}
	if x, ok := a.([]any); ok {
		if y, ok := b.([]any); ok {
			if len(x) != len(y) {
				return false
			}
			for i := range x {
				if !pythonEqual(plain(x[i]), plain(y[i])) {
					return false
				}
			}
			return true
		}
	}
	return reflect.DeepEqual(a, b)
}
func pythonString(value any) (string, error) {
	switch v := value.(type) {
	case nil:
		return "None", nil
	case bool:
		if v {
			return "True", nil
		}
		return "False", nil
	case string:
		return v, nil
	case int:
		return strconv.Itoa(v), nil
	}
	return "", fmt.Errorf("str does not support values of type %T", value)
}
func pythonSpace(r rune) bool { return unicode.IsSpace(r) || r >= 0x1c && r <= 0x1f }
func decimalValue(r rune) (int, bool) {
	if r >= '0' && r <= '9' {
		return int(r - '0'), true
	}
	for _, entry := range unicode.Nd.R16 {
		if r >= rune(entry.Lo) && r <= rune(entry.Hi) && (r-rune(entry.Lo))%rune(entry.Stride) == 0 {
			return int((r-rune(entry.Lo))/rune(entry.Stride)) % 10, true
		}
	}
	for _, entry := range unicode.Nd.R32 {
		if r >= rune(entry.Lo) && r <= rune(entry.Hi) && (r-rune(entry.Lo))%rune(entry.Stride) == 0 {
			return int((r-rune(entry.Lo))/rune(entry.Stride)) % 10, true
		}
	}
	return 0, false
}
func pythonInteger(input string) (int, error) {
	// int accepts Unicode whitespace but excludes the four ASCII information
	// separators that Python's str.strip additionally considers whitespace.
	r := []rune(strings.TrimFunc(input, unicode.IsSpace))
	var normalized strings.Builder
	start := 0
	if len(r) > 0 && (r[0] == '+' || r[0] == '-') {
		normalized.WriteRune(r[0])
		start = 1
	}
	if start == len(r) {
		return 0, fmt.Errorf("invalid decimal integer %q", input)
	}
	previousDigit := false
	for i := start; i < len(r); i++ {
		if r[i] == '_' {
			if !previousDigit || i+1 == len(r) {
				return 0, fmt.Errorf("invalid decimal integer %q", input)
			}
			previousDigit = false
			continue
		}
		value, ok := decimalValue(r[i])
		if !ok {
			return 0, fmt.Errorf("invalid decimal integer %q", input)
		}
		normalized.WriteByte(byte('0' + value))
		previousDigit = true
	}
	return strconv.Atoi(normalized.String())
}
