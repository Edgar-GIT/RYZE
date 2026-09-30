package payments

import (
	"fmt"
	"strings"
)

// minorUnitsPerMajorUnit is the number of minor units in one major currency
// unit for every currency RYZE supports. EUR — the only currency accepted for
// program pricing — has two decimal places, so a stored amount of 4999 minor
// units means 49.99 EUR. Any future currency with a different number of decimal
// places must introduce its own conversion here rather than silently inheriting
// this one.
const minorUnitsPerMajorUnit = 100

// MinorUnitsToDecimalString converts a minor-unit amount (e.g. 4999) into the
// decimal string (e.g. "49.99") required by providers that express amounts in
// major units, such as the PayPal Orders API.
//
// It lives here, in the payments domain, because the same conversion is needed
// both to create a provider order and to verify a provider event amount, and
// those two uses must never drift apart: a divergence would let a mismatched
// provider amount pass verification. It is therefore deliberately a single
// implementation shared by the provider and the webhook verification path.
func MinorUnitsToDecimalString(minorUnits int64) string {
	negative := minorUnits < 0
	if negative {
		minorUnits = -minorUnits
	}

	whole := minorUnits / minorUnitsPerMajorUnit
	fraction := minorUnits % minorUnitsPerMajorUnit

	sign := ""
	if negative {
		sign = "-"
	}
	return fmt.Sprintf("%s%d.%02d", sign, whole, fraction)
}

// ValidateCurrencyCode reports whether a purchase currency is a usable ISO 4217
// code. Every provider requires such a code, so an unusable value fails closed
// here — before any external provider call — rather than being forwarded to a
// provider that would reject it or, worse, interpret it differently.
//
// This validates the *shape* of the code only: three letters, in any case,
// since the stored snapshot and each provider use a different case convention
// and the caller normalises it. It deliberately does not restrict RYZE to any
// particular currency and never converts one into another: which currencies
// RYZE accepts is a business decision enforced when a program price is set, and
// a purchase's snapshot currency is always authoritative as written.
func ValidateCurrencyCode(currency string) error {
	code := strings.TrimSpace(currency)
	if len(code) != 3 {
		return fmt.Errorf("currency %q is not a 3-letter ISO 4217 code", currency)
	}
	for _, char := range code {
		if (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') {
			return fmt.Errorf("currency %q is not a 3-letter ISO 4217 code", currency)
		}
	}
	return nil
}
