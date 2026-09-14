// Package demo 负责开发环境演示数据的用例编排和 HTTP 适配。
package demo

import (
	"fmt"

	analysisdomain "github.com/disturb-yy/stock-quant/internal/analysis/domain"
	marketdomain "github.com/disturb-yy/stock-quant/internal/market/domain"
	stockdomain "github.com/disturb-yy/stock-quant/internal/stock/domain"
)

const (
	// SeedName 是数据库中演示数据元数据的稳定名称。
	SeedName = "fnd-003-demo"
	// SeedVersion 是本 US 的可追踪 fixture 版本。
	SeedVersion = "fnd-003-demo-v1"
	// SeedAsOf 是 fixture 的统一观测日期。
	SeedAsOf = "2024-06-28"
)

// Counts 是三类演示数据的数量。
type Counts struct {
	Instruments      int64 `json:"instruments"`
	DailyBars        int64 `json:"daily_bars"`
	FinancialMetrics int64 `json:"financial_metrics"`
}

// Fixture 是一次完整、确定的演示数据输入。
type Fixture struct {
	Version          string
	AsOf             string
	Instruments      []stockdomain.Instrument
	DailyBars        []marketdomain.DailyBar
	FinancialMetrics []analysisdomain.FinancialMetric
}

// DemoFixture 返回新的 fixture 值，调用方可以安全地修改其切片。
func DemoFixture() Fixture {
	return Fixture{
		Version: SeedVersion,
		AsOf:    SeedAsOf,
		Instruments: []stockdomain.Instrument{
			{Code: "000001.SZ", Name: "平安银行", Exchange: "SZSE", Status: stockdomain.InstrumentStatusActive, AsOf: SeedAsOf},
			{Code: "300750.SZ", Name: "宁德时代", Exchange: "SZSE", Status: stockdomain.InstrumentStatusActive, AsOf: SeedAsOf},
			{Code: "600519.SH", Name: "贵州茅台", Exchange: "SSE", Status: stockdomain.InstrumentStatusActive, AsOf: SeedAsOf},
		},
		DailyBars: []marketdomain.DailyBar{
			{InstrumentCode: "000001.SZ", TradeDate: "2024-06-27", Open: "10.12", High: "10.28", Low: "10.05", Close: "10.22", Volume: 78210000},
			{InstrumentCode: "000001.SZ", TradeDate: SeedAsOf, Open: "10.22", High: "10.36", Low: "10.18", Close: "10.31", Volume: 81540000},
			{InstrumentCode: "300750.SZ", TradeDate: "2024-06-27", Open: "185.20", High: "188.60", Low: "183.10", Close: "187.45", Volume: 36210000},
			{InstrumentCode: "300750.SZ", TradeDate: SeedAsOf, Open: "187.45", High: "191.80", Low: "186.70", Close: "190.12", Volume: 40180000},
			{InstrumentCode: "600519.SH", TradeDate: "2024-06-27", Open: "1468.00", High: "1482.50", Low: "1459.01", Close: "1478.00", Volume: 2210000},
			{InstrumentCode: "600519.SH", TradeDate: SeedAsOf, Open: "1478.00", High: "1491.20", Low: "1470.00", Close: "1488.88", Volume: 2390000},
		},
		FinancialMetrics: []analysisdomain.FinancialMetric{
			{InstrumentCode: "000001.SZ", MetricDate: SeedAsOf, MetricName: "pe_ttm", MetricValue: "5.82"},
			{InstrumentCode: "000001.SZ", MetricDate: SeedAsOf, MetricName: "roe", MetricValue: "10.84"},
			{InstrumentCode: "300750.SZ", MetricDate: SeedAsOf, MetricName: "pe_ttm", MetricValue: "22.41"},
			{InstrumentCode: "300750.SZ", MetricDate: SeedAsOf, MetricName: "roe", MetricValue: "18.26"},
			{InstrumentCode: "600519.SH", MetricDate: SeedAsOf, MetricName: "pe_ttm", MetricValue: "28.63"},
			{InstrumentCode: "600519.SH", MetricDate: SeedAsOf, MetricName: "roe", MetricValue: "31.42"},
		},
	}
}

// Validate 检查 fixture 版本及三类实体的领域约束。
func (fixture Fixture) Validate() error {
	if fixture.Version == "" || fixture.AsOf == "" {
		return fmt.Errorf("fixture version and as-of are required")
	}
	if len(fixture.Instruments) == 0 || len(fixture.DailyBars) == 0 || len(fixture.FinancialMetrics) == 0 {
		return fmt.Errorf("fixture must contain instruments, daily bars, and financial metrics")
	}
	for _, instrument := range fixture.Instruments {
		if err := instrument.Validate(); err != nil {
			return fmt.Errorf("validate instrument %q: %w", instrument.Code, err)
		}
	}
	for _, bar := range fixture.DailyBars {
		if err := bar.Validate(); err != nil {
			return fmt.Errorf("validate daily bar %q/%q: %w", bar.InstrumentCode, bar.TradeDate, err)
		}
	}
	for _, metric := range fixture.FinancialMetrics {
		if err := metric.Validate(); err != nil {
			return fmt.Errorf("validate financial metric %q/%q: %w", metric.InstrumentCode, metric.MetricName, err)
		}
	}
	return nil
}

// DataCounts 返回 fixture 的预期数量。
func (fixture Fixture) DataCounts() Counts {
	return Counts{
		Instruments:      int64(len(fixture.Instruments)),
		DailyBars:        int64(len(fixture.DailyBars)),
		FinancialMetrics: int64(len(fixture.FinancialMetrics)),
	}
}

// Equal 判断数据库快照是否仍与该 fixture 的数量完全一致。
func (counts Counts) Equal(other Counts) bool {
	return counts == other
}
