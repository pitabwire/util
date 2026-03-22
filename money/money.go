// Package money provides conversions between google.type.Money protobuf
// messages and decimalx.Decimal values.
package money

import (
	"fmt"
	"math"

	"github.com/cockroachdb/apd/v3"
	"github.com/pitabwire/util/decimalx"
	"google.golang.org/genproto/googleapis/type/money"
)

// CentsPerUnit is the number of cents in one currency unit.
const CentsPerUnit = 100

// NanosPerCent is the number of nanos in one cent.
const NanosPerCent = 10_000_000

// ToMoney converts a Decimal to a google.type.Money protobuf message.
// Units holds the integer part; Nanos holds the fractional part scaled to 10^9.
func ToMoney(currency string, amount decimalx.Decimal) *money.Money {
	a := amount.Inner()
	ctx := decimalx.Ctx()

	// Quantize to DecimalPrecision digits after the decimal point.
	cleaned := new(apd.Decimal)
	_, _ = ctx.Quantize(cleaned, a, -decimalx.DecimalPrecision)

	// Extract the integer part (units) by truncating toward zero.
	truncCtx := *ctx
	truncCtx.Rounding = apd.RoundDown
	units := new(apd.Decimal)
	_, _ = truncCtx.Quantize(units, cleaned, 0)

	// fractional = cleaned - units
	frac := new(apd.Decimal)
	_, _ = ctx.Sub(frac, cleaned, units)

	// nanos = fractional * NanoSize
	nanosD := new(apd.Decimal)
	_, _ = ctx.Mul(nanosD, frac, apd.New(decimalx.NanoSize, 0))

	// Round nanos to integer.
	nanosRounded := new(apd.Decimal)
	_, _ = ctx.Quantize(nanosRounded, nanosD, 0)

	unitsI64, _ := units.Int64()
	nanosI64, _ := nanosRounded.Int64()

	// Clamp nanos to int32 range.
	if nanosI64 > math.MaxInt32 {
		nanosI64 = math.MaxInt32
	} else if nanosI64 < math.MinInt32 {
		nanosI64 = math.MinInt32
	}

	return &money.Money{
		CurrencyCode: currency,
		Units:        unitsI64,
		Nanos:        int32(nanosI64),
	}
}

// FromMoney converts a google.type.Money protobuf message back to a Decimal.
func FromMoney(m *money.Money) decimalx.Decimal {
	if m == nil {
		return decimalx.Zero()
	}
	units := decimalx.NewFromInt64(m.GetUnits())
	nanos := decimalx.New(int64(m.GetNanos()), -9)
	return units.Add(nanos)
}

// CompareMoney compares two Money values numerically, returning -1, 0, or 1.
func CompareMoney(a, b *money.Money) int {
	da := FromMoney(a)
	db := FromMoney(b)
	return da.Cmp(db)
}

// ToFloat64 converts a google.type.Money to a float64.
func ToFloat64(m *money.Money) float64 {
	if m == nil {
		return 0
	}
	return float64(m.GetUnits()) + float64(m.GetNanos())/float64(decimalx.NanoSize)
}

// FromFloat64 converts a float64 amount and currency code into a google.type.Money.
func FromFloat64(currency string, amount float64) *money.Money {
	d, err := decimalx.NewFromString(fmt.Sprintf("%.9f", amount))
	if err != nil {
		d = decimalx.Zero()
	}
	return ToMoney(currency, d)
}

// ToCents converts units and nanos to the smallest currency unit (cents).
func ToCents(units int64, nanos int32) int64 {
	return units*CentsPerUnit + int64(nanos/NanosPerCent)
}

// ---------------------------------------------------------------------------
// Smallest-unit (Int64) conversions — for currencies where amounts are
// tracked as indivisible integers in their smallest denomination.
// Examples: ETH wei (decimals=18), BTC satoshi (decimals=8),
// USD cents (decimals=2).
// ---------------------------------------------------------------------------

// ToSmallestUnit converts a google.type.Money to its smallest unit
// representation given the number of decimal places for the currency.
// For example, 1.5 ETH with decimals=18 returns 1500000000000000000 (wei).
func ToSmallestUnit(m *money.Money, decimals int32) int64 {
	d := FromMoney(m)
	return d.ToMinorUnits(decimals)
}

// FromSmallestUnit converts a smallest-unit integer back to a
// google.type.Money. For example, 1500000000000000000 wei with
// decimals=18 and currency "ETH" returns Money{Units: 1, Nanos: 500000000}.
func FromSmallestUnit(currency string, amount int64, decimals int32) *money.Money {
	d := decimalx.FromMinorUnits(amount, decimals)
	return ToMoney(currency, d)
}

// ToSmallestUnitDecimal converts a google.type.Money to a Decimal
// representing the amount in the smallest unit. This avoids int64 overflow
// for very large values (e.g. wei amounts exceeding MaxInt64).
func ToSmallestUnitDecimal(m *money.Money, decimals int32) decimalx.Decimal {
	d := FromMoney(m)
	multiplier := decimalx.New(1, decimals)
	return d.Mul(multiplier)
}

// FromSmallestUnitDecimal converts a Decimal in the smallest unit back to
// a google.type.Money. Use this when the smallest-unit value may exceed
// int64 range.
func FromSmallestUnitDecimal(currency string, amount decimalx.Decimal, decimals int32) *money.Money {
	divisor := decimalx.New(1, decimals)
	d := amount.Div(divisor)
	return ToMoney(currency, d)
}

// FromInt64 converts an int64 amount in the smallest unit to a
// google.type.Money. Shorthand for FromSmallestUnit.
func FromInt64(currency string, amount int64, decimals int32) *money.Money {
	return FromSmallestUnit(currency, amount, decimals)
}

// ToInt64 converts a google.type.Money to an int64 in the smallest unit.
// Shorthand for ToSmallestUnit.
func ToInt64(m *money.Money, decimals int32) int64 {
	return ToSmallestUnit(m, decimals)
}
