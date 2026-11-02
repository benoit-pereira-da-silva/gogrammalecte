// Package tokenizer ports Grammalecte's Python tokenization rules with Unicode
// code-point positions and NFC-normalized values. Inputs must be valid UTF-8.
// Based on graphspell/tokenizer.py by Olivier R., GNU GPL version 3.
package tokenizer

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Token retains the Python field meanings and JSON wire names. Start and End
// refer to the original input, before NFC normalization, in Unicode code points.
type Token struct {
	Index      int      `json:"i"`
	Type       string   `json:"sType"`
	Value      string   `json:"sValue"`
	Start      int      `json:"nStart"`
	End        int      `json:"nEnd"`
	Morphology []string `json:"lMorph,omitempty"`
}

type pattern struct {
	kind       string
	expression *regexp.Regexp
	boundary   bool
	fallback   *regexp.Regexp
}

// Tokenizer is immutable after construction and safe for concurrent calls.
type Tokenizer struct {
	language string
	patterns []pattern
}

var french = build("fr")
var generic = build("default")

// New selects French or default tokenization. Unknown languages use the default
// rules, correcting the Python constructor's inconsistent fallback lookup.
func New(language string) *Tokenizer {
	if language == "fr" {
		return french
	}
	return generic
}

// Tokenize uses French rules without synthetic start/end tokens.
func Tokenize(input string) []Token { return french.Tokens(input, false) }

// Language reports the selected rule set.
func (t *Tokenizer) Language() string { return t.language }

func build(language string) *Tokenizer {
	t := &Tokenizer{language: language}
	add := func(kind, expression string, boundary bool) {
		t.patterns = append(t.patterns, pattern{kind: kind, expression: compile(expression), boundary: boundary})
	}
	add("FOLDERUNIX", `/(?:bin|boot|dev|etc|home|lib|mnt|opt|root|sbin|tmp|usr|var|Bureau|Documents|Images|Musique|Public|Téléchargements|Vidéos)(?:/[\w.()-]+)*`, false)
	add("FOLDERWIN", `[a-zA-Z]:\\(?:Program Files(?: [(]x86[)]|)|[\w.()]+)(?:\\[\w.()-]+)*`, false)
	add("PUNC", `[\]\[,.;:!?…«»“”‘’"(){}·–—¿¡]`, false)
	add("WORD_ACRONYM", `[A-Z][.][A-Z][.](?:[A-Z][.])*`, false)
	add("LINK", `(?:https?://|www[.]|\w+[@.]\w\w+[@.])\w[\w./?&!%=+*"'@$#-]+`, false)
	add("HASHTAG", `[#@][\w-]+`, false)
	add("HTML", `<\w+.*?>|</\w+ *>`, false)
	add("PSEUDOHTML", `\[/?\w+\]`, false)
	if language == "fr" {
		add("WORDELD", "(?:l|d|n|m|t|s|j|c|ç|lorsqu|puisqu|jusqu|quoiqu|qu|presqu|quelqu)['’ʼ‘‛´`′‵՚ꞌꞋ]", false)
		add("WORDORD", `\d+(?:ers?|res?|è[rm]es?|i[èe][mr]es?|de?s?|nde?s?|ès?|es?|ᵉʳˢ?|ʳᵉˢ?|ᵈᵉ?ˢ?|ⁿᵈᵉ?ˢ?|ᵉˢ?)`, true)
	}
	add("HOUR", `\d\d?[h:]\d\d(?:[m:]\d\ds?|)`, true)
	t.patterns[len(t.patterns)-1].fallback = compile(`\d\d?[h:]\d\d`)
	if language == "fr" {
		add("NUM", `\d+(?:[.,]\d+|)`, false)
	} else {
		add("NUM", `\d+(?:[.,]\d+)`, false)
	}
	add("SIGN", `[&%‰€$+±=*/<>⩾⩽#|×¥£§¢฿¬÷@-]`, false)
	// Python's negative lookahead removes underscore from its Unicode word class.
	// The resulting explicit class is exactly letters, numbers and U+0300–036F.
	add("WORD", "[\\p{L}\\p{N}\\x{0300}-\\x{036f}]+(?:[’'`-][\\p{L}\\p{N}\\x{0300}-\\x{036f}]+)*", false)
	add("UNDERSCORE", `_+`, false)
	add("SYMBOL", `[\x{2600}-\x{26ff}\x{2700}-\x{27bf}\x{1f650}-\x{1f67f}\x{1f700}-\x{1f77f}\x{1f780}-\x{1f7ff}\x{1f800}-\x{1f8ff}]`, false)
	add("EMOJI", `[\x{1f300}-\x{1f5ff}\x{1f600}-\x{1f64f}\x{1f680}-\x{1f6ff}\x{1f900}-\x{1f9ff}]+`, false)
	return t
}

// compile lowers the fixed rule inventory, not arbitrary Python regex syntax.
// Python's Unicode word and digit classes differ from Go's ASCII shortcuts.
func compile(expression string) *regexp.Regexp {
	expression = strings.ReplaceAll(expression, `[a-zA-Z]`, `[a-zA-Zİı]`)
	expression = strings.ReplaceAll(expression, `[A-Z]`, `[A-Zİı]`)
	var b strings.Builder
	inClass := false
	for i := 0; i < len(expression); {
		r, size := utf8.DecodeRuneInString(expression[i:])
		i += size
		if r == '\\' && i < len(expression) {
			next, n := utf8.DecodeRuneInString(expression[i:])
			i += n
			switch next {
			case 'w':
				if inClass {
					b.WriteString(`\p{L}\p{N}_`)
				} else {
					b.WriteString(`[\p{L}\p{N}_]`)
				}
			case 'd':
				b.WriteString(`\p{Nd}`)
			default:
				b.WriteRune('\\')
				b.WriteRune(next)
			}
			continue
		}
		if r == '[' {
			inClass = true
		}
		if r == ']' {
			inClass = false
		}
		if (r == 'i' || r == 'I') && !inClass {
			b.WriteString(`[iİı]`)
		} else {
			b.WriteRune(r)
		}
	}
	return regexp.MustCompile(`(?i)^(?:` + b.String() + `)`)
}

// Tokens tokenizes the original string, skipping whitespace and preserving
// the upstream alternative order. includeBoundaries adds INFO sentinel tokens.
func (t *Tokenizer) Tokens(input string, includeBoundaries bool) []Token {
	result := []Token{}
	if includeBoundaries {
		result = append(result, Token{Index: 0, Type: "INFO", Value: "<start>", Morphology: []string{"<start>"}})
	}
	byteOffset, pointOffset, index := 0, 0, 0
	for byteOffset < len(input) {
		remaining := input[byteOffset:]
		length, kind := 0, ""
		for _, p := range t.patterns {
			match := p.expression.FindStringIndex(remaining)
			if match == nil {
				continue
			}
			end := match[1]
			if p.boundary && !boundaryAt(remaining, end) {
				// Only HOUR can backtrack across a non-word separator to a shorter
				// valid boundary. Ordinal suffixes contain word characters exclusively.
				if p.fallback == nil {
					continue
				}
				fallback := p.fallback.FindStringIndex(remaining)
				if fallback == nil || !boundaryAt(remaining, fallback[1]) {
					continue
				}
				end = fallback[1]
			}
			length, kind = end, p.kind
			break
		}
		if length == 0 {
			r, n := utf8.DecodeRuneInString(remaining)
			length = n
			if !pythonSpace(r) {
				kind = "OTHER"
			}
		}
		points := utf8.RuneCountInString(remaining[:length])
		if kind != "" {
			index++
			result = append(result, Token{Index: index, Type: kind, Value: norm.NFC.String(remaining[:length]), Start: pointOffset, End: pointOffset + points})
		}
		byteOffset += length
		pointOffset += points
	}
	if includeBoundaries {
		result = append(result, Token{Index: index + 1, Type: "INFO", Value: "<end>", Start: pointOffset, End: pointOffset, Morphology: []string{"<end>"}})
	}
	return result
}
func pythonWord(r rune) bool  { return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r) }
func pythonSpace(r rune) bool { return unicode.IsSpace(r) || (r >= 0x1c && r <= 0x1f) }
func boundaryAt(s string, end int) bool {
	if end == 0 {
		return false
	}
	left, _ := utf8.DecodeLastRuneInString(s[:end])
	if end == len(s) {
		return pythonWord(left)
	}
	right, _ := utf8.DecodeRuneInString(s[end:])
	return pythonWord(left) != pythonWord(right)
}
