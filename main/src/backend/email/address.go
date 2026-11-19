package email

import (
	"net/mail"
	"strings"
)

// NormalizeAddress trims the surrounding whitespace from a recipient address.
func NormalizeAddress(address string) string {
	return strings.TrimSpace(address)
}

// IsValidAddress reports whether value is one bare, deliverable-looking email
// address. Display names, comments and other mailbox decorations are rejected
// because every RYZE email recipient is stored and sent as a plain address.
func IsValidAddress(value string) bool {
	address := NormalizeAddress(value)
	if address == "" || len(address) > 255 {
		return false
	}
	parsed, err := mail.ParseAddress(address)
	return err == nil && parsed.Address == address
}
