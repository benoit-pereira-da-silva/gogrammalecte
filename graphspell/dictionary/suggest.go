// Spelling suggestions are ported from Olivier R.'s graphspell/ibdawg.py.
package dictionary

import (
	"context"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/charplayer"
	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/strtransform"
)

type suggestionResult struct {
	word, simplified                  string
	distanceLimit, minDistance, limit int
	all                               map[string]bool
	accepted                          map[string]float64
}

func newSuggestionResult(word string, limit int) *suggestionResult {
	return &suggestionResult{word: word, simplified: strtransform.SimplifyWord(word), distanceLimit: utf8.RuneCountInString(word)/3 + 1, minDistance: 1000, limit: limit, all: map[string]bool{}, accepted: map[string]float64{}}
}
func (r *suggestionResult) add(word string) {
	if r.all[word] {
		return
	}
	r.all[word] = true
	simplified := strtransform.SimplifyWord(word)
	distance := strtransform.DistanceSift4(r.simplified, simplified)
	r.minDistance = min(r.minDistance, distance)
	if distance <= r.minDistance+1 {
		score := math.Min(strtransform.DistanceDamerauLevenshteinX(r.word, word), strtransform.DistanceDamerauLevenshteinX(r.simplified, simplified))
		if strings.Contains(word, " ") {
			score++
		}
		r.accepted[word] = score
		if len(r.accepted) > r.limit*6 {
			r.distanceLimit = -1
		}
	}
	r.distanceLimit = min(r.distanceLimit, r.minDistance+1)
}
func (r *suggestionResult) results() []string {
	words := make([]string, 0, len(r.accepted))
	for word := range r.accepted {
		words = append(words, word)
	}
	sort.Slice(words, func(i, j int) bool {
		a, b := r.accepted[words[i]], r.accepted[words[j]]
		if a == b {
			return words[i] < words[j]
		}
		return a < b
	})
	if len(words) > r.limit {
		words = words[:r.limit]
	}
	out := []string{}
	seen := map[string]bool{}
	upper := strtransform.IsUpper(r.word)
	firstUpper := strtransform.IsUpper(first(r.word))
	for _, word := range words {
		if upper {
			word = strtransform.Upper(word)
		} else if firstUpper {
			word = strtransform.Upper(first(word)) + dropFirst(word)
		}
		if !seen[word] {
			seen[word] = true
			out = append(out, word)
		}
	}
	return out
}

var elidedPrefix = regexp.MustCompile(`(?i)^([ldmtsnjcç]|lorsqu|presqu|jusqu|puisqu|quoiqu|quelqu|qu)[’'‘` + "`" + `ʼ]([\pL\pN_-]+)`)
var compoundSuffix = regexp.MustCompile(`(?i)^([\pL\pN_]+)(-(?:(?:les?|la)-(?:mo[iİı]|to[iİı]|lu[iİı]|[nv]ous|leur)|t-(?:[iİı]l|elle|on)|y|en|[mts]’(?:y|en)|les?|l[aà]|[mt]o[iİı]|leur|lu[iİı]|je|tu|[iİı]ls?|elles?|on|[nv]ous|ce))(?:\n)?$`)

func splitFrenchWord(word string) (string, string, string) {
	prefix, suffix := "", ""
	if match := elidedPrefix.FindStringSubmatch(word); match != nil {
		prefix = match[1] + "’"
		word = match[2]
	}
	if match := compoundSuffix.FindStringSubmatch(word); match != nil {
		word = match[1]
		suffix = match[2]
	}
	return prefix, word, suffix
}

// Suggest returns the ranked suggestions of the Python dictionary algorithm.
// An optional true flag enables splitting trailing ASCII numbers into exponents.
// Input must be valid UTF-8. A nonpositive limit returns an empty result.
func (d *Dictionary) Suggest(word string, limit int, splitTrailingNumbers ...bool) []string {
	split := len(splitTrailingNumbers) > 0 && splitTrailingNumbers[0]
	suggestions, _ := d.SuggestContext(context.Background(), word, limit, split)
	return suggestions
}

// SuggestContext supports cancellation without returning partial results as a
// successful analysis. It uses no shared mutable scratch buffers.
func (d *Dictionary) SuggestContext(ctx context.Context, word string, limit int, splitTrailingNumbers bool) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if limit <= 0 || !utf8.ValidString(word) {
		return []string{}, nil
	}
	word = strtransform.SpellingNormalization(strings.TrimRight(word, "."))
	prefix, suffix := "", ""
	if d.data.LanguageCode == "fr" {
		prefix, word, suffix = splitFrenchWord(word)
	}
	length := utf8.RuneCountInString(word)
	result := newSuggestionResult(word, limit)
	word = strtransform.CleanWord(word)
	if splitTrailingNumbers {
		splitNumbers(result, word)
	}
	for _, separator := range []string{"'", "’"} {
		left, right, found := strings.Cut(word, separator)
		if found && d.IsValid(left) && d.IsValid(right) {
			result.add(left + " " + right)
		}
	}
	search := suggestionSearch{dictionary: d, result: result, ctx: ctx}
	search.walk(word, 0, 0, 0, 0, 0, 0, "", false)
	search.walk(word, max(length/3, 1), length/5, max((length-5)/4, 1), max(length/4, 1), 0, 0, "", false)
	if search.err != nil {
		return nil, search.err
	}
	out := result.results()
	if prefix != "" || suffix != "" {
		for i := range out {
			out[i] = prefix + out[i] + suffix
		}
	}
	return out, nil
}
func splitNumbers(result *suggestionResult, word string) {
	runes := []rune(word)
	i := len(runes)
	for i > 0 && runes[i-1] >= '0' && runes[i-1] <= '9' {
		i--
	}
	if i == 0 || i == len(runes) || !strtransform.IsAlpha(string(runes[i-1])) {
		return
	}
	for _, r := range runes[:i] {
		if unicode.Is(unicode.Nd, r) {
			return
		}
	}
	result.add(string(runes[:i]) + "\u00a0" + strtransform.NumbersToExponent(string(runes[i:])))
}

type suggestionSearch struct {
	dictionary *Dictionary
	result     *suggestionResult
	ctx        context.Context
	err        error
}

func (s *suggestionSearch) walk(remain string, switches, deletions, replacements, jumps, distance, address int, word string, avoidLoop bool) {
	if s.err != nil {
		return
	}
	if err := s.ctx.Err(); err != nil {
		s.err = err
		return
	}
	d, r := s.dictionary, s.result
	if d.data.Graph[address]&d.finalMask != 0 {
		if remain == "" {
			r.add(word)
			for _, tail := range d.tails(address, "", 2) {
				r.add(word + tail)
			}
			return
		}
		if utf8.RuneCountInString(word)+utf8.RuneCountInString(remain) == utf8.RuneCountInString(r.word) && strings.HasPrefix(strtransform.Lower(r.word), strtransform.Lower(word)) && d.IsValid(remain) {
			if d.data.LanguageCode == "fr" && elisionWords[strtransform.Lower(word)] && charplayer.IsVowel([]rune(remain)[0]) {
				r.add(word + "’" + remain)
			}
			if utf8.RuneCountInString(word) > 1 && utf8.RuneCountInString(remain) > 1 || strings.Contains("aày", word) || strings.Contains("aày", remain) {
				r.add(word + " " + remain)
			}
		}
	}
	if distance > r.distanceLimit {
		return
	}
	current, rest := first(remain), dropFirst(remain)
	for _, arc := range d.characterArcs(address, current) {
		character := d.data.Values[arc.value]
		if strings.Contains(equivalents(current), character) {
			s.walk(rest, switches, deletions, replacements, jumps, distance, arc.address, word+character, false)
		} else if !avoidLoop {
			if replacements > 0 && d.IsNgramsOK(character+first(rest)) {
				s.walk(rest, switches, deletions, replacements-1, jumps, distance+1, arc.address, word+character, true)
			}
			if jumps > 0 {
				s.walk(remain, switches, deletions, replacements, jumps-1, distance+1, arc.address, word+character, true)
			}
		}
	}
	if avoidLoop {
		return
	}
	count := utf8.RuneCountInString(remain)
	if count > 1 {
		second := first(rest)
		if current == second {
			s.walk(rest, switches, deletions, replacements, jumps, distance, address, word, false)
		} else {
			if switches > 0 && d.IsNgramsOK(last(word)+second) && d.IsNgramsOK(second+current) {
				s.walk(second+current+dropFirst(rest), switches-1, deletions, replacements, jumps, distance+1, address, word, true)
			}
			if deletions > 0 && d.IsNgramsOK(last(word)+second) {
				s.walk(rest, switches, deletions-1, replacements, jumps, distance+1, address, word, true)
			}
		}
		for _, replacement := range charplayer.OneToMany(last(word), current, second) {
			s.walk(replacement+rest, switches, deletions, replacements, jumps, distance, address, word, true)
		}
		for _, replacement := range charplayer.TwoToMany(current + second) {
			s.walk(replacement+dropFirst(rest), switches, deletions, replacements, jumps, distance, address, word, true)
		}
	}
	if count == 2 {
		for _, replacement := range charplayer.FinalTwo(remain) {
			s.walk(replacement, switches, deletions, replacements, jumps, distance, address, word, true)
		}
	} else if count == 1 {
		s.walk("", switches, deletions, replacements, jumps, distance, address, word, true)
		for _, replacement := range charplayer.FinalOne(remain) {
			s.walk(replacement, switches, deletions, replacements, jumps, distance, address, word, true)
		}
	}
}

var elisionWords = map[string]bool{"l": true, "d": true, "n": true, "m": true, "t": true, "s": true, "c": true, "j": true, "qu": true, "lorsqu": true, "puisqu": true, "quoiqu": true, "jusqu": true, "quelqu": true}

func equivalents(character string) string {
	if equivalent := charplayer.OneToOne(character); equivalent != "" {
		return equivalent
	}
	return character
}
func first(s string) string {
	if s == "" {
		return ""
	}
	_, size := utf8.DecodeRuneInString(s)
	return s[:size]
}
func dropFirst(s string) string {
	if s == "" {
		return ""
	}
	_, size := utf8.DecodeRuneInString(s)
	return s[size:]
}
func last(s string) string {
	if s == "" {
		return ""
	}
	_, size := utf8.DecodeLastRuneInString(s)
	return s[len(s)-size:]
}

type characterArc struct{ value, address int }

func (d *Dictionary) characterArcs(address int, current string) []characterArc {
	preferred, others := []characterArc{}, []characterArc{}
	similar := equivalents(current)
	for i := address; ; i += 2 {
		raw := d.data.Graph[i]
		value := int(raw & d.arcMask)
		if value > 0 && value <= d.data.Characters {
			arc := characterArc{value, int(d.data.Graph[i+1])}
			if strings.Contains(similar, d.data.Values[value]) {
				preferred = append(preferred, arc)
			} else {
				others = append(others, arc)
			}
		}
		if raw&d.lastMask != 0 {
			break
		}
	}
	return append(preferred, others...)
}
func (d *Dictionary) tails(address int, tail string, depth int) []string {
	found := map[string]bool{}
	for _, arc := range d.characterArcs(address, "") {
		next := tail + d.data.Values[arc.value]
		if d.data.Graph[arc.address]&d.finalMask != 0 {
			found[next] = true
		}
		if depth > 0 && len(found) == 0 {
			for _, value := range d.tails(arc.address, next, depth-1) {
				found[value] = true
			}
		}
	}
	result := make([]string, 0, len(found))
	for value := range found {
		result = append(result, value)
	}
	// The reference returns a hash set here. Sort it to make repeated Go runs
	// deterministic while preserving the final score/lexicographic ranking.
	sort.Strings(result)
	return result
}

// IsNgramsOK tests a two-code-point sequence against known dictionary bigrams.
// Other lengths, or dictionaries without a bigram table, are accepted.
func (d *Dictionary) IsNgramsOK(sequence string) bool {
	if utf8.RuneCountInString(sequence) != 2 || len(d.ngrams) == 0 {
		return true
	}
	return d.ngrams[sequence]
}

// SimilarEntries expands each spelling suggestion to its morphological entries.
func (d *Dictionary) SimilarEntries(word string, limit int) []Entry {
	out := []Entry{}
	if word == "" {
		return out
	}
	for _, suggestion := range d.Suggest(word, limit) {
		for _, morph := range d.Morph(suggestion) {
			stem, tag, found := strings.Cut(strings.TrimPrefix(morph, ">"), "/")
			if found {
				out = append(out, Entry{suggestion, stem, tag})
			}
		}
	}
	return out
}
