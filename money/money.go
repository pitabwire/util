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
