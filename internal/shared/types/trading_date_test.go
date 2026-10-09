package types

import (
	"math"
	"testing"
	"time"
)

func TestTradingDateRoundTripsCanonicalAndTushareFormats(t *testing.T) {
	date, err := ParseTradingDate("2026-10-08")
	if err != nil {
		t.Fatalf("ParseTradingDate() error = %v", err)
	}
	if got := date.String(); got != "2026-10-08" {
		t.Fatalf("String() = %q, want 2026-10-08", got)
	}
	tushareDate, err := date.TushareString()
	if err != nil {
		t.Fatalf("TushareString() error = %v", err)
	}
	if got := tushareDate; got != "20261008" {
		t.Fatalf("TushareString() = %q, want 20261008", got)
	}
	parsed, err := ParseTushareDate(tushareDate)
	if err != nil {
		t.Fatalf("ParseTushareDate() error = %v", err)
	}
	if parsed != date {
		t.Fatalf("ParseTushareDate() = %v, want %v", parsed, date)
	}
}

func TestZeroTradingDateIsExplicitlyInvalid(t *testing.T) {
	var date TradingDate
	if date.Valid() {
		t.Fatal("zero TradingDate.Valid() = true")
	}
	if got := date.String(); got != "<invalid-trading-date>" {
		t.Fatalf("zero TradingDate.String() = %q, want explicit invalid marker", got)
	}
	if _, err := date.DatabaseString(); err == nil {
		t.Fatal("zero TradingDate.DatabaseString() error = nil")
	}
	if _, err := date.TushareString(); err == nil {
		t.Fatal("zero TradingDate.TushareString() error = nil")
	}
}

func TestParseTradingDateRejectsInvalidRepresentations(t *testing.T) {
	for _, value := range []string{"", "2026-02-30", "2026-2-03", "2026-10-08x", "0000-01-01"} {
		t.Run(value, func(t *testing.T) {
			if _, err := ParseTradingDate(value); err == nil {
				t.Fatalf("ParseTradingDate(%q) error = nil", value)
			}
		})
	}
}

func TestParseTushareDateRejectsInvalidRepresentations(t *testing.T) {
	for _, value := range []string{"", "20260230", "2026-10-08", "20260008", "00000101"} {
		t.Run(value, func(t *testing.T) {
			if _, err := ParseTushareDate(value); err == nil {
				t.Fatalf("ParseTushareDate(%q) error = nil", value)
			}
		})
	}
}

func TestTradingDateFromTimeUsesExplicitLocation(t *testing.T) {
	shanghai := time.FixedZone("UTC+08", 8*60*60)
	localMidnight := time.Date(2026, time.October, 8, 0, 0, 0, 0, shanghai)
	date, err := TradingDateFromTime(localMidnight.UTC(), shanghai)
	if err != nil {
		t.Fatalf("TradingDateFromTime() error = %v", err)
	}
	if got := date.String(); got != "2026-10-08" {
		t.Fatalf("TradingDateFromTime() = %q, want Shanghai trading date 2026-10-08", got)
	}
}

func TestTradingDateFromTimeRequiresLocation(t *testing.T) {
	instant := time.Date(2026, time.October, 8, 0, 0, 0, 0, time.UTC)
	if _, err := TradingDateFromTime(instant, nil); err == nil {
		t.Fatal("TradingDateFromTime() error = nil for missing location")
	}
}

func TestAmountYuanAcceptsFiniteValuesWithoutRounding(t *testing.T) {
	for _, value := range []float64{0, -1.25, 30_000_000.1234} {
		amount, err := NewAmountYuan(value)
		if err != nil {
			t.Fatalf("NewAmountYuan(%v) error = %v", value, err)
		}
		if amount.Float64() != value {
			t.Fatalf("NewAmountYuan(%v) = %v, want unchanged value", value, amount.Float64())
		}
	}
}

func TestAmountYuanRejectsNonFiniteValues(t *testing.T) {
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := NewAmountYuan(value); err == nil {
			t.Fatalf("NewAmountYuan(%v) error = nil", value)
		}
	}
}
