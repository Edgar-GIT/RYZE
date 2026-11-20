package taxid

import "testing"

func TestNormalize(t *testing.T) {
	valid := map[string]string{
		"123456789":      "123456789",
		" 123 456 789 ":  "123456789",
		"PT123456789":    "123456789",
		"pt 501 964 843": "501964843",
		"999999990":      "999999990",
	}
	for raw, want := range valid {
		if got, ok := Normalize(raw); !ok || got != want {
			t.Fatalf("Normalize(%q) = %q, %v; want %q", raw, got, ok, want)
		}
	}
	for _, raw := range []string{"", "12345678", "1234567890", "123456780", "012345678", "12345678a", "ES123456789"} {
		if got, ok := Normalize(raw); ok {
			t.Fatalf("Normalize(%q) accepted %q", raw, got)
		}
	}
}
