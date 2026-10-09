package types

import (
	"encoding/json"
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
		got, err := amount.Float64Checked()
		if err != nil || got != value {
			t.Fatalf("NewAmountYuan(%v) = %v, %v; want unchanged finite value", value, got, err)
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

func TestDecimalCanonicalizesAndPreservesArbitraryPrecision(t *testing.T) {
	value, err := ParseDecimal("9007199254740993.0000")
	if err != nil {
		t.Fatalf("ParseDecimal() error = %v", err)
	}
	if got := value.String(); got != "9007199254740993" {
		t.Fatalf("String() = %q, want canonical exact value", got)
	}
	converted, err := value.MultiplyInt64(1000)
	if err != nil || converted.String() != "9007199254740993000" {
		t.Fatalf("MultiplyInt64(1000) = %q, %v", converted.String(), err)
	}
}

func TestDecimalFloat64BoundaryMakesPrecisionLossExplicit(t *testing.T) {
	value, err := ParseDecimal("9007199254740993.0000")
	if err != nil {
		t.Fatal(err)
	}
	floating, err := value.Float64Checked()
	if err != nil {
		t.Fatalf("Float64Checked() error = %v", err)
	}
	converted, err := DecimalFromFloat64(floating)
	if err != nil {
		t.Fatalf("DecimalFromFloat64() error = %v", err)
	}
	if got := converted.String(); got != "9007199254740992" {
		t.Fatalf("float64 calculation boundary = %s, want documented precision loss 9007199254740992", got)
	}
}

func TestDecimalFitsWithoutRoundingAndConvertsExplicitly(t *testing.T) {
	valid, err := ParseDecimal("12500500.1251")
	if err != nil {
		t.Fatal(err)
	}
	if !valid.Fits(24, 4) || valid.Fits(24, 3) {
		t.Fatalf("Fits() did not enforce DECIMAL scale for %s", valid.String())
	}
	if _, err := valid.Float64Checked(); err != nil {
		t.Fatalf("Float64Checked() error = %v", err)
	}
	tooLarge, err := ParseDecimal("123456789012345678901.1")
	if err != nil {
		t.Fatal(err)
	}
	if tooLarge.Fits(24, 4) {
		t.Fatal("Fits() accepted value beyond DECIMAL integer precision")
	}
	subunit, err := ParseDecimal("0.1234")
	if err != nil || !subunit.Fits(4, 4) {
		t.Fatalf("DECIMAL(4,4) rejected a valid subunit value: %v", err)
	}
	zero, err := ParseDecimal("0")
	if err != nil || !zero.Fits(4, 4) {
		t.Fatalf("DECIMAL(4,4) rejected zero: %v", err)
	}
}

func TestDecimalJSONAndSQLTextRoundTripPreservesValue(t *testing.T) {
	want := "9007199254740993.0000"
	var value Decimal
	if err := json.Unmarshal([]byte(want), &value); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	encoded, err := json.Marshal(value)
	if err != nil || string(encoded) != "9007199254740993" {
		t.Fatalf("MarshalJSON() = %s, %v", encoded, err)
	}
	// Repositories bind this canonical string and parse CAST(... AS CHAR) results.
	if value.String() != "9007199254740993" {
		t.Fatalf("SQL bind text = %q", value.String())
	}
	readBack, err := ParseDecimal(want)
	if err != nil || readBack.String() != value.String() {
		t.Fatalf("ParseDecimal(CAST text) = %q, %v", readBack.String(), err)
	}
}

func TestDecimalRejectsInvalidJSONAndSQLValues(t *testing.T) {
	for _, input := range []string{`"1.2"`, `null`, `NaN`, `1e999999`} {
		var value Decimal
		if err := json.Unmarshal([]byte(input), &value); err == nil {
			t.Errorf("UnmarshalJSON(%s) error = nil", input)
		}
	}
	if _, err := ParseDecimal("not-a-decimal"); err == nil {
		t.Fatal("ParseDecimal() accepted invalid SQL text")
	}
}
