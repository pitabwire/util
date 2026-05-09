// Package money — ISO 4217 currency precision lookup and strict
// conversion helpers. The conversions in moneyx.go take an explicit
// `decimals` parameter; in many callers the decimals are determined
// solely by the currency code, and a strict check that the wire money
// matches an expected currency avoids silent currency coercion bugs.
package moneyx

import (
	"errors"
	"fmt"
	"strings"

	commonv1 "buf.build/gen/go/antinvestor/common/protocolbuffers/go/common/v1"
)

// ─── ISO 4217 precision lookup ──────────────────────────────────────

// zeroDecimal is the set of ISO 4217 codes whose minor unit equals the
// major unit (no fractional digits).
var zeroDecimal = map[string]struct{}{
	"BIF": {}, "CLP": {}, "DJF": {}, "GNF": {}, "ISK": {}, "JPY": {},
	"KMF": {}, "KRW": {}, "PYG": {}, "RWF": {}, "UGX": {}, "UYI": {},
	"VND": {}, "VUV": {}, "XAF": {}, "XOF": {}, "XPF": {},
}

// threeDecimal is the set of ISO 4217 codes with three fractional digits.
var threeDecimal = map[string]struct{}{
	"BHD": {}, "IQD": {}, "JOD": {}, "KWD": {}, "LYD": {}, "OMR": {}, "TND": {},
}

// Decimals returns the ISO 4217 minor-unit count (0, 2, or 3) for the
// supplied currency code. The lookup is case-insensitive. Unknown or
// empty codes default to 2 — the most common precision and the safest
// fallback for codes outside the published list.
func Decimals(currencyCode string) int32 {
	c := strings.ToUpper(strings.TrimSpace(currencyCode))
	if _, ok := zeroDecimal[c]; ok {
		return 0
	}
	if _, ok := threeDecimal[c]; ok {
		return 3
	}
	return 2
}

// ─── Strict converters ──────────────────────────────────────────────

// ErrCurrencyMismatch is returned when a *commonv1.Money's currency code
// does not match the expected currency. Callers should never silently
// coerce money between currencies; converting an off-currency amount
// without explicit FX is a correctness bug.
var ErrCurrencyMismatch = errors.New("money: currency mismatch")

// ErrSignMismatch is returned when a *commonv1.Money has units and nanos
// with opposite signs, which is invalid per google.type.Money semantics.
var ErrSignMismatch = errors.New("money: units and nanos have opposite signs")

// ErrNilMoney is returned when a nil *commonv1.Money is passed to a strict
// converter that requires a non-nil input.
var ErrNilMoney = errors.New("money: nil input")

// ToSmallestUnitStrict converts a *commonv1.Money to int64 minor units,
// validating that the currency matches expectedCurrency (case-insensitive)
// and that the units/nanos signs agree. It is the strict variant of
// ToSmallestUnit for callers that need to refuse silent coercion.
func ToSmallestUnitStrict(m *commonv1.Money, expectedCurrency string, decimals int32) (int64, error) {
	if m == nil {
		return 0, ErrNilMoney
	}
	if !strings.EqualFold(m.GetCurrencyCode(), expectedCurrency) {
		return 0, fmt.Errorf("%w: got %q want %q", ErrCurrencyMismatch,
			m.GetCurrencyCode(), expectedCurrency)
	}
	if (m.GetUnits() > 0 && m.GetNanos() < 0) || (m.GetUnits() < 0 && m.GetNanos() > 0) {
		return 0, ErrSignMismatch
	}
	return ToSmallestUnit(m, decimals), nil
}

// ToMinorUnitsByCurrency is a shortcut combining Decimals + ToSmallestUnitStrict:
// it looks up the ISO 4217 precision for expectedCurrency, validates currency
// match, and converts. Use it when the caller wants currency-aware precision
// without juggling the decimals argument.
func ToMinorUnitsByCurrency(m *commonv1.Money, expectedCurrency string) (int64, error) {
	return ToSmallestUnitStrict(m, expectedCurrency, Decimals(expectedCurrency))
}

// FromMinorUnitsByCurrency converts an int64 minor-unit amount and a
// currency code to a *commonv1.Money, looking up the ISO 4217 precision
// for the code. The returned message has the supplied currency stamped
// on it; if the currency is unknown the conversion uses 2 decimals (the
// Decimals fallback).
func FromMinorUnitsByCurrency(currency string, minor int64) *commonv1.Money {
	return FromSmallestUnit(currency, minor, Decimals(currency))
}
