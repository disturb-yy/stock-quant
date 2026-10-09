package domain

import (
	"testing"

	"stock-quant/internal/shared/types"
)

func TestStockListedOnUsesInclusiveListingInterval(t *testing.T) {
	listDate := mustTradingDate(t, "2020-01-02")
	delistDate := mustTradingDate(t, "2024-06-30")
	stock := Stock{ListDate: listDate, DelistDate: &delistDate}

	tests := []struct {
		date string
		want bool
	}{
		{date: "2020-01-01", want: false},
		{date: "2020-01-02", want: true},
		{date: "2024-06-30", want: true},
		{date: "2024-07-01", want: false},
		{date: "not-a-date", want: false},
	}
	for _, test := range tests {
		t.Run(test.date, func(t *testing.T) {
			date, err := types.ParseTradingDate(test.date)
			if err != nil {
				if stock.ListedOn(types.TradingDate{}) {
					t.Fatal("invalid query date must not be considered listed")
				}
				return
			}
			if got := stock.ListedOn(date); got != test.want {
				t.Fatalf("ListedOn(%s) = %t, want %t", test.date, got, test.want)
			}
		})
	}
}

func TestStockListedOnOpenEndedAndInvalidIntervals(t *testing.T) {
	listDate := mustTradingDate(t, "2025-01-01")
	stock := Stock{ListDate: listDate}
	if !stock.ListedOn(mustTradingDate(t, "2026-01-01")) {
		t.Fatal("open-ended listing interval should include dates after list date")
	}
	if stock.ListedOn(mustTradingDate(t, "2024-12-31")) {
		t.Fatal("open-ended listing interval should exclude dates before list date")
	}
	if (Stock{}).ListedOn(listDate) {
		t.Fatal("invalid listing start should not be considered listed")
	}
}

func mustTradingDate(t *testing.T, value string) types.TradingDate {
	t.Helper()
	date, err := types.ParseTradingDate(value)
	if err != nil {
		t.Fatalf("parse trading date: %v", err)
	}
	return date
}
