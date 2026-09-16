package domain

import (
	"fmt"
	"strings"
	"testing"
)

func TestApplyAdjustmentAndMovingAverages(t *testing.T) {
	bars := []BarWithAdjustment{
		{Bar: chartDailyBar("2024-06-03", "10", 100), QFQFactor: "0.95", HFQFactor: "1.10"},
		{Bar: chartDailyBar("2024-06-04", "11", 200), QFQFactor: "0.95", HFQFactor: "1.10"},
	}
	adjusted, err := ApplyAdjustment(bars[0], AdjustmentQFQ)
	if err != nil {
		t.Fatalf("ApplyAdjustment() error = %v", err)
	}
	if adjusted.Close != "9.50" || adjusted.Volume != 100 {
		t.Fatalf("adjusted bar = %#v, want close 9.50 and unchanged volume", adjusted)
	}
	hfqAdjusted, err := ApplyAdjustment(bars[0], AdjustmentHFQ)
	if err != nil {
		t.Fatalf("ApplyAdjustment(hfq) error = %v", err)
	}
	if hfqAdjusted.Close != "11.00" || hfqAdjusted.Volume != 100 {
		t.Fatalf("hfq adjusted bar = %#v, want close 11.00 and unchanged volume", hfqAdjusted)
	}

	chartBars := make([]ChartBar, 0, 20)
	for index := 0; index < 20; index++ {
		bar, err := ApplyAdjustment(BarWithAdjustment{
			Bar: chartDailyBar(fmt.Sprintf("2024-06-%02d", index+1), fmt.Sprintf("%d", index+1), 100), QFQFactor: "1", HFQFactor: "1",
		}, AdjustmentNone)
		if err != nil {
			t.Fatalf("ApplyAdjustment(%d) error = %v", index, err)
		}
		chartBars = append(chartBars, bar)
	}
	if err := CalculateMovingAverages(chartBars); err != nil {
		t.Fatalf("CalculateMovingAverages() error = %v", err)
	}
	if chartBars[3].MA5 != nil || chartBars[18].MA20 != nil {
		t.Fatal("moving averages must be nil before their windows are complete")
	}
	if chartBars[4].MA5 == nil || *chartBars[4].MA5 != "3.00" || chartBars[19].MA20 == nil || *chartBars[19].MA20 != "10.50" {
		t.Fatalf("moving averages = %#v/%#v, want 3.00/10.50", chartBars[4].MA5, chartBars[19].MA20)
	}
}

func TestBuildBenchmarkPointsUses共同TradingDates(t *testing.T) {
	stock := []ChartBar{
		{TradeDate: "2024-06-03", Close: "100"},
		{TradeDate: "2024-06-04", Close: "110"},
		{TradeDate: "2024-06-05", Close: "105"},
	}
	benchmark := []BenchmarkBar{
		{TradeDate: "2024-06-03", Close: "200"},
		{TradeDate: "2024-06-05", Close: "220"},
	}
	points, err := BuildBenchmarkPoints(stock, benchmark)
	if err != nil {
		t.Fatalf("BuildBenchmarkPoints() error = %v", err)
	}
	if len(points) != 2 || points[0].TradeDate != "2024-06-03" || points[1].TradeDate != "2024-06-05" {
		t.Fatalf("points = %#v, want two common dates", points)
	}
	if points[1].StockReturnPct != "5.00" || points[1].BenchmarkReturnPct != "10.00" || points[1].RelativeReturnPct != "-5.00" {
		t.Fatalf("returns = %#v, want 5.00/10.00/-5.00", points[1])
	}
}

func TestBuildBenchmarkPointsReturnsEmptyWhenNoCommonDate(t *testing.T) {
	points, err := BuildBenchmarkPoints(
		[]ChartBar{{TradeDate: "2024-06-03", Close: "100"}},
		[]BenchmarkBar{{TradeDate: "2024-06-04", Close: "200"}},
	)
	if err != nil {
		t.Fatalf("BuildBenchmarkPoints() error = %v", err)
	}
	if points == nil || len(points) != 0 {
		t.Fatalf("points = %#v, want non-nil empty list", points)
	}
}

func TestApplyAdjustmentRejectsMissingFactor(t *testing.T) {
	_, err := ApplyAdjustment(BarWithAdjustment{Bar: chartDailyBar("2024-06-03", "10", 1)}, AdjustmentQFQ)
	if err == nil || !strings.Contains(err.Error(), "adjustment factor") {
		t.Fatalf("error = %v, want missing adjustment factor", err)
	}
}

func chartDailyBar(date, close string, volume int64) DailyBar {
	return DailyBar{InstrumentCode: "000001.SZ", TradeDate: date, Open: close, High: close, Low: close, Close: close, Volume: volume, TurnoverAmount: "1"}
}
