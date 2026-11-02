package french

import (
	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/strtransform"
	"strings"
)

// IsMasculineForm reports whether a masculine spelling has registered feminine forms.
func IsMasculineForm(word string) bool {
	loadInflection()
	_, ok := inflection.Masculine[word]
	return ok
}

// FeminineForms returns the singular or plural feminine forms, preserving table order.
func FeminineForms(word string, plural bool) []string {
	loadInflection()
	index, ok := inflection.Masculine[word]
	if !ok {
		return []string{}
	}
	code := inflection.FeminineTags[index]
	cut := strings.IndexByte(code, '/')
	if plural {
		code = code[cut+1:]
	} else {
		code = code[:cut]
	}
	return applySuffixes(word, strings.Split(code, "|"))
}

// HasIrregularPlural reports whether a word has a registered irregular plural.
func HasIrregularPlural(word string) bool {
	loadInflection()
	_, ok := inflection.Plurals[word]
	return ok
}

// IrregularPlurals returns all registered plural forms in source order.
func IrregularPlurals(word string) []string {
	loadInflection()
	index, ok := inflection.Plurals[word]
	if !ok {
		return []string{}
	}
	return applySuffixes(word, strings.Split(inflection.PluralTags[index], "|"))
}
func applySuffixes(word string, codes []string) []string {
	out := make([]string, 0, len(codes))
	for _, code := range codes {
		out = append(out, modifySuffix(word, code))
	}
	return out
}
func modifySuffix(word, code string) string {
	if code == "" {
		return ""
	}
	result, err := strtransform.ChangeWordWithSuffixCode(word, code)
	if err != nil {
		panic(err)
	}
	return result
}
