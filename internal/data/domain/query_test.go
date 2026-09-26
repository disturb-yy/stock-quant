package domain

import (
	"errors"
	"testing"
)

func TestParseStockIdentifier(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		want      StockIdentifier
		wantError error
	}{
		{name: "sh", raw: "600519.SH", want: StockIdentifier{Symbol: "600519", Market: "SH"}},
		{name: "sz", raw: "000001.SZ", want: StockIdentifier{Symbol: "000001", Market: "SZ"}},
		{name: "bj", raw: "830001.BJ", want: StockIdentifier{Symbol: "830001", Market: "BJ"}},
		{name: "missing market suffix", raw: "600519", wantError: ErrInvalidStockSymbol},
		{name: "invalid suffix", raw: "600519.XX", wantError: ErrInvalidStockSymbol},
		{name: "unknown a share code", raw: "999999.SH", wantError: ErrStockNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseStockIdentifier(test.raw)
			if !errors.Is(err, test.wantError) {
				t.Fatalf("ParseStockIdentifier error = %v, want %v", err, test.wantError)
			}
			if test.wantError == nil && got != test.want {
				t.Fatalf("ParseStockIdentifier = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestParseOptionalDateRangeRequiresPairAndValidOrder(t *testing.T) {
	start := "2026-09-01"
	end := "2026-09-30"
	rangeValue, err := ParseOptionalDateRange(&start, &end)
	if err != nil || rangeValue.Start == nil || rangeValue.End == nil {
		t.Fatalf("ParseOptionalDateRange = %#v, %v", rangeValue, err)
	}
	if _, err := ParseOptionalDateRange(&start, nil); !errors.Is(err, ErrInvalidDateRange) {
		t.Fatalf("missing end date error = %v", err)
	}
	badEnd := "2026-08-31"
	if _, err := ParseOptionalDateRange(&start, &badEnd); !errors.Is(err, ErrInvalidDateRange) {
		t.Fatalf("reversed date error = %v", err)
	}
}
