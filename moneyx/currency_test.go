package moneyx

import (
	"errors"
	"strconv"
	"testing"

	commonv1 "buf.build/gen/go/antinvestor/common/protocolbuffers/go/common/v1"
)

func TestDecimals(t *testing.T) {
	tests := []struct {
		code string
		want int32
	}{
		// Two-decimal (default + named)
		{"USD", 2}, {"KES", 2}, {"EUR", 2}, {"GBP", 2},
		// Zero-decimal
		{"JPY", 0}, {"KRW", 0}, {"BIF", 0}, {"VND", 0}, {"XOF", 0},
		// Three-decimal
		{"KWD", 3}, {"BHD", 3}, {"OMR", 3}, {"JOD", 3}, {"TND", 3},
		// Case-insensitivity
		{"usd", 2}, {"jpy", 0}, {"kwd", 3},
		// Whitespace tolerance
		{" KES ", 2}, {"\tJPY\n", 0},
		// Empty / unknown default to 2
		{"", 2}, {"XXX", 2}, {"ZZZ", 2},
	}
	for _, tc := range tests {
		t.Run(tc.code, func(t *testing.T) {
			got := Decimals(tc.code)
			if got != tc.want {
				t.Errorf("Decimals(%q) = %d, want %d", tc.code, got, tc.want)
			}
		})
	}
}

func TestToSmallestUnitStrict_Success(t *testing.T) {
	m := &commonv1.Money{CurrencyCode: "USD", Units: 12, Nanos: 340_000_000}
	got, err := ToSmallestUnitStrict(m, "USD", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 1234 {
		t.Errorf("ToSmallestUnitStrict = %d, want 1234", got)
	}
}

func TestToSmallestUnitStrict_CaseInsensitiveCurrency(t *testing.T) {
	m := &commonv1.Money{CurrencyCode: "usd", Units: 1, Nanos: 0}
	got, err := ToSmallestUnitStrict(m, "USD", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 100 {
		t.Errorf("got %d want 100", got)
	}
}

func TestToSmallestUnitStrict_CurrencyMismatch(t *testing.T) {
	m := &commonv1.Money{CurrencyCode: "USD", Units: 1}
	_, err := ToSmallestUnitStrict(m, "KES", 2)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrCurrencyMismatch) {
		t.Errorf("expected ErrCurrencyMismatch, got %v", err)
	}
}

func TestToSmallestUnitStrict_NilMoney(t *testing.T) {
	_, err := ToSmallestUnitStrict(nil, "USD", 2)
	if !errors.Is(err, ErrNilMoney) {
		t.Errorf("expected ErrNilMoney, got %v", err)
	}
}

func TestToSmallestUnitStrict_SignMismatch(t *testing.T) {
	tests := []struct {
		name string
		m    *commonv1.Money
	}{
		{"positive units, negative nanos", &commonv1.Money{CurrencyCode: "USD", Units: 1, Nanos: -100}},
		{"negative units, positive nanos", &commonv1.Money{CurrencyCode: "USD", Units: -1, Nanos: 100}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ToSmallestUnitStrict(tc.m, "USD", 2)
			if !errors.Is(err, ErrSignMismatch) {
				t.Errorf("expected ErrSignMismatch, got %v", err)
			}
		})
	}
}

func TestToMinorUnitsByCurrency(t *testing.T) {
	tests := []struct {
		name             string
		m                *commonv1.Money
		expectedCurrency string
		want             int64
		wantErr          bool
	}{
		{"USD 12.34", &commonv1.Money{CurrencyCode: "USD", Units: 12, Nanos: 340_000_000}, "USD", 1234, false},
		{"JPY 500 (zero decimals)", &commonv1.Money{CurrencyCode: "JPY", Units: 500, Nanos: 0}, "JPY", 500, false},
		{"KWD 1.234 (three decimals)", &commonv1.Money{CurrencyCode: "KWD", Units: 1, Nanos: 234_000_000}, "KWD", 1234, false},
		{"USD vs KES mismatch", &commonv1.Money{CurrencyCode: "USD", Units: 1}, "KES", 0, true},
		{"nil money", nil, "USD", 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ToMinorUnitsByCurrency(tc.m, tc.expectedCurrency)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %d want %d", got, tc.want)
			}
		})
	}
}

func TestFromMinorUnitsByCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		minor    int64
		units    int64
		nanos    int32
	}{
		{"USD 12.34", "USD", 1234, 12, 340_000_000},
		{"JPY 500", "JPY", 500, 500, 0},
		{"KWD 1.234", "KWD", 1234, 1, 234_000_000},
		{"unknown defaults to 2", "ZZZ", 1234, 12, 340_000_000},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := FromMinorUnitsByCurrency(tc.currency, tc.minor)
			if m.GetCurrencyCode() != tc.currency {
				t.Errorf("currency: got %q want %q", m.GetCurrencyCode(), tc.currency)
			}
			if m.GetUnits() != tc.units {
				t.Errorf("units: got %d want %d", m.GetUnits(), tc.units)
			}
			if m.GetNanos() != tc.nanos {
				t.Errorf("nanos: got %d want %d", m.GetNanos(), tc.nanos)
			}
		})
	}
}

func TestRoundTripByCurrency(t *testing.T) {
	for _, code := range []string{"USD", "KES", "JPY", "KWD"} {
		for _, minor := range []int64{-12345, 0, 1, 99, 100, 12345, 1_000_000_000} {
			t.Run(code+"/"+strconv.FormatInt(minor, 10), func(t *testing.T) {
				m := FromMinorUnitsByCurrency(code, minor)
				back, err := ToMinorUnitsByCurrency(m, code)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if back != minor {
					t.Errorf("round-trip: in=%d, out=%d (Money{%d,%d})", minor, back, m.GetUnits(), m.GetNanos())
				}
			})
		}
	}
}
