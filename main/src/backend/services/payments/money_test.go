package payments

import (
	"errors"
	"testing"
)

// TestMinorUnitsToDecimalString pins the conversion used to both create and
// verify a PayPal order. A change here would silently desynchronise the amount
// RYZE asks PayPal for from the amount RYZE accepts back, so the boundaries are
// asserted explicitly rather than derived from the implementation.
func TestMinorUnitsToDecimalString(t *testing.T) {
	tests := []struct {
		minorUnits int64
		want       string
	}{
		{minorUnits: 0, want: "0.00"},
		{minorUnits: 1, want: "0.01"},
		{minorUnits: 9, want: "0.09"},
		{minorUnits: 10, want: "0.10"},
		{minorUnits: 49, want: "0.49"},
		{minorUnits: 50, want: "0.50"},
		{minorUnits: 99, want: "0.99"},
		{minorUnits: 100, want: "1.00"},
		{minorUnits: 101, want: "1.01"},
		{minorUnits: 2500, want: "25.00"},
		{minorUnits: 4999, want: "49.99"},
		{minorUnits: 500000, want: "5000.00"},
		{minorUnits: 123456, want: "1234.56"},
		// A negative amount is never valid for a purchase, but the conversion
		// must stay well defined so it can never be formatted into a value that
		// a provider would read as a positive charge.
		{minorUnits: -100, want: "-1.00"},
		{minorUnits: -4999, want: "-49.99"},
	}

	for _, test := range tests {
		if got := MinorUnitsToDecimalString(test.minorUnits); got != test.want {
			t.Errorf("MinorUnitsToDecimalString(%d) = %q, want %q", test.minorUnits, got, test.want)
		}
	}
}

// TestValidateCurrencyCode asserts the shape check every provider relies on to
// fail closed before contacting a provider. It must accept any case and
// surrounding whitespace, because the stored snapshot, Stripe and PayPal each
// use a different case convention, and it must reject anything that is not
// three letters.
func TestValidateCurrencyCode(t *testing.T) {
	valid := []string{"EUR", "eur", "Usd", "gbp", " GBP ", "JPY"}
	for _, currency := range valid {
		if err := ValidateCurrencyCode(currency); err != nil {
			t.Errorf("ValidateCurrencyCode(%q) = %v, want nil", currency, err)
		}
	}

	invalid := []string{"", " ", "E", "EU", "EURO", "E1R", "12", "€", "E R"}
	for _, currency := range invalid {
		if err := ValidateCurrencyCode(currency); err == nil {
			t.Errorf("ValidateCurrencyCode(%q) = nil, want error", currency)
		}
	}
}

// TestValidateCurrencyCodeDoesNotRestrictCurrencies documents that the shared
// check validates shape only. Which currencies RYZE may price a program in is a
// business decision made when the price is set, so this helper must never grow
// into a hidden allow-list that would reject a legitimately stored snapshot.
func TestValidateCurrencyCodeDoesNotRestrictCurrencies(t *testing.T) {
	for _, currency := range []string{"JPY", "GBP", "USD", "BRL"} {
		if err := ValidateCurrencyCode(currency); err != nil {
			t.Errorf("ValidateCurrencyCode(%q) = %v, want nil — shape validation must not act as a currency allow-list", currency, err)
		}
	}
}

// TestStripeCurrencyCodeReusesSharedShapeCheck confirms the Stripe entry point
// still fails closed with a provider error after being rebuilt on top of the
// shared validator, so the refactor preserved its original behaviour.
func TestStripeCurrencyCodeReusesSharedShapeCheck(t *testing.T) {
	if code, err := stripeCurrencyCode(" EUR "); err != nil || code != "eur" {
		t.Fatalf("stripeCurrencyCode(\" EUR \") = %q, %v; want \"eur\", nil", code, err)
	}
	for _, currency := range []string{"", "EURO", "E1R"} {
		if _, err := stripeCurrencyCode(currency); !errors.Is(err, ErrProviderFailure) {
			t.Errorf("stripeCurrencyCode(%q) = %v, want ErrProviderFailure", currency, err)
		}
	}
}
