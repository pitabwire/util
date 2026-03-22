package money

import (
	"testing"

	"github.com/pitabwire/util/decimalx"
	"google.golang.org/genproto/googleapis/type/money"
)

func TestMoneyRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		val  string
	}{
		{"positive", "123.456789000"},
		{"negative", "-99.100000000"},
		{"zero", "0"},
		{"large", "9999999999.999999999"},
		{"small_fraction", "0.000000001"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			orig, err := decimalx.NewFromString(tc.val)
			if err != nil {
				t.Fatal(err)
			}
			m := ToMoney("KES", orig)
			back := FromMoney(m)
			if !orig.Equal(back) {
				t.Errorf("round-trip failed: %s -> Money{%d, %d} -> %s", orig, m.GetUnits(), m.GetNanos(), back)
			}
		})
	}
}

func TestFromMoneyNil(t *testing.T) {
	d := FromMoney(nil)
	if !d.IsZero() {
		t.Errorf("FromMoney(nil) = %s, want 0", d)
	}
}

func TestCompareMoney(t *testing.T) {
	a := &money.Money{CurrencyCode: "USD", Units: 10, Nanos: 500000000}
	b := &money.Money{CurrencyCode: "USD", Units: 10, Nanos: 600000000}
	c := &money.Money{CurrencyCode: "USD", Units: 10, Nanos: 500000000}

	if CompareMoney(a, b) != -1 {
		t.Error("expected a < b")
	}
	if CompareMoney(b, a) != 1 {
		t.Error("expected b > a")
	}
	if CompareMoney(a, c) != 0 {
		t.Error("expected a == c")
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name string
		m    *money.Money
		want float64
	}{
		{"nil", nil, 0},
		{"whole", &money.Money{Units: 100}, 100.0},
		{"fractional", &money.Money{Units: 42, Nanos: 500000000}, 42.5},
		{"negative", &money.Money{Units: -10, Nanos: -250000000}, -10.25},
		{"cents", &money.Money{Units: 0, Nanos: 10000000}, 0.01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToFloat64(tt.m)
			if got != tt.want {
				t.Errorf("ToFloat64() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestFromFloat64(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		wantUnit int64
		wantNano int32
	}{
		{"whole", 100.0, 100, 0},
		{"fractional", 42.5, 42, 500000000},
		{"negative", -10.25, -10, -250000000},
		{"cents", 0.01, 0, 10000000},
		{"zero", 0.0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromFloat64("USD", tt.amount)
			if got.Units != tt.wantUnit {
				t.Errorf("Units = %d, want %d", got.Units, tt.wantUnit)
			}
			if got.Nanos != tt.wantNano {
				t.Errorf("Nanos = %d, want %d", got.Nanos, tt.wantNano)
			}
		})
	}
}

func TestToCents(t *testing.T) {
	tests := []struct {
		name  string
		units int64
		nanos int32
		want  int64
	}{
		{"10 dollars", 10, 0, 1000},
		{"10.50", 10, 500000000, 1050},
		{"0.99", 0, 990000000, 99},
		{"0.01", 0, 10000000, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToCents(tt.units, tt.nanos)
			if got != tt.want {
				t.Errorf("ToCents(%d, %d) = %d, want %d", tt.units, tt.nanos, got, tt.want)
			}
		})
	}
}
