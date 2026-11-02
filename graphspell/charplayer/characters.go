// Package charplayer contains the pinned character and keyboard tables used by
// Grammalecte's spelling suggestion algorithms. Adapted from Olivier R.'s
// char_player.py and keyboard_chars_proximity.py under GNU GPL version 3.
package charplayer

import (
	_ "embed"
	"encoding/json"
	"slices"
	"strings"
)

//go:embed data/characters.json
var encodedTables []byte

var tables = loadTables()

type characterTables struct {
	Distance                                 map[string]map[string]float64
	Vowels, Consonants, Double               string
	OneToOne                                 map[string]string
	OneToMany, TwoToMany, FinalOne, FinalTwo map[string][]string
	Keyboards                                map[string]map[string]string
	KeyboardNames                            []string
}

func loadTables() characterTables {
	var t characterTables
	if err := json.Unmarshal(encodedTables, &t); err != nil {
		panic(err)
	}
	return t
}

// DistanceBetweenChars returns the asymmetric upstream substitution cost.
func DistanceBetweenChars(first, second rune) float64 {
	if first == second {
		return 0
	}
	if v, ok := KnownDistance(first, second); ok {
		return v
	}
	return 1
}

// KnownDistance distinguishes a listed pair from an unlisted substitution.
func KnownDistance(first, second rune) (float64, bool) {
	v, ok := tables.Distance[string(first)][string(second)]
	return v, ok
}

// IsVowel reports membership in the language-specific vowel table.
func IsVowel(r rune) bool { return strings.ContainsRune(tables.Vowels, r) }

// IsConsonant reports membership in the language-specific consonant table.
func IsConsonant(r rune) bool { return strings.ContainsRune(tables.Consonants, r) }

// CanDouble reports whether a letter may occur twice in a row.
func CanDouble(r rune) bool { return strings.ContainsRune(tables.Double, r) }

// OneToOne returns the ordered string of similar characters.
func OneToOne(character string) string { return tables.OneToOne[character] }

// OneToMany returns ordered replacements, suppressing consonant expansion next
// to another consonant. Empty strings represent missing adjacent characters.
func OneToMany(previous, current, next string) []string {
	if singleConsonant(current) && (singleConsonant(previous) || singleConsonant(next)) {
		return []string{}
	}
	return copyList(tables.OneToMany[current])
}

// TwoToMany returns replacements for a two-character sequence.
func TwoToMany(sequence string) []string { return copyList(tables.TwoToMany[sequence]) }

// FinalOne returns ordered replacements at the end of a word.
func FinalOne(character string) []string { return copyList(tables.FinalOne[character]) }

// FinalTwo returns ordered replacements for the last two characters.
func FinalTwo(sequence string) []string { return copyList(tables.FinalTwo[sequence]) }
func copyList(input []string) []string {
	if input == nil {
		return []string{}
	}
	return slices.Clone(input)
}

// KeyboardNames returns supported layouts in the original order.
func KeyboardNames() []string { return slices.Clone(tables.KeyboardNames) }

// KeyboardMap returns an independent copy of a keyboard proximity map.
func KeyboardMap(layout string) map[string]string {
	r := map[string]string{}
	for k, v := range tables.Keyboards[strings.ToLower(layout)] {
		r[k] = v
	}
	return r
}

func singleConsonant(s string) bool { r := []rune(s); return len(r) == 1 && IsConsonant(r[0]) }
