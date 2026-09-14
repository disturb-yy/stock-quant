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
	SeedVersion = "fnd-003-demo-v3"
	// SeedAsOf 是 fixture 的统一观测日期。
	SeedAsOf = "2024-06-28"
	// SeedObservedAt 是 Seed 数据统一的 UTC 观测时间。
	SeedObservedAt = "2024-06-28T07:00:00Z"
)

// Counts 是四类演示数据的数量。
type Counts struct {
	Instruments      int64 `json:"instruments"`
	DailyBars        int64 `json:"daily_bars"`
	FinancialMetrics int64 `json:"financial_metrics"`
	IndexSnapshots   int64 `json:"index_snapshots"`
}

// Fixture 是一次完整、确定的演示数据输入。
type Fixture struct {
	Version           string
	AsOf              string
	Instruments       []stockdomain.Instrument
	Sectors           []marketdomain.Sector
	SectorMemberships []marketdomain.SectorMembership
	DailyBars         []marketdomain.DailyBar
	FinancialMetrics  []analysisdomain.FinancialMetric
	IndexSnapshots    []marketdomain.IndexSnapshot
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
		Sectors: []marketdomain.Sector{
			{Code: "BANK", Name: "银行"},
			{Code: "EQUIPMENT", Name: "电力设备"},
			{Code: "FOOD_BEVERAGE", Name: "食品饮料"},
		},
		SectorMemberships: []marketdomain.SectorMembership{
			{SectorCode: "BANK", InstrumentCode: "000001.SZ"},
			{SectorCode: "EQUIPMENT", InstrumentCode: "300750.SZ"},
			{SectorCode: "FOOD_BEVERAGE", InstrumentCode: "600519.SH"},
		},
		DailyBars: []marketdomain.DailyBar{
			{InstrumentCode: "000001.SZ", TradeDate: "2024-06-27", Open: "10.12", High: "10.28", Low: "10.05", Close: "10.22", Volume: 78210000, TurnoverAmount: "798296400.00"},
			{InstrumentCode: "000001.SZ", TradeDate: SeedAsOf, Open: "10.22", High: "10.36", Low: "10.18", Close: "10.31", Volume: 81540000, TurnoverAmount: "840027600.00"},
			{InstrumentCode: "300750.SZ", TradeDate: "2024-06-27", Open: "185.20", High: "188.60", Low: "183.10", Close: "187.45", Volume: 36210000, TurnoverAmount: "6785914500.00"},
			{InstrumentCode: "300750.SZ", TradeDate: SeedAsOf, Open: "187.45", High: "191.80", Low: "186.70", Close: "190.12", Volume: 40180000, TurnoverAmount: "7633021600.00"},
			{InstrumentCode: "600519.SH", TradeDate: "2024-06-27", Open: "1468.00", High: "1482.50", Low: "1459.01", Close: "1478.00", Volume: 2210000, TurnoverAmount: "3266380000.00"},
			{InstrumentCode: "600519.SH", TradeDate: SeedAsOf, Open: "1478.00", High: "1485.20", Low: "1470.00", Close: "1475.50", Volume: 2390000, TurnoverAmount: "3529445000.00"},
		},
		FinancialMetrics: []analysisdomain.FinancialMetric{
			{InstrumentCode: "000001.SZ", MetricDate: SeedAsOf, MetricName: "pe_ttm", MetricValue: "5.82"},
			{InstrumentCode: "000001.SZ", MetricDate: SeedAsOf, MetricName: "roe", MetricValue: "10.84"},
			{InstrumentCode: "300750.SZ", MetricDate: SeedAsOf, MetricName: "pe_ttm", MetricValue: "22.41"},
			{InstrumentCode: "300750.SZ", MetricDate: SeedAsOf, MetricName: "roe", MetricValue: "18.26"},
			{InstrumentCode: "600519.SH", MetricDate: SeedAsOf, MetricName: "pe_ttm", MetricValue: "28.63"},
			{InstrumentCode: "600519.SH", MetricDate: SeedAsOf, MetricName: "roe", MetricValue: "31.42"},
		},
		IndexSnapshots: []marketdomain.IndexSnapshot{
			{Code: "000001.SH", Name: "上证指数", TradeDate: SeedAsOf, ObservedAt: SeedObservedAt, Close: "2994.73", Change: "-3.89", ChangePercent: "-0.13"},
			{Code: "399001.SZ", Name: "深证成指", TradeDate: SeedAsOf, ObservedAt: SeedObservedAt, Close: "8848.42", Change: "-25.67", ChangePercent: "-0.29"},
			{Code: "399006.SZ", Name: "创业板指", TradeDate: SeedAsOf, ObservedAt: SeedObservedAt, Close: "1683.34", Change: "-8.12", ChangePercent: "-0.48"},
			{Code: "000300.SH", Name: "沪深300", TradeDate: SeedAsOf, ObservedAt: SeedObservedAt, Close: "3401.76", Change: "-5.87", ChangePercent: "-0.17"},
		},
	}
}

// Validate 检查 fixture 版本及各类实体的领域约束。
func (fixture Fixture) Validate() error {
	if fixture.Version == "" || fixture.AsOf == "" {
		return fmt.Errorf("fixture version and as-of are required")
	}
	if len(fixture.Instruments) == 0 || len(fixture.Sectors) == 0 || len(fixture.SectorMemberships) == 0 || len(fixture.DailyBars) == 0 || len(fixture.FinancialMetrics) == 0 || len(fixture.IndexSnapshots) == 0 {
		return fmt.Errorf("fixture must contain instruments, sectors, sector memberships, daily bars, financial metrics, and index snapshots")
	}
	for _, instrument := range fixture.Instruments {
		if err := instrument.Validate(); err != nil {
			return fmt.Errorf("validate instrument %q: %w", instrument.Code, err)
		}
	}
	if err := validateSectors(fixture); err != nil {
		return err
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
	for _, index := range fixture.IndexSnapshots {
		if err := index.Validate(); err != nil {
			return fmt.Errorf("validate index snapshot %q: %w", index.Code, err)
		}
	}
	return nil
}

func validateSectors(fixture Fixture) error {
	sectorCodes := make(map[string]struct{}, len(fixture.Sectors))
	for _, sector := range fixture.Sectors {
		if err := sector.Validate(); err != nil {
			return fmt.Errorf("validate sector %q: %w", sector.Code, err)
		}
		if _, exists := sectorCodes[sector.Code]; exists {
			return fmt.Errorf("duplicate sector %q", sector.Code)
		}
		sectorCodes[sector.Code] = struct{}{}
	}
	instrumentCodes := make(map[string]struct{}, len(fixture.Instruments))
	for _, instrument := range fixture.Instruments {
		instrumentCodes[instrument.Code] = struct{}{}
	}
	seenMemberships := make(map[string]struct{}, len(fixture.SectorMemberships))
	for _, membership := range fixture.SectorMemberships {
		if err := membership.Validate(); err != nil {
			return fmt.Errorf("validate sector membership %q/%q: %w", membership.SectorCode, membership.InstrumentCode, err)
		}
		if _, exists := sectorCodes[membership.SectorCode]; !exists {
			return fmt.Errorf("sector membership references unknown sector %q", membership.SectorCode)
		}
		if _, exists := instrumentCodes[membership.InstrumentCode]; !exists {
			return fmt.Errorf("sector membership references unknown instrument %q", membership.InstrumentCode)
		}
		key := membership.SectorCode + "\x00" + membership.InstrumentCode
		if _, exists := seenMemberships[key]; exists {
			return fmt.Errorf("duplicate sector membership %q/%q", membership.SectorCode, membership.InstrumentCode)
		}
		seenMemberships[key] = struct{}{}
	}
	return nil
}

// DataCounts 返回 fixture 的预期数量。
func (fixture Fixture) DataCounts() Counts {
	return Counts{
		Instruments:      int64(len(fixture.Instruments)),
		DailyBars:        int64(len(fixture.DailyBars)),
		FinancialMetrics: int64(len(fixture.FinancialMetrics)),
		IndexSnapshots:   int64(len(fixture.IndexSnapshots)),
	}
}

// Equal 判断数据库快照是否仍与该 fixture 的数量完全一致。
func (counts Counts) Equal(other Counts) bool {
	return counts == other
}
