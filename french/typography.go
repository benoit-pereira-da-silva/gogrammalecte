package french

import (
	"regexp"
	"strings"
	"unicode"
)

// SplitVerb separates a compound verb from its enclitic pronouns. For a string
// without a hyphen it preserves the upstream last-code-point split.
func SplitVerb(verb string) (string, string) {
	r := []rune(verb)
	right := -1
	for i, c := range r {
		if c == '-' {
			right = i
		}
	}
	if right < 0 {
		right = len(r) - 1
		if right < 0 {
			right = 0
		}
	}
	suffix := string(r[right:])
	verb = string(r[:right])
	for _, ending := range []string{"-t", "-le", "-la", "-les", "-nous", "-vous", "-leur", "-lui"} {
		if strings.HasSuffix(verb, ending) {
			i := strings.LastIndexByte(verb, '-')
			return verb[:i], verb[i:] + suffix
		}
	}
	return verb, suffix
}

// JoinVerbAndSuffix applies the reference's euphonic and imperative linking rules.
func JoinVerbAndSuffix(verb, suffix string) string {
	if (strings.HasPrefix(suffix, "-t-") || strings.HasPrefix(suffix, "-T-")) && strings.ContainsAny(lastRune(verb), "tdTD") {
		return verb + suffix[2:]
	}
	if strings.ContainsAny(lastRune(verb), "eacEAC") {
		lower := strings.ToLower(suffix)
		if lower == "-en" || lower == "-y" {
			return verb + "s" + suffix
		}
		if lower == "-il" || lower == "-iel" || lower == "-elle" || lower == "-on" {
			return verb + "-t" + suffix
		}
	}
	return verb + suffix
}
func lastRune(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return ""
	}
	return string(r[len(r)-1])
}

// SuggestDemonstrative suggests ce/cet using the reference's initial-letter rule.
func SuggestDemonstrative(word string) string {
	r := []rune(word)
	if len(r) == 0 {
		return "ce|cet"
	}
	first := unicode.ToLower(r[0])
	if strings.ContainsRune("aeéèêiouyâîï", first) || r[0] == 'İ' || r[0] == 'ı' {
		return "cet"
	}
	if first == 'h' {
		return "ce|cet"
	}
	return "ce"
}

// FormatNumber returns pipe-separated grouping and telephone formatting suggestions.
func FormatNumber(number string, simple bool) string {
	r := []rune(number)
	n := len(r)
	if n < 4 {
		return number
	}
	if i := strings.IndexByte(number, ','); i >= 0 {
		return groupNumber(number[:i], 3) + number[i:]
	}
	result := groupNumber(number, 3)
	if simple {
		return result
	}
	binary := true
	for _, c := range r {
		if c != '0' && c != '1' {
			binary = false
		}
	}
	if binary {
		result += "|" + groupNumber(number, 4)
	}
	if n == 10 {
		if r[0] == '0' {
			result += "|" + groupNumber(number, 2)
			if r[1] == '4' && (r[2] == '7' || r[2] == '8' || r[2] == '9') {
				result += "|" + string(r[:4]) + " " + string(r[4:6]) + " " + string(r[6:8]) + " " + string(r[8:])
			}
			result += "|" + string(r[:3]) + " " + string(r[3:6]) + " " + string(r[6:8]) + " " + string(r[8:])
		}
		result += "|" + string(r[:4]) + " " + string(r[4:7]) + "-" + string(r[7:])
	} else if n == 9 && r[0] == '0' {
		result += "|" + string(r[:3]) + " " + string(r[3:5]) + " " + string(r[5:7]) + " " + string(r[7:9])
		result += "|" + string(r[:2]) + " " + string(r[2:5]) + " " + string(r[5:7]) + " " + string(r[7:9])
	}
	return result
}
func groupNumber(number string, size int) string {
	r := []rune(number)
	parts := []string{}
	for start := len(r); start > 0; {
		end := start
		start = max(0, start-size)
		parts = append(parts, string(r[start:end]))
	}
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, " ")
}

var standardPattern = regexp.MustCompile(`^NF[  -]?(C|E|P|Q|S|X|Z|EN(?:[  -]ISO|))[  -]?([0-9]+(?:[/‑-][0-9]+|))`)

// FormatFrenchStandard normalizes a French standard reference's spacing and hyphens.
func FormatFrenchStandard(input string) string {
	m := standardPattern.FindStringSubmatch(input)
	if m == nil {
		return ""
	}
	kind := strings.NewReplacer(" ", " ", "-", " ").Replace(strings.ToUpper(m[1]))
	number := strings.NewReplacer("/", "‑", "-", "‑").Replace(m[2])
	return "NF " + kind + " " + number
}

// UndoLigature expands a supported typographic ligature or returns the upstream placeholder.
func UndoLigature(character string) string {
	v, ok := map[string]string{"ﬁ": "fi", "ﬂ": "fl", "ﬀ": "ff", "ﬃ": "ffi", "ﬄ": "ffl", "ﬅ": "ft", "ﬆ": "st"}[character]
	if !ok {
		return "_"
	}
	return v
}

// NormalizeInclusiveWriting replaces supported inclusive separators with middle dots.
func NormalizeInclusiveWriting(token string) string {
	token = strings.Map(func(r rune) rune {
		if strings.ContainsRune("().·•–—/", r) {
			return '·'
		}
		return r
	}, token)
	return strings.NewReplacer("èr·", "er·", "ÈR·", "ER·").Replace(token)
}
