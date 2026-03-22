// Package decimalx provides an immutable arbitrary-precision decimal type
// built on cockroachdb/apd, with SQL and JSON support.
package decimalx

import (
	"database/sql/driver"
	"fmt"
	"math"

	"github.com/cockroachdb/apd/v3"
)

// Precision constants.
const (
	// DecimalPrecision is the precision used for decimal calculations.
	DecimalPrecision = 9
	// NanoSize is the multiplier for converting decimal fractions to nano units.
	NanoSize = 1_000_000_000
	// MaxNanosValue is the maximum value for nanos (10^9 - 1).
	MaxNanosValue = 999999999
)

// ctx is the shared arithmetic context: 38 digits, half-up rounding.
var ctx = &apd.Context{
	Precision:   38,
	Rounding:    apd.RoundHalfUp,
	MaxExponent: 5000,
	MinExponent: -5000,
	Traps:       apd.DefaultTraps,
}

// Decimal wraps *apd.Decimal for precise decimal calculations.
type Decimal struct {
	d *apd.Decimal
}

// inner returns the underlying apd.Decimal, never nil.
func (v Decimal) inner() *apd.Decimal {
	if v.d == nil {
		return apd.New(0, 0)
	}
	return v.d
}

// Inner returns the underlying *apd.Decimal for interop with other packages.
func (v Decimal) Inner() *apd.Decimal {
	return v.inner()
}

// Ctx returns the shared arithmetic context.
func Ctx() *apd.Context {
	return ctx
}

// ---------------------------------------------------------------------------
// Constructors
// ---------------------------------------------------------------------------

// New creates a Decimal from a coefficient and exponent (coefficient * 10^exponent).
func New(coeff int64, exponent int32) Decimal {
	return Decimal{d: apd.New(coeff, exponent)}
}

// NewFromInt64 creates a Decimal from an int64 value.
func NewFromInt64(v int64) Decimal {
	return Decimal{d: apd.New(v, 0)}
}

// NewFromString parses a decimal string. Returns an error for invalid input.
func NewFromString(s string) (Decimal, error) {
	d, _, err := apd.NewFromString(s)
	if err != nil {
		return Decimal{}, fmt.Errorf("decimalx: invalid decimal string %q: %w", s, err)
	}
	return Decimal{d: d}, nil
}

// Zero returns a Decimal representing 0.
func Zero() Decimal {
	return Decimal{d: apd.New(0, 0)}
}

// NewFromBasisPoints converts basis points to a decimal fraction.
// For example, 1500 bp becomes 0.15.
func NewFromBasisPoints(bp int64) Decimal {
	result := new(apd.Decimal)
	_, _ = ctx.Quo(result, apd.New(bp, 0), apd.New(10000, 0))
	return Decimal{d: result}
}

// ---------------------------------------------------------------------------
// Arithmetic — all operations return a new Decimal (immutable).
// ---------------------------------------------------------------------------

// Add returns v + other.
func (v Decimal) Add(other Decimal) Decimal {
	result := new(apd.Decimal)
	_, _ = ctx.Add(result, v.inner(), other.inner())
	return Decimal{d: result}
}

// Sub returns v - other.
func (v Decimal) Sub(other Decimal) Decimal {
	result := new(apd.Decimal)
	_, _ = ctx.Sub(result, v.inner(), other.inner())
	return Decimal{d: result}
}

// Mul returns v * other.
func (v Decimal) Mul(other Decimal) Decimal {
	result := new(apd.Decimal)
	_, _ = ctx.Mul(result, v.inner(), other.inner())
	return Decimal{d: result}
}

// Div returns v / other. Panics on division by zero.
func (v Decimal) Div(other Decimal) Decimal {
	result := new(apd.Decimal)
	_, err := ctx.Quo(result, v.inner(), other.inner())
	if err != nil {
		panic(fmt.Sprintf("decimalx: division error: %v", err))
	}
	return Decimal{d: result}
}

// Neg returns -v.
func (v Decimal) Neg() Decimal {
	result := new(apd.Decimal)
	result.Neg(v.inner())
	return Decimal{d: result}
}

// ---------------------------------------------------------------------------
// Comparison
// ---------------------------------------------------------------------------

// Cmp compares v and other: -1 if v < other, 0 if equal, +1 if v > other.
func (v Decimal) Cmp(other Decimal) int {
	return v.inner().Cmp(other.inner())
}

// IsZero returns true when v == 0.
func (v Decimal) IsZero() bool {
	return v.inner().IsZero()
}

// IsNegative returns true when v < 0.
func (v Decimal) IsNegative() bool {
	return v.inner().Sign() < 0
}

// IsPositive returns true when v > 0.
func (v Decimal) IsPositive() bool {
	return v.inner().Sign() > 0
}

// Equal returns true when v == other.
func (v Decimal) Equal(other Decimal) bool {
	return v.inner().Cmp(other.inner()) == 0
}

// LessThan returns true when v < other.
func (v Decimal) LessThan(other Decimal) bool {
	return v.inner().Cmp(other.inner()) < 0
}

// GreaterThan returns true when v > other.
func (v Decimal) GreaterThan(other Decimal) bool {
	return v.inner().Cmp(other.inner()) > 0
}

// ---------------------------------------------------------------------------
// Conversion
// ---------------------------------------------------------------------------

// ToMinorUnits converts the decimal to the smallest currency unit for the
// given precision. For example, 123.45 with precision 2 returns 12345.
func (v Decimal) ToMinorUnits(precision int32) int64 {
	multiplier := apd.New(1, precision)
	result := new(apd.Decimal)
	_, _ = ctx.Mul(result, v.inner(), multiplier)

	// Quantize to 0 decimal places (round).
	rounded := new(apd.Decimal)
	_, _ = ctx.Quantize(rounded, result, 0)

	i64, err := rounded.Int64()
	if err != nil {
		return 0
	}
	return i64
}

// String returns the decimal as a plain string.
func (v Decimal) String() string {
	return v.inner().Text('f')
}

// Int64 truncates the decimal and returns the integer part.
func (v Decimal) Int64() int64 {
	truncated := new(apd.Decimal)
	_, _ = ctx.Quantize(truncated, v.inner(), 0)
	i64, err := truncated.Int64()
	if err != nil {
		return 0
	}
	return i64
}

// ---------------------------------------------------------------------------
// Basis-points helpers
// ---------------------------------------------------------------------------

// ApplyBasisPoints computes amount * bp / 10000.
func ApplyBasisPoints(amount Decimal, bp int64) Decimal {
	bpDec := apd.New(bp, 0)
	divisor := apd.New(10000, 0)

	tmp := new(apd.Decimal)
	_, _ = ctx.Mul(tmp, amount.inner(), bpDec)

	result := new(apd.Decimal)
	_, _ = ctx.Quo(result, tmp, divisor)
	return Decimal{d: result}
}

// ---------------------------------------------------------------------------
// MinorUnits helpers
// ---------------------------------------------------------------------------

// FromMinorUnits converts a minor-unit integer to a Decimal.
// For example, 12345 with precision 2 returns 123.45.
func FromMinorUnits(amount int64, precision int32) Decimal {
	return Decimal{d: apd.New(amount, -precision)}
}

// ---------------------------------------------------------------------------
// Max value helper (for CleanDecimal / clamping)
// ---------------------------------------------------------------------------

// GetMaxDecimalValue returns the maximum decimal value supported,
// matching the payment service's NUMERIC(29,9) column.
func GetMaxDecimalValue() Decimal {
	maxUnits := apd.New(math.MaxInt64, 0)
	maxNanos := apd.New(MaxNanosValue, -9)
	result := new(apd.Decimal)
	_, _ = ctx.Add(result, maxUnits, maxNanos)
	return Decimal{d: result}
}

// ---------------------------------------------------------------------------
// database/sql.Scanner & driver.Valuer — stores as NUMERIC in PostgreSQL.
// ---------------------------------------------------------------------------

// Scan implements database/sql.Scanner.
func (v *Decimal) Scan(src interface{}) error {
	if src == nil {
		v.d = apd.New(0, 0)
		return nil
	}
	var s string
	switch val := src.(type) {
	case string:
		s = val
	case []byte:
		s = string(val)
	default:
		return fmt.Errorf("decimalx: cannot scan %T into Decimal", src)
	}
	d, _, err := apd.NewFromString(s)
	if err != nil {
		return fmt.Errorf("decimalx: scan error: %w", err)
	}
	v.d = d
	return nil
}

// Value implements database/sql/driver.Valuer.
func (v Decimal) Value() (driver.Value, error) {
	return v.String(), nil
}

// ---------------------------------------------------------------------------
// json.Marshaler & json.Unmarshaler — serializes as a quoted string.
// ---------------------------------------------------------------------------

// MarshalJSON implements json.Marshaler.
func (v Decimal) MarshalJSON() ([]byte, error) {
	return []byte(`"` + v.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (v *Decimal) UnmarshalJSON(data []byte) error {
	// Strip surrounding quotes.
	s := string(data)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	d, _, err := apd.NewFromString(s)
	if err != nil {
		return fmt.Errorf("decimalx: unmarshal error: %w", err)
	}
	v.d = d
	return nil
}
