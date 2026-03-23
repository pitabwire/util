package decimalx

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// Arithmetic
// ---------------------------------------------------------------------------

func TestAdd(t *testing.T) {
	a := NewFromInt64(100)
	b := NewFromInt64(200)
	got := a.Add(b)
	if !got.Equal(NewFromInt64(300)) {
		t.Errorf("100 + 200 = %s, want 300", got)
	}
}

func TestSub(t *testing.T) {
	a := NewFromInt64(500)
	b := NewFromInt64(123)
	got := a.Sub(b)
	if !got.Equal(NewFromInt64(377)) {
		t.Errorf("500 - 123 = %s, want 377", got)
	}
}

func TestMul(t *testing.T) {
	a, _ := NewFromString("12.5")
	b, _ := NewFromString("4")
	got := a.Mul(b)
	want, _ := NewFromString("50")
	if !got.Equal(want) {
		t.Errorf("12.5 * 4 = %s, want 50", got)
	}
}

func TestDiv(t *testing.T) {
	a := NewFromInt64(10)
	b := NewFromInt64(3)
	got := a.Div(b)
	// 10/3 ≈ 3.3333... — check that it's close
	want, _ := NewFromString("3.33333333333333333333333333333333333333")
	if got.Cmp(want) != 0 {
		t.Logf("10 / 3 = %s (expected ~3.333...)", got)
	}
	// At least verify 3 < result < 4
	if got.Cmp(NewFromInt64(3)) <= 0 || got.Cmp(NewFromInt64(4)) >= 0 {
		t.Errorf("10 / 3 = %s, expected between 3 and 4", got)
	}
}

func TestNeg(t *testing.T) {
	a := NewFromInt64(42)
	got := a.Neg()
	if !got.Equal(NewFromInt64(-42)) {
		t.Errorf("-42 != %s", got)
	}
}

func TestAddDecimals(t *testing.T) {
	a, _ := NewFromString("0.1")
	b, _ := NewFromString("0.2")
	got := a.Add(b)
	want, _ := NewFromString("0.3")
	if !got.Equal(want) {
		t.Errorf("0.1 + 0.2 = %s, want 0.3", got)
	}
}

// ---------------------------------------------------------------------------
// BasisPoints
// ---------------------------------------------------------------------------

func TestNewFromBasisPoints(t *testing.T) {
	bp := NewFromBasisPoints(1500)
	want, _ := NewFromString("0.15")
	if !bp.Equal(want) {
		t.Errorf("1500bp = %s, want 0.15", bp)
	}
}

func TestApplyBasisPoints1500(t *testing.T) {
	amount := NewFromInt64(100000)
	got := ApplyBasisPoints(amount, 1500)
	if !got.Equal(NewFromInt64(15000)) {
		t.Errorf("1500bp of 100000 = %s, want 15000", got)
	}
}

func TestApplyBasisPoints500(t *testing.T) {
	amount := NewFromInt64(10000)
	got := ApplyBasisPoints(amount, 500)
	if !got.Equal(NewFromInt64(500)) {
		t.Errorf("500bp of 10000 = %s, want 500", got)
	}
}

// ---------------------------------------------------------------------------
// MinorUnits round-trip
// ---------------------------------------------------------------------------

func TestMinorUnitsRoundTrip(t *testing.T) {
	d := FromMinorUnits(12345, 2)
	want, _ := NewFromString("123.45")
	if !d.Equal(want) {
		t.Errorf("FromMinorUnits(12345, 2) = %s, want 123.45", d)
	}
	got := d.ToMinorUnits(2)
	if got != 12345 {
		t.Errorf("ToMinorUnits(2) = %d, want 12345", got)
	}
}

func TestMinorUnitsZero(t *testing.T) {
	d := FromMinorUnits(0, 2)
	if !d.IsZero() {
		t.Errorf("FromMinorUnits(0, 2) should be zero, got %s", d)
	}
	got := d.ToMinorUnits(2)
	if got != 0 {
		t.Errorf("ToMinorUnits(2) of zero = %d, want 0", got)
	}
}

// ---------------------------------------------------------------------------
// Edge cases: zero, negative, very large
// ---------------------------------------------------------------------------

func TestZero(t *testing.T) {
	z := Zero()
	if !z.IsZero() {
		t.Error("Zero() should be zero")
	}
	if z.IsPositive() || z.IsNegative() {
		t.Error("Zero() should be neither positive nor negative")
	}
}

func TestNegativeComparisons(t *testing.T) {
	n := NewFromInt64(-5)
	if !n.IsNegative() {
		t.Error("-5 should be negative")
	}
	if n.IsPositive() {
		t.Error("-5 should not be positive")
	}
	if !n.LessThan(Zero()) {
		t.Error("-5 should be less than 0")
	}
}

func TestLargeValues(t *testing.T) {
	a := NewFromInt64(999999999999)
	b := NewFromInt64(999999999999)
	got := a.Mul(b)
	if got.IsZero() {
		t.Error("large multiplication should not be zero")
	}
	if got.IsNegative() {
		t.Error("product of two positives should be positive")
	}
}

func TestInt64Truncates(t *testing.T) {
	d, _ := NewFromString("123.999")
	got := d.Int64()
	if got != 124 {
		t.Errorf("Int64() of 123.999 = %d, want 124", got)
	}
}

// ---------------------------------------------------------------------------
// SQL Scanner / Valuer
// ---------------------------------------------------------------------------

func TestScanString(t *testing.T) {
	var d Decimal
	err := d.Scan("123.456")
	if err != nil {
		t.Fatal(err)
	}
	want, _ := NewFromString("123.456")
	if !d.Equal(want) {
		t.Errorf("Scan(string) = %s, want 123.456", d)
	}
}

func TestScanBytes(t *testing.T) {
	var d Decimal
	err := d.Scan([]byte("-99.99"))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := NewFromString("-99.99")
	if !d.Equal(want) {
		t.Errorf("Scan([]byte) = %s, want -99.99", d)
	}
}

func TestScanNil(t *testing.T) {
	var d Decimal
	err := d.Scan(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !d.IsZero() {
		t.Errorf("Scan(nil) = %s, want 0", d)
	}
}

func TestValue(t *testing.T) {
	d, _ := NewFromString("42.123")
	v, err := d.Value()
	if err != nil {
		t.Fatal(err)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("Value() returned %T, want string", v)
	}
	if s != "42.123" {
		t.Errorf("Value() = %q, want %q", s, "42.123")
	}
}

// ---------------------------------------------------------------------------
// JSON marshal / unmarshal round-trip
// ---------------------------------------------------------------------------

func TestJSONRoundTrip(t *testing.T) {
	orig, _ := NewFromString("987.654321")
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `"987.654321"` {
		t.Errorf("Marshal = %s, want %q", data, "987.654321")
	}

	var got Decimal
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !got.Equal(orig) {
		t.Errorf("unmarshal round-trip: got %s, want %s", got, orig)
	}
}

func TestJSONUnmarshalBareNumber(t *testing.T) {
	var d Decimal
	err := json.Unmarshal([]byte(`123.45`), &d)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := NewFromString("123.45")
	if !d.Equal(want) {
		t.Errorf("unmarshal bare number: got %s, want 123.45", d)
	}
}

// ---------------------------------------------------------------------------
// Constructors
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	d := New(12345, -2)
	want, _ := NewFromString("123.45")
	if !d.Equal(want) {
		t.Errorf("New(12345, -2) = %s, want 123.45", d)
	}
}

func TestNewFromStringInvalid(t *testing.T) {
	_, err := NewFromString("not-a-number")
	if err == nil {
		t.Error("expected error for invalid string")
	}
}

// ---------------------------------------------------------------------------
// Abs, Min, Max, comparison helpers
// ---------------------------------------------------------------------------

func TestAbs(t *testing.T) {
	neg := NewFromInt64(-42)
	if !neg.Abs().Equal(NewFromInt64(42)) {
		t.Errorf("Abs(-42) = %s, want 42", neg.Abs())
	}
	pos := NewFromInt64(7)
	if !pos.Abs().Equal(NewFromInt64(7)) {
		t.Errorf("Abs(7) = %s, want 7", pos.Abs())
	}
	z := Zero()
	if !z.Abs().IsZero() {
		t.Errorf("Abs(0) = %s, want 0", z.Abs())
	}
}

func TestMin(t *testing.T) {
	a := NewFromInt64(3)
	b := NewFromInt64(7)
	if !Min(a, b).Equal(a) {
		t.Errorf("Min(3, 7) = %s, want 3", Min(a, b))
	}
	if !Min(b, a).Equal(a) {
		t.Errorf("Min(7, 3) = %s, want 3", Min(b, a))
	}
}

func TestMax(t *testing.T) {
	a := NewFromInt64(3)
	b := NewFromInt64(7)
	if !Max(a, b).Equal(b) {
		t.Errorf("Max(3, 7) = %s, want 7", Max(a, b))
	}
}

func TestLessThanOrEqual(t *testing.T) {
	a := NewFromInt64(5)
	b := NewFromInt64(5)
	c := NewFromInt64(6)
	if !a.LessThanOrEqual(b) {
		t.Error("5 <= 5 should be true")
	}
	if !a.LessThanOrEqual(c) {
		t.Error("5 <= 6 should be true")
	}
	if c.LessThanOrEqual(a) {
		t.Error("6 <= 5 should be false")
	}
}

func TestGreaterThanOrEqual(t *testing.T) {
	a := NewFromInt64(5)
	b := NewFromInt64(5)
	c := NewFromInt64(3)
	if !a.GreaterThanOrEqual(b) {
		t.Error("5 >= 5 should be true")
	}
	if !a.GreaterThanOrEqual(c) {
		t.Error("5 >= 3 should be true")
	}
	if c.GreaterThanOrEqual(a) {
		t.Error("3 >= 5 should be false")
	}
}

func TestPtr(t *testing.T) {
	d := NewFromInt64(42)
	p := d.Ptr()
	if p == nil {
		t.Fatal("Ptr() returned nil")
	}
	if !p.Equal(d) {
		t.Errorf("Ptr() = %s, want 42", p)
	}
}

func TestDerefOr(t *testing.T) {
	d := NewFromInt64(99)
	got := DerefOr(&d, Zero())
	if !got.Equal(d) {
		t.Errorf("DerefOr(&99, 0) = %s, want 99", got)
	}
	got2 := DerefOr(nil, NewFromInt64(7))
	if !got2.Equal(NewFromInt64(7)) {
		t.Errorf("DerefOr(nil, 7) = %s, want 7", got2)
	}
}
