//nolint:testpackage // tests exercise unexported helpers
package moneyx

import (
	"testing"

	commonv1 "buf.build/gen/go/antinvestor/common/protocolbuffers/go/common/v1"
	"github.com/pitabwire/util/decimalx"
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
				t.Errorf(
					"round-trip failed: %s -> Money{%d, %d} -> %s",
					orig,
					m.GetUnits(),
					m.GetNanos(),
					back,
				)
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
	a := &commonv1.Money{CurrencyCode: "USD", Units: 10, Nanos: 500000000}
	b := &commonv1.Money{CurrencyCode: "USD", Units: 10, Nanos: 600000000}
	c := &commonv1.Money{CurrencyCode: "USD", Units: 10, Nanos: 500000000}

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
		m    *commonv1.Money
		want float64
	}{
		{"nil", nil, 0},
		{"whole", &commonv1.Money{Units: 100}, 100.0},
		{"fractional", &commonv1.Money{Units: 42, Nanos: 500000000}, 42.5},
		{"negative", &commonv1.Money{Units: -10, Nanos: -250000000}, -10.25},
		{"cents", &commonv1.Money{Units: 0, Nanos: 10000000}, 0.01},
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
			if got.GetUnits() != tt.wantUnit {
				t.Errorf("Units = %d, want %d", got.GetUnits(), tt.wantUnit)
			}
			if got.GetNanos() != tt.wantNano {
				t.Errorf("Nanos = %d, want %d", got.GetNanos(), tt.wantNano)
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

// ---------------------------------------------------------------------------
// Smallest-unit / Int64 conversions
// ---------------------------------------------------------------------------

func TestSmallestUnitRoundTrip_Cents(t *testing.T) {
	// 10.50 USD -> 1050 cents -> back to 10.50 USD
	m := &commonv1.Money{CurrencyCode: "USD", Units: 10, Nanos: 500000000}
	cents := ToSmallestUnit(m, 2)
	if cents != 1050 {
		t.Errorf("ToSmallestUnit(10.50 USD, 2) = %d, want 1050", cents)
	}
	back := FromSmallestUnit("USD", cents, 2)
	if back.GetUnits() != 10 || back.GetNanos() != 500000000 {
		t.Errorf(
			"FromSmallestUnit(1050, 2) = {%d, %d}, want {10, 500000000}",
			back.GetUnits(),
			back.GetNanos(),
		)
	}
}

func TestSmallestUnitRoundTrip_Satoshi(t *testing.T) {
	// 1.5 BTC -> 150000000 satoshi -> back
	m := &commonv1.Money{CurrencyCode: "BTC", Units: 1, Nanos: 500000000}
	sats := ToSmallestUnit(m, 8)
	if sats != 150000000 {
		t.Errorf("ToSmallestUnit(1.5 BTC, 8) = %d, want 150000000", sats)
	}
	back := FromSmallestUnit("BTC", sats, 8)
	if back.GetUnits() != 1 || back.GetNanos() != 500000000 {
		t.Errorf(
			"FromSmallestUnit(150000000, 8) = {%d, %d}, want {1, 500000000}",
			back.GetUnits(),
			back.GetNanos(),
		)
	}
}

func TestSmallestUnitDecimal_Wei(t *testing.T) {
	// 1.5 ETH in wei = 1500000000000000000
	m := &commonv1.Money{CurrencyCode: "ETH", Units: 1, Nanos: 500000000}
	wei := ToSmallestUnitDecimal(m, 18)
	want, _ := decimalx.NewFromString("1500000000000000000")
	if !wei.Equal(want) {
		t.Errorf("ToSmallestUnitDecimal(1.5 ETH, 18) = %s, want 1500000000000000000", wei)
	}

	// Round-trip back
	back := FromSmallestUnitDecimal("ETH", wei, 18)
	if back.GetUnits() != 1 || back.GetNanos() != 500000000 {
		t.Errorf(
			"FromSmallestUnitDecimal round-trip: {%d, %d}, want {1, 500000000}",
			back.GetUnits(),
			back.GetNanos(),
		)
	}
}

func TestFromInt64_Alias(t *testing.T) {
	m := FromInt64("KES", 15050, 2)
	if m.GetUnits() != 150 || m.GetNanos() != 500000000 {
		t.Errorf("FromInt64(KES, 15050, 2) = {%d, %d}, want {150, 500000000}", m.GetUnits(), m.GetNanos())
	}
}

func TestToInt64_Alias(t *testing.T) {
	m := &commonv1.Money{CurrencyCode: "KES", Units: 150, Nanos: 500000000}
	got := ToInt64(m, 2)
	if got != 15050 {
		t.Errorf("ToInt64(150.50 KES, 2) = %d, want 15050", got)
	}
}
