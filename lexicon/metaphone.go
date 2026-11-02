// Double Metaphone algorithm, copyright 1998, 1999 Lawrence Philips.
// Based on Kevin Atkinson's C implementation and Andrew Collins's 2007 Python
// translation (which claims no rights), with Matthew Somerville's 2009 fixes.
// This native translation preserves the pinned Grammalecte metaphone2.py behavior.

package lexicon

import (
	"strings"
	"unicode"

	"github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/strtransform"
	"golang.org/x/text/unicode/norm"
)

type metaphoneStep struct {
	primary, secondary string
	advance            int
}

// DoubleMetaphone returns the complete primary and alternate phonetic codes.
// It preserves the reference's unlimited code length and historical edge cases.
func DoubleMetaphone(word string) [2]string {
	word = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, norm.NFD.String(word))
	word = strtransform.Upper(word)
	slavoGermanic := strings.Contains(word, "W") || strings.Contains(word, "K") || strings.Contains(word, "CZ") || strings.Contains(word, "WITZ")
	first := 2
	letters := []rune("--" + word + "------")
	last := len(letters) - 7
	position := first
	primary, secondary := "", ""
	if metaphoneIn(metaphoneSlice(letters, first, first+2), "GN", "KN", "PN", "WR", "PS") {
		position++
	}
	if letters[first] == 'X' {
		primary = "S"
		secondary = "S"
		position++
	}
	for position <= last {
		character := string(letters[position])
		next := metaphoneStep{"", "", 1}
		if strings.Contains("AEIOUY", character) {
			next = metaphoneStep{"", "", 1}
			if position == first {
				next = metaphoneStep{"A", "A", 1}
			}
		} else if character == "B" {
			if string(letters[(position+1)]) == "B" {
				next = metaphoneStep{"P", "P", 2}
			} else {
				next = metaphoneStep{"P", "P", 1}
			}
		} else if character == "C" {
			if (position > (first + 1)) && !strings.Contains("AEIOUY", string(letters[(position-2)])) && (metaphoneSlice(letters, (position-1), (position+2)) == "ACH") && !metaphoneIn(string(letters[(position+2)]), "I") && (!metaphoneIn(string(letters[(position+2)]), "E") || metaphoneIn(metaphoneSlice(letters, (position-2), (position+4)), "BACHER", "MACHER")) {
				next = metaphoneStep{"K", "K", 2}
			} else if (position == first) && (metaphoneSlice(letters, first, (first+6)) == "CAESAR") {
				next = metaphoneStep{"S", "S", 2}
			} else if metaphoneSlice(letters, position, (position+4)) == "CHIA" {
				next = metaphoneStep{"K", "K", 2}
			} else if metaphoneSlice(letters, position, (position+2)) == "CH" {
				if (position > first) && (metaphoneSlice(letters, position, (position+4)) == "CHAE") {
					next = metaphoneStep{"K", "X", 2}
				} else if (position == first) && (metaphoneIn(metaphoneSlice(letters, (position+1), (position+6)), "HARAC", "HARIS") || metaphoneIn(metaphoneSlice(letters, (position+1), (position+4)), "HOR", "HYM", "HIA", "HEM")) && (metaphoneSlice(letters, first, (first+5)) != "CHORE") {
					next = metaphoneStep{"K", "K", 2}
				} else if metaphoneIn(metaphoneSlice(letters, first, (first+4)), "VAN ", "VON ") || (metaphoneSlice(letters, first, (first+3)) == "SCH") || metaphoneIn(metaphoneSlice(letters, (position-2), (position+4)), "ORCHES", "ARCHIT", "ORCHID") || metaphoneIn(string(letters[(position+2)]), "T", "S") || ((metaphoneIn(string(letters[(position-1)]), "A", "O", "U", "E") || (position == first)) && metaphoneIn(string(letters[(position+2)]), "L", "R", "N", "M", "B", "H", "F", "V", "W")) {
					next = metaphoneStep{"K", "K", 2}
				} else if position > first {
					if metaphoneSlice(letters, first, (first+2)) == "MC" {
						next = metaphoneStep{"K", "K", 2}
					} else {
						next = metaphoneStep{"X", "K", 2}
					}
				} else {
					next = metaphoneStep{"X", "X", 2}
				}
			} else if (metaphoneSlice(letters, position, (position+2)) == "CZ") && (metaphoneSlice(letters, (position-2), (position+2)) != "WICZ") {
				next = metaphoneStep{"S", "X", 2}
			} else if metaphoneSlice(letters, (position+1), (position+4)) == "CIA" {
				next = metaphoneStep{"X", "X", 3}
			} else if (metaphoneSlice(letters, position, (position+2)) == "CC") && !((position == (first + 1)) && (string(letters[first]) == "M")) {
				if metaphoneIn(string(letters[(position+2)]), "I", "E", "H") && (metaphoneSlice(letters, (position+2), (position+4)) != "HU") {
					if ((position == (first + 1)) && (string(letters[first]) == "A")) || metaphoneIn(metaphoneSlice(letters, (position-1), (position+4)), "UCCEE", "UCCES") {
						next = metaphoneStep{"KS", "KS", 3}
					} else {
						next = metaphoneStep{"X", "X", 3}
					}
				} else {
					next = metaphoneStep{"K", "K", 2}
				}
			} else if metaphoneIn(metaphoneSlice(letters, position, (position+2)), "CK", "CG", "CQ") {
				next = metaphoneStep{"K", "K", 2}
			} else if metaphoneIn(metaphoneSlice(letters, position, (position+2)), "CI", "CE", "CY") {
				if metaphoneIn(metaphoneSlice(letters, position, (position+3)), "CIO", "CIE", "CIA") {
					next = metaphoneStep{"S", "X", 2}
				} else {
					next = metaphoneStep{"S", "S", 2}
				}
			} else if metaphoneIn(metaphoneSlice(letters, (position+1), (position+3)), " C", " Q", " G") {
				next = metaphoneStep{"K", "K", 3}
			} else if metaphoneIn(string(letters[(position+1)]), "C", "K", "Q") && !metaphoneIn(metaphoneSlice(letters, (position+1), (position+3)), "CE", "CI") {
				next = metaphoneStep{"K", "K", 2}
			} else {
				next = metaphoneStep{"K", "K", 1}
			}
		} else if character == "Ç" {
			next = metaphoneStep{"S", "S", 1}
		} else if character == "D" {
			if metaphoneSlice(letters, position, (position+2)) == "DG" {
				if metaphoneIn(string(letters[(position+2)]), "I", "E", "Y") {
					next = metaphoneStep{"J", "J", 3}
				} else {
					next = metaphoneStep{"TK", "TK", 2}
				}
			} else if metaphoneIn(metaphoneSlice(letters, position, (position+2)), "DT", "DD") {
				next = metaphoneStep{"T", "T", 2}
			} else {
				next = metaphoneStep{"T", "T", 1}
			}
		} else if character == "F" {
			if string(letters[(position+1)]) == "F" {
				next = metaphoneStep{"F", "F", 2}
			} else {
				next = metaphoneStep{"F", "F", 1}
			}
		} else if character == "G" {
			if string(letters[(position+1)]) == "H" {
				if (position > first) && !strings.Contains("AEIOUY", string(letters[(position-1)])) {
					next = metaphoneStep{"K", "K", 2}
				} else if position < (first + 3) {
					if position == first {
						if string(letters[(position+2)]) == "I" {
							next = metaphoneStep{"J", "J", 2}
						} else {
							next = metaphoneStep{"K", "K", 2}
						}
					}
				} else if ((position > (first + 1)) && metaphoneIn(string(letters[(position-2)]), "B", "H", "D")) || ((position > (first + 2)) && metaphoneIn(string(letters[(position-3)]), "B", "H", "D")) || ((position > (first + 3)) && metaphoneIn(string(letters[(position-3)]), "B", "H")) {
					next = metaphoneStep{"", "", 2}
				} else if (position > (first + 2)) && (string(letters[(position-1)]) == "U") && metaphoneIn(string(letters[(position-3)]), "C", "G", "L", "R", "T") {
					next = metaphoneStep{"F", "F", 2}
				} else if (position > first) && (string(letters[(position-1)]) != "I") {
					next = metaphoneStep{"K", "K", 2}
				}
			} else if string(letters[(position+1)]) == "N" {
				if (position == (first + 1)) && strings.Contains("AEIOUY", string(letters[first])) && !(slavoGermanic) {
					next = metaphoneStep{"KN", "N", 2}
				} else if (metaphoneSlice(letters, (position+2), (position+4)) != "EY") && (string(letters[(position+1)]) != "Y") && !(slavoGermanic) {
					next = metaphoneStep{"N", "KN", 2}
				} else {
					next = metaphoneStep{"KN", "KN", 2}
				}
			} else if (metaphoneSlice(letters, (position+1), (position+3)) == "LI") && !(slavoGermanic) {
				next = metaphoneStep{"KL", "L", 2}
			} else if (position == first) && ((string(letters[(position+1)]) == "Y") || metaphoneIn(metaphoneSlice(letters, (position+1), (position+3)), "ES", "EP", "EB", "EL", "EY", "IB", "IL", "IN", "IE", "EI", "ER")) {
				next = metaphoneStep{"K", "J", 2}
			} else if ((metaphoneSlice(letters, (position+1), (position+3)) == "ER") || (string(letters[(position+1)]) == "Y")) && !metaphoneIn(metaphoneSlice(letters, first, (first+6)), "DANGER", "RANGER", "MANGER") && !metaphoneIn(string(letters[(position-1)]), "E", "I") && !metaphoneIn(metaphoneSlice(letters, (position-1), (position+2)), "RGY", "OGY") {
				next = metaphoneStep{"K", "J", 2}
			} else if metaphoneIn(string(letters[(position+1)]), "E", "I", "Y") || metaphoneIn(metaphoneSlice(letters, (position-1), (position+3)), "AGGI", "OGGI") {
				if metaphoneIn(metaphoneSlice(letters, first, (first+4)), "VON ", "VAN ") || (metaphoneSlice(letters, first, (first+3)) == "SCH") || (metaphoneSlice(letters, (position+1), (position+3)) == "ET") {
					next = metaphoneStep{"K", "K", 2}
				} else if metaphoneSlice(letters, (position+1), (position+5)) == "IER " {
					next = metaphoneStep{"J", "J", 2}
				} else {
					next = metaphoneStep{"J", "K", 2}
				}
			} else if string(letters[(position+1)]) == "G" {
				next = metaphoneStep{"K", "K", 2}
			} else {
				next = metaphoneStep{"K", "K", 1}
			}
		} else if character == "H" {
			if ((position == first) || strings.Contains("AEIOUY", string(letters[(position-1)]))) && strings.Contains("AEIOUY", string(letters[(position+1)])) {
				next = metaphoneStep{"H", "H", 2}
			} else {
				next = metaphoneStep{"", "", 1}
			}
		} else if character == "J" {
			if (metaphoneSlice(letters, position, (position+4)) == "JOSE") || (metaphoneSlice(letters, first, (first+4)) == "SAN ") {
				if ((position == first) && (string(letters[(position+4)]) == " ")) || (metaphoneSlice(letters, first, (first+4)) == "SAN ") {
					next = metaphoneStep{"H", "H", 0}
				} else {
					next = metaphoneStep{"J", "H", 0}
				}
			} else if (position == first) && (metaphoneSlice(letters, position, (position+4)) != "JOSE") {
				next = metaphoneStep{"J", "A", 0}
			} else if strings.Contains("AEIOUY", string(letters[(position-1)])) && !(slavoGermanic) && metaphoneIn(string(letters[(position+1)]), "A", "O") {
				next = metaphoneStep{"J", "H", 0}
			} else if position == last {
				next = metaphoneStep{"J", " ", 0}
			} else if !metaphoneIn(string(letters[(position+1)]), "L", "T", "K", "S", "N", "M", "B", "Z") && !metaphoneIn(string(letters[(position-1)]), "S", "K", "L") {
				next = metaphoneStep{"J", "J", 0}
			} else {
				next = metaphoneStep{"", "", 0}
			}
			if string(letters[(position+1)]) == "J" {
				next.advance = 2
			} else {
				next.advance = 1
			}
		} else if character == "K" {
			if string(letters[(position+1)]) == "K" {
				next = metaphoneStep{"K", "K", 2}
			} else {
				next = metaphoneStep{"K", "K", 1}
			}
		} else if character == "L" {
			if string(letters[(position+1)]) == "L" {
				if ((position == (last - 2)) && metaphoneIn(metaphoneSlice(letters, (position-1), (position+3)), "ILLO", "ILLA", "ALLE")) || ((metaphoneIn(metaphoneSlice(letters, (last-1), (last+1)), "AS", "OS") || metaphoneIn(string(letters[last]), "A", "O")) && (metaphoneSlice(letters, (position-1), (position+3)) == "ALLE")) {
					next = metaphoneStep{"L", " ", 2}
				} else {
					next = metaphoneStep{"L", "L", 2}
				}
			} else {
				next = metaphoneStep{"L", "L", 1}
			}
		} else if character == "M" {
			if ((metaphoneSlice(letters, (position+1), (position+4)) == "UMB") && (((position + 1) == last) || (metaphoneSlice(letters, (position+2), (position+4)) == "ER"))) || (string(letters[(position+1)]) == "M") {
				next = metaphoneStep{"M", "M", 2}
			} else {
				next = metaphoneStep{"M", "M", 1}
			}
		} else if character == "N" {
			if string(letters[(position+1)]) == "N" {
				next = metaphoneStep{"N", "N", 2}
			} else {
				next = metaphoneStep{"N", "N", 1}
			}
		} else if character == "Ñ" {
			next = metaphoneStep{"N", "N", 1}
		} else if character == "P" {
			if string(letters[(position+1)]) == "H" {
				next = metaphoneStep{"F", "F", 2}
			} else if metaphoneIn(string(letters[(position+1)]), "P", "B") {
				next = metaphoneStep{"P", "P", 2}
			} else {
				next = metaphoneStep{"P", "P", 1}
			}
		} else if character == "Q" {
			if string(letters[(position+1)]) == "Q" {
				next = metaphoneStep{"K", "K", 2}
			} else {
				next = metaphoneStep{"K", "K", 1}
			}
		} else if character == "R" {
			if (position == last) && !(slavoGermanic) && (metaphoneSlice(letters, (position-2), position) == "IE") && !metaphoneIn(metaphoneSlice(letters, (position-4), (position-2)), "ME", "MA") {
				next = metaphoneStep{"", "R", 0}
			} else {
				next = metaphoneStep{"R", "R", 0}
			}
			if string(letters[(position+1)]) == "R" {
				next.advance = 2
			} else {
				next.advance = 1
			}
		} else if character == "S" {
			if metaphoneIn(metaphoneSlice(letters, (position-1), (position+2)), "ISL", "YSL") {
				next = metaphoneStep{"", "", 1}
			} else if (position == first) && (metaphoneSlice(letters, first, (first+5)) == "SUGAR") {
				next = metaphoneStep{"X", "S", 1}
			} else if metaphoneSlice(letters, position, (position+2)) == "SH" {
				if metaphoneIn(metaphoneSlice(letters, (position+1), (position+5)), "HEIM", "HOEK", "HOLM", "HOLZ") {
					next = metaphoneStep{"S", "S", 2}
				} else {
					next = metaphoneStep{"X", "X", 2}
				}
			} else if metaphoneIn(metaphoneSlice(letters, position, (position+3)), "SIO", "SIA") || (metaphoneSlice(letters, position, (position+4)) == "SIAN") {
				if !(slavoGermanic) {
					next = metaphoneStep{"S", "X", 3}
				} else {
					next = metaphoneStep{"S", "S", 3}
				}
			} else if ((position == first) && metaphoneIn(string(letters[(position+1)]), "M", "N", "L", "W")) || (string(letters[(position+1)]) == "Z") {
				next = metaphoneStep{"S", "X", 0}
				if string(letters[(position+1)]) == "Z" {
					next.advance = 2
				} else {
					next.advance = 1
				}
			} else if metaphoneSlice(letters, position, (position+2)) == "SC" {
				if string(letters[(position+2)]) == "H" {
					if metaphoneIn(metaphoneSlice(letters, (position+3), (position+5)), "OO", "ER", "EN", "UY", "ED", "EM") {
						if metaphoneIn(metaphoneSlice(letters, (position+3), (position+5)), "ER", "EN") {
							next = metaphoneStep{"X", "SK", 3}
						} else {
							next = metaphoneStep{"SK", "SK", 3}
						}
					} else if (position == first) && !strings.Contains("AEIOUY", string(letters[(first+3)])) && (string(letters[(first+3)]) != "W") {
						next = metaphoneStep{"X", "S", 3}
					} else {
						next = metaphoneStep{"X", "X", 3}
					}
				} else if metaphoneIn(string(letters[(position+2)]), "I", "E", "Y") {
					next = metaphoneStep{"S", "S", 3}
				} else {
					next = metaphoneStep{"SK", "SK", 3}
				}
			} else if (position == last) && metaphoneIn(metaphoneSlice(letters, (position-2), position), "AI", "OI") {
				next = metaphoneStep{"", "S", 1}
			} else {
				next = metaphoneStep{"S", "S", 0}
				if metaphoneIn(string(letters[(position+1)]), "S", "Z") {
					next.advance = 2
				} else {
					next.advance = 1
				}
			}
		} else if character == "T" {
			if metaphoneSlice(letters, position, (position+4)) == "TION" {
				next = metaphoneStep{"X", "X", 3}
			} else if metaphoneIn(metaphoneSlice(letters, position, (position+3)), "TIA", "TCH") {
				next = metaphoneStep{"X", "X", 3}
			} else if (metaphoneSlice(letters, position, (position+2)) == "TH") || (metaphoneSlice(letters, position, (position+3)) == "TTH") {
				if metaphoneIn(metaphoneSlice(letters, (position+2), (position+4)), "OM", "AM") || metaphoneIn(metaphoneSlice(letters, first, (first+4)), "VON ", "VAN ") || (metaphoneSlice(letters, first, (first+3)) == "SCH") {
					next = metaphoneStep{"T", "T", 2}
				} else {
					next = metaphoneStep{"0", "T", 2}
				}
			} else if metaphoneIn(string(letters[(position+1)]), "T", "D") {
				next = metaphoneStep{"T", "T", 2}
			} else {
				next = metaphoneStep{"T", "T", 1}
			}
		} else if character == "V" {
			if string(letters[(position+1)]) == "V" {
				next = metaphoneStep{"F", "F", 2}
			} else {
				next = metaphoneStep{"F", "F", 1}
			}
		} else if character == "W" {
			if metaphoneSlice(letters, position, (position+2)) == "WR" {
				next = metaphoneStep{"R", "R", 2}
			} else if (position == first) && (strings.Contains("AEIOUY", string(letters[(position+1)])) || (metaphoneSlice(letters, position, (position+2)) == "WH")) {
				if strings.Contains("AEIOUY", string(letters[(position+1)])) {
					next = metaphoneStep{"A", "F", 1}
				} else {
					next = metaphoneStep{"A", "A", 1}
				}
			} else if ((position == last) && strings.Contains("AEIOUY", string(letters[(position-1)]))) || metaphoneIn(metaphoneSlice(letters, (position-1), (position+4)), "EWSKI", "EWSKY", "OWSKI", "OWSKY") || (metaphoneSlice(letters, first, (first+3)) == "SCH") {
				next = metaphoneStep{"", "F", 1}
			} else if metaphoneIn(metaphoneSlice(letters, position, (position+4)), "WICZ", "WITZ") {
				next = metaphoneStep{"TS", "FX", 4}
			} else {
				next = metaphoneStep{"", "", 1}
			}
		} else if character == "X" {
			next = metaphoneStep{"", "", 0}
			if !((position == last) && (metaphoneIn(metaphoneSlice(letters, (position-3), position), "IAU", "EAU") || metaphoneIn(metaphoneSlice(letters, (position-2), position), "AU", "OU"))) {
				next = metaphoneStep{"KS", "KS", 0}
			}
			if metaphoneIn(string(letters[(position+1)]), "C", "X") {
				next.advance = 2
			} else {
				next.advance = 1
			}
		} else if character == "Z" {
			if string(letters[(position+1)]) == "H" {
				next = metaphoneStep{"J", "J", 0}
			} else if metaphoneIn(metaphoneSlice(letters, (position+1), (position+3)), "ZO", "ZI", "ZA") || (slavoGermanic && (position > first) && (string(letters[(position-1)]) != "T")) {
				next = metaphoneStep{"S", "TS", 0}
			} else {
				next = metaphoneStep{"S", "S", 0}
			}
			if (string(letters[(position+1)]) == "Z") || (string(letters[(position+1)]) == "H") {
				next.advance = 2
			} else {
				next.advance = 1
			}
		}
		primary += next.primary
		secondary += next.secondary
		position += next.advance
	}
	if primary == secondary {
		secondary = ""
	}
	return [2]string{primary, secondary}
}

func metaphoneIn(value string, alternatives ...string) bool {
	for _, candidate := range alternatives {
		if value == candidate {
			return true
		}
	}
	return false
}

func metaphoneSlice(letters []rune, start, end int) string {
	if start < 0 {
		start += len(letters)
	}
	if end < 0 {
		end += len(letters)
	}
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}
	if start > len(letters) {
		start = len(letters)
	}
	if end > len(letters) {
		end = len(letters)
	}
	if end < start {
		return ""
	}
	return string(letters[start:end])
}
