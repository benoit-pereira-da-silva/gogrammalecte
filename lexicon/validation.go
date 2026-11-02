// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
// Ported from Grammalecte genfrdic.py and tags.py by Olivier R.

package lexicon

import (
	_ "embed"
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

//go:embed data/validation-tags.json
var validationTagData []byte
var validationTags = func() map[string][]string {
	var tags map[string][]string
	if err := json.Unmarshal(validationTagData, &tags); err != nil {
		panic(err)
	}
	return tags
}()
var verbCategory = regexp.MustCompile(`^v[0123]`)
var verbAttributes = regexp.MustCompile(`^[eas_][ix_][tx_][nx_][pqreuvx_][mx_][ex_z][ax_z](?:$|[^\p{L}\p{N}_])`)
var verbFlag = regexp.MustCompile(`^[abcdf]0`)
var participleFlag = regexp.MustCompile(`p[+.]`)
var transitiveVerb = regexp.MustCompile(`^v...t`)
var inflectionGender = regexp.MustCompile(`epi|mas|fem|inv|sg|pl`)
var inflectionNumber = regexp.MustCompile(`pl|sg|inv`)

// ValidationWarnings returns the original diagnostic fragments, including the
// duplicated number/gender check. Auxiliary fields use stable lx/se/et order.
func (e *Entry) ValidationWarnings() []string {
	warnings := []string{}
	add := func(s string) { warnings = append(warnings, s) }
	lemma := []rune(e.Lemma)
	if len(lemma) == 0 {
		add(" > lemme vide")
	}
	if len(lemma) > 0 && unicode.IsSpace(lemma[0]) {
		add(" > espace en début de lemme <" + e.Lemma + ">")
	}
	if len(lemma) > 0 && unicode.IsSpace(lemma[len(lemma)-1]) {
		add(" > espace en fin de lemme <" + e.Lemma + ">")
	}
	for _, name := range []string{"po", "is", "lx", "se", "et"} {
		value := e.Fields[name]
		if value == "" {
			continue
		}
		tags := strings.Fields(value)
		if name == "lx" || name == "se" || name == "et" {
			tags = strings.Split(value, " ")
		}
		for _, tag := range tags {
			if !slices.Contains(validationTags[name], tag) && (name == "is" || !verbCategory.MatchString(tag)) {
				add(" > Étiquette inconnue pour l’attribut <" + name + ">: " + tag)
			}
		}
	}
	po, flags, inflection := e.Fields["po"], e.Flags, e.Fields["is"]
	if verbCategory.MatchString(po) && !verbAttributes.MatchString(po[2:]) {
		add(" > verbe mal étiqueté: " + po)
	}
	if verbFlag.MatchString(flags) {
		if !participleFlag.MatchString(flags) {
			add(" > verbe sans participe passé: " + po)
		}
		if !strings.Contains(flags, "()") {
			add(" > drapeau () manquant")
		}
		if transitiveVerb.MatchString(po) && !strings.Contains(flags, "p+") {
			add(" > drapeau p+ absent sur un verbe étiqueté transitif")
		}
	}
	if startsAny(flags, "a0", "b0", "c0", "d0") && !strings.HasSuffix(e.Lemma, "er") {
		add(" > drapeau pour verbe du 1ᵉʳ groupe sur un lemme qui ne finit pas par -er")
	}
	if strings.HasPrefix(flags, "f") && !endsAny(e.Lemma, "ir", "ïr") {
		add(" > drapeau pour verbe du 2ᵉ groupe sur un lemme qui ne finit pas par -ir")
	}
	if (startsAny(flags, "S.", "S+") && endsAny(e.Lemma, "s", "x", "z")) || (strings.HasPrefix(flags, "X.") && !endsAny(e.Lemma, "u", "l")) {
		add(" > drapeau inutile")
	}
	if inflection == "" && flagStarts(flags, "SXAI") && po != "" {
		add(" > étiquette <is> vide")
	}
	if startsAny(inflection, "pl", "sg", "inv") {
		add(" > étiquettes <is> incomplètes, genre non spécifié (mas/fem/epi)")
	}
	if startsAny(flags, "F", "W") && inflectionGender.MatchString(inflection) {
		add(" > étiquettes <is> incohérentes")
	}
	if inflectionNumber.MatchString(inflection) && flagStarts(flags, "SXAIFGW") {
		add(" > étiquettes <is> incohérentes ")
		add(" > étiquettes <is> incohérentes ")
	}
	if endsAny(inflection, "mas", "fem", "epi") && (flags == "" || !startsAny(flags, "S", "X", "F", "W", "A", "I", "U")) {
		add(" > étiquettes <is> incomplètes")
	}
	if flagStarts(flags, "SXAIFGW") && !strings.Contains(flags, "()") {
		add(" > drapeau () manquant")
	}
	return warnings
}

func startsAny(s string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}
func endsAny(s string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}
func flagStarts(flags, letters string) bool {
	return len(flags) > 0 && strings.ContainsRune(letters, rune(flags[0])) && (len(flags) == 1 || flags[1] != '=')
}

// ValidationMessage formats one source validation diagnostic, or an empty string.
func (e *Entry) ValidationMessage() string {
	warnings := e.ValidationWarnings()
	if len(warnings) == 0 {
		return ""
	}
	return "   erreur " + strings.Join(warnings, "") + "   sur   " + e.Lemma + "/" + e.Flags + " " + e.Morphology(2)
}

// ValidationMessages returns diagnostics in the current dictionary entry order.
func (d *Dictionary) ValidationMessages() []string {
	messages := []string{}
	for _, entry := range d.Entries {
		if message := entry.ValidationMessage(); message != "" {
			messages = append(messages, message)
		}
	}
	return messages
}

// ShortDescription is the reference lemma/flags/subdictionary label.
func (e *Entry) ShortDescription() string {
	s := e.Lemma
	if e.Flags != "" {
		s += "/" + e.Flags
	}
	if e.Fields["di"] != "*" {
		s += " di:" + e.Fields["di"]
	}
	return s
}
