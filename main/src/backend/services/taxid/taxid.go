// Package taxid validates the Portuguese taxpayer number (NIF) a buyer may
// give at checkout for their purchase document.
package taxid

import "strings"

// Normalize validates a NIF as typed: spaces and an optional "PT" prefix are
// ignored, then it must be nine digits, not starting with 0, that pass the
// check digit (weights 9..2, modulo 11). ok is false for anything else.
func Normalize(raw string) (normalized string, ok bool) {
	value := strings.ToUpper(strings.Join(strings.Fields(raw), ""))
	value = strings.TrimPrefix(value, "PT")
	if len(value) != 9 || value[0] == '0' {
		return "", false
	}
	sum := 0
	for index := 0; index < 9; index++ {
		digit := value[index]
		if digit < '0' || digit > '9' {
			return "", false
		}
		if index < 8 {
			sum += int(digit-'0') * (9 - index)
		}
	}
	check := 11 - sum%11
	if check >= 10 {
		check = 0
	}
	if int(value[8]-'0') != check {
		return "", false
	}
	return value, true
}
