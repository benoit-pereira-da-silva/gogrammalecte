package french

import (
	"strconv"
	"strings"
	"time"
	"unicode"
)

var monthNumbers = map[string]int{"janvier": 1, "février": 2, "mars": 3, "avril": 4, "mai": 5, "juin": 6, "juillet": 7, "août": 8, "aout": 8, "septembre": 9, "octobre": 10, "novembre": 11, "décembre": 12}
var dayNames = []string{"dimanche", "lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi"}

// CheckDate validates a day, numeric or French month, and year using the
// reference's proleptic Gregorian date validity, including years 1 through 9999.
func CheckDate(day, month, year string) (time.Time, bool) {
	d, ok := pythonInteger(day)
	if !ok {
		return time.Time{}, false
	}
	y, ok := pythonInteger(year)
	if !ok || y < 1 || y > 9999 {
		return time.Time{}, false
	}
	m := 0
	if decimalDigits(month) {
		m, ok = pythonInteger(month)
		if !ok {
			return time.Time{}, false
		}
	} else {
		m = monthNumbers[strings.ToLower(month)]
	}
	if m < 1 || m > 12 || d < 1 || d > 31 {
		return time.Time{}, false
	}
	date := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
	if date.Day() != d || date.Month() != time.Month(m) {
		return time.Time{}, false
	}
	return date, true
}

// WeekdayName preserves the reference's historical French transition convention:
// dates before December 10, 1582 use its fixed weekday shift, and December 10–19
// have no weekday. This deliberately reproduces the reference, not a general
// astronomical Julian-calendar conversion.
func WeekdayName(day, month, year string) string {
	date, ok := CheckDate(day, month, year)
	if !ok {
		return ""
	}
	weekday := int(date.Weekday())
	if date.Year() > 1582 {
		return dayNames[weekday]
	}
	if date.Year() < 1582 || date.Month() < 12 || date.Day() <= 9 {
		return dayNames[(weekday+3)%7]
	}
	if date.Day() >= 20 {
		return dayNames[weekday]
	}
	return ""
}

// WeekdayMatches treats an invalid date as compatible with any weekday, as the
// reference does, leaving date validation to the corresponding grammar rule.
func WeekdayMatches(weekday, day, month, year string) bool {
	if _, ok := CheckDate(day, month, year); !ok {
		return true
	}
	return WeekdayName(day, month, year) == strings.ToLower(weekday)
}
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
func decimalDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if _, ok := decimalValue(r); !ok {
			return false
		}
	}
	return true
}
func pythonInteger(s string) (int, bool) {
	s = strings.TrimFunc(s, func(r rune) bool { return unicode.IsSpace(r) || (r >= 0x1c && r <= 0x1f) })
	runes := []rune(s)
	if len(runes) == 0 {
		return 0, false
	}
	var b strings.Builder
	start := 0
	if runes[0] == '+' || runes[0] == '-' {
		b.WriteRune(runes[0])
		start = 1
	}
	if start == len(runes) {
		return 0, false
	}
	previousDigit := false
	for i := start; i < len(runes); i++ {
		r := runes[i]
		if r == '_' {
			if !previousDigit || i+1 == len(runes) {
				return 0, false
			}
			previousDigit = false
			continue
		}
		digit, ok := decimalValue(r)
		if !ok {
			return 0, false
		}
		previousDigit = true
		b.WriteByte(byte('0' + digit))
	}
	n, e := strconv.Atoi(b.String())
	return n, e == nil
}
