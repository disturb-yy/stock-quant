// Package demo 负责开发环境演示数据的用例编排和 HTTP 适配。
package demo

import (
	"fmt"
	"time"

	analysisdomain "github.com/disturb-yy/stock-quant/internal/analysis/domain"
	marketdomain "github.com/disturb-yy/stock-quant/internal/market/domain"
	stockdomain "github.com/disturb-yy/stock-quant/internal/stock/domain"
)

const (
	// SeedName 是数据库中演示数据元数据的稳定名称。
	SeedName = "fnd-003-demo"
	// SeedVersion 是本 US 的可追踪 fixture 版本。
	SeedVersion = "fnd-003-demo-v5"
	// SeedAsOf 是 fixture 的统一观测日期。
	SeedAsOf = "2024-06-28"
	// SeedObservedAt 是 Seed 数据统一的 UTC 观测时间。
	SeedObservedAt = "2024-06-28T07:00:00Z"
)

// Counts 是演示数据各类记录的数量。
type Counts struct {
	Instruments      int64 `json:"instruments"`
	DailyBars        int64 `json:"daily_bars"`
	FinancialMetrics int64 `json:"financial_metrics"`
	DailyBasics      int64 `json:"daily_basics"`
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
	DailyBasics       []marketdomain.DailyBasic
	FinancialMetrics  []analysisdomain.FinancialMetric
	IndexSnapshots    []marketdomain.IndexSnapshot
}

// DemoFixture 返回新的 fixture 值，调用方可以安全地修改其切片。
func DemoFixture() Fixture {
	plans := signalFixturePlans()
	return Fixture{
		Version:     SeedVersion,
		AsOf:        SeedAsOf,
		Instruments: signalFixtureInstruments(plans),
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
		DailyBars: signalFixtureDailyBars(plans),
		DailyBasics: []marketdomain.DailyBasic{
			{InstrumentCode: "000001.SZ", TradeDate: SeedAsOf, MarketCap: "203425.00", PB: "0.48"},
			{InstrumentCode: "300750.SZ", TradeDate: SeedAsOf, MarketCap: "887612.00", PB: "3.92"},
			{InstrumentCode: "600519.SH", TradeDate: SeedAsOf, MarketCap: "1854300.00", PB: "8.41"},
		},
		FinancialMetrics: []analysisdomain.FinancialMetric{
			{InstrumentCode: "000001.SZ", MetricDate: SeedAsOf, MetricName: "pe_ttm", Basis: "ttm", MetricValue: "5.82"},
			{InstrumentCode: "000001.SZ", MetricDate: SeedAsOf, MetricName: "roe", Basis: "latest_report", MetricValue: "10.84"},
			{InstrumentCode: "300750.SZ", MetricDate: SeedAsOf, MetricName: "pe_ttm", Basis: "ttm", MetricValue: "22.41"},
			{InstrumentCode: "300750.SZ", MetricDate: SeedAsOf, MetricName: "roe", Basis: "latest_report", MetricValue: "18.26"},
			{InstrumentCode: "600519.SH", MetricDate: SeedAsOf, MetricName: "pe_ttm", Basis: "ttm", MetricValue: "28.63"},
			{InstrumentCode: "600519.SH", MetricDate: SeedAsOf, MetricName: "roe", Basis: "latest_report", MetricValue: "31.42"},
			// 换手率属于版本化演示指标，不得作为实时行情使用。
			{InstrumentCode: "000001.SZ", MetricDate: SeedAsOf, MetricName: "turnover_rate", Basis: "latest_daily_basic", MetricValue: "1.86"},
			{InstrumentCode: "300750.SZ", MetricDate: SeedAsOf, MetricName: "turnover_rate", Basis: "latest_daily_basic", MetricValue: "3.54"},
			{InstrumentCode: "600519.SH", MetricDate: SeedAsOf, MetricName: "turnover_rate", Basis: "latest_daily_basic", MetricValue: "0.43"},
			{InstrumentCode: "000002.SZ", MetricDate: SeedAsOf, MetricName: "turnover_rate", Basis: "latest_daily_basic", MetricValue: "4.50"},
			{InstrumentCode: "000858.SZ", MetricDate: SeedAsOf, MetricName: "turnover_rate", Basis: "latest_daily_basic", MetricValue: "2.24"},
			{InstrumentCode: "002594.SZ", MetricDate: SeedAsOf, MetricName: "turnover_rate", Basis: "latest_daily_basic", MetricValue: "2.78"},
			{InstrumentCode: "601318.SH", MetricDate: SeedAsOf, MetricName: "turnover_rate", Basis: "latest_daily_basic", MetricValue: "0.96"},
			{InstrumentCode: "601398.SH", MetricDate: SeedAsOf, MetricName: "turnover_rate", Basis: "latest_daily_basic", MetricValue: "0.51"},
			{InstrumentCode: "601166.SH", MetricDate: SeedAsOf, MetricName: "turnover_rate", Basis: "latest_daily_basic", MetricValue: "1.34"},
			{InstrumentCode: "600036.SH", MetricDate: SeedAsOf, MetricName: "turnover_rate", Basis: "latest_daily_basic", MetricValue: "1.12"},
		},
		IndexSnapshots: []marketdomain.IndexSnapshot{
			{Code: "000001.SH", Name: "上证指数", TradeDate: SeedAsOf, ObservedAt: SeedObservedAt, Close: "2994.73", Change: "-3.89", ChangePercent: "-0.13"},
			{Code: "399001.SZ", Name: "深证成指", TradeDate: SeedAsOf, ObservedAt: SeedObservedAt, Close: "8848.42", Change: "-25.67", ChangePercent: "-0.29"},
			{Code: "399006.SZ", Name: "创业板指", TradeDate: SeedAsOf, ObservedAt: SeedObservedAt, Close: "1683.34", Change: "-8.12", ChangePercent: "-0.48"},
			{Code: "000300.SH", Name: "沪深300", TradeDate: SeedAsOf, ObservedAt: SeedObservedAt, Close: "3401.76", Change: "-5.87", ChangePercent: "-0.17"},
		},
	}
}

type signalFixturePlan struct {
	Code             string
	Name             string
	Exchange         string
	StartClose       float64
	EndClose         float64
	HistoricalVolume int64
	LatestVolume     int64
}

func signalFixturePlans() []signalFixturePlan {
	return []signalFixturePlan{
		{Code: "000001.SZ", Name: "平安银行", Exchange: "SZSE", StartClose: 9.50, EndClose: 10.31, HistoricalVolume: 50000000, LatestVolume: 81540000},
		{Code: "300750.SZ", Name: "宁德时代", Exchange: "SZSE", StartClose: 150.00, EndClose: 190.12, HistoricalVolume: 20000000, LatestVolume: 40180000},
		{Code: "600519.SH", Name: "贵州茅台", Exchange: "SSE", StartClose: 1550.00, EndClose: 1475.50, HistoricalVolume: 2500000, LatestVolume: 2390000},
		{Code: "000002.SZ", Name: "万科A", Exchange: "SZSE", StartClose: 20.00, EndClose: 25.50, HistoricalVolume: 30000000, LatestVolume: 36000000},
		{Code: "000858.SZ", Name: "五粮液", Exchange: "SZSE", StartClose: 100.00, EndClose: 110.00, HistoricalVolume: 18000000, LatestVolume: 22000000},
		{Code: "002594.SZ", Name: "比亚迪", Exchange: "SZSE", StartClose: 200.00, EndClose: 210.00, HistoricalVolume: 15000000, LatestVolume: 19000000},
		{Code: "601318.SH", Name: "中国平安", Exchange: "SSE", StartClose: 40.00, EndClose: 50.00, HistoricalVolume: 25000000, LatestVolume: 30000000},
		{Code: "601398.SH", Name: "工商银行", Exchange: "SSE", StartClose: 4.00, EndClose: 4.50, HistoricalVolume: 45000000, LatestVolume: 50000000},
		{Code: "601166.SH", Name: "兴业银行", Exchange: "SSE", StartClose: 15.00, EndClose: 16.00, HistoricalVolume: 22000000, LatestVolume: 26000000},
		{Code: "600036.SH", Name: "招商银行", Exchange: "SSE", StartClose: 30.00, EndClose: 35.00, HistoricalVolume: 20000000, LatestVolume: 24000000},
	}
}

func signalFixtureInstruments(plans []signalFixturePlan) []stockdomain.Instrument {
	instruments := make([]stockdomain.Instrument, 0, len(plans))
	for _, plan := range plans {
		instruments = append(instruments, stockdomain.Instrument{
			Code: plan.Code, Name: plan.Name, Exchange: plan.Exchange,
			Status: stockdomain.InstrumentStatusActive, AsOf: SeedAsOf,
		})
	}
	return instruments
}

func signalFixtureDailyBars(plans []signalFixturePlan) []marketdomain.DailyBar {
	dates := signalFixtureTradingDates(121)
	bars := make([]marketdomain.DailyBar, 0, len(plans)*len(dates))
	for _, plan := range plans {
		for index, date := range dates {
			bars = append(bars, signalFixtureBar(plan, index, date, len(dates)))
		}
	}
	return bars
}

func signalFixtureTradingDates(count int) []string {
	end, _ := time.Parse("2006-01-02", SeedAsOf)
	dates := make([]string, count)
	index := count - 1
	for index >= 0 {
		if end.Weekday() != time.Saturday && end.Weekday() != time.Sunday {
			dates[index] = end.Format("2006-01-02")
			index--
		}
		end = end.AddDate(0, 0, -1)
	}
	return dates
}

func signalFixtureBar(plan signalFixturePlan, index int, date string, total int) marketdomain.DailyBar {
	progress := float64(index) / float64(total-1)
	close := plan.StartClose + (plan.EndClose-plan.StartClose)*progress
	volume := plan.HistoricalVolume
	if index == total-1 {
		close = plan.EndClose
		volume = plan.LatestVolume
	}
	if index == total-2 {
		return signalFixturePreviousBar(plan)
	}
	if index == total-1 {
		if bar, ok := signalFixtureLatestBar(plan); ok {
			return bar
		}
	}
	return marketdomain.DailyBar{
		InstrumentCode: plan.Code, TradeDate: date,
		Open: formatFixturePrice(close * 0.995), High: formatFixturePrice(close * 1.001),
		Low: formatFixturePrice(close * 0.985), Close: formatFixturePrice(close),
		Volume: volume, TurnoverAmount: formatFixturePrice(close * float64(volume)),
	}
}

func signalFixturePreviousBar(plan signalFixturePlan) marketdomain.DailyBar {
	previous := marketdomain.DailyBar{InstrumentCode: plan.Code, TradeDate: "2024-06-27"}
	switch plan.Code {
	case "000001.SZ":
		previous.Open, previous.High, previous.Low, previous.Close = "10.12", "10.28", "10.05", "10.22"
		previous.Volume, previous.TurnoverAmount = 78210000, "798296400.00"
	case "300750.SZ":
		previous.Open, previous.High, previous.Low, previous.Close = "185.20", "188.60", "183.10", "187.45"
		previous.Volume, previous.TurnoverAmount = 36210000, "6785914500.00"
	case "600519.SH":
		previous.Open, previous.High, previous.Low, previous.Close = "1468.00", "1482.50", "1459.01", "1478.00"
		previous.Volume, previous.TurnoverAmount = 2210000, "3266380000.00"
	default:
		close := plan.EndClose * 0.998
		previous.Open, previous.High, previous.Low, previous.Close = formatFixturePrice(close), formatFixturePrice(close*1.01), formatFixturePrice(close*0.985), formatFixturePrice(close)
		previous.Volume, previous.TurnoverAmount = plan.HistoricalVolume, formatFixturePrice(close*float64(plan.HistoricalVolume))
	}
	return previous
}

func signalFixtureLatestBar(plan signalFixturePlan) (marketdomain.DailyBar, bool) {
	switch plan.Code {
	case "000001.SZ":
		return marketdomain.DailyBar{InstrumentCode: plan.Code, TradeDate: SeedAsOf, Open: "10.22", High: "10.36", Low: "10.18", Close: "10.31", Volume: 81540000, TurnoverAmount: "840027600.00"}, true
	case "300750.SZ":
		return marketdomain.DailyBar{InstrumentCode: plan.Code, TradeDate: SeedAsOf, Open: "187.45", High: "191.80", Low: "186.70", Close: "190.12", Volume: 40180000, TurnoverAmount: "7633021600.00"}, true
	case "600519.SH":
		return marketdomain.DailyBar{InstrumentCode: plan.Code, TradeDate: SeedAsOf, Open: "1478.00", High: "1485.20", Low: "1470.00", Close: "1475.50", Volume: 2390000, TurnoverAmount: "3529445000.00"}, true
	default:
		return marketdomain.DailyBar{}, false
	}
}

func formatFixturePrice(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

// Validate 检查 fixture 版本及各类实体的领域约束。
func (fixture Fixture) Validate() error {
	if fixture.Version == "" || fixture.AsOf == "" {
		return fmt.Errorf("fixture version and as-of are required")
	}
	if len(fixture.Instruments) == 0 || len(fixture.Sectors) == 0 || len(fixture.SectorMemberships) == 0 || len(fixture.DailyBars) == 0 || len(fixture.DailyBasics) == 0 || len(fixture.FinancialMetrics) == 0 || len(fixture.IndexSnapshots) == 0 {
		return fmt.Errorf("fixture must contain instruments, sectors, sector memberships, daily bars, daily basics, financial metrics, and index snapshots")
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
	for _, basic := range fixture.DailyBasics {
		if err := basic.Validate(); err != nil {
			return fmt.Errorf("validate daily basic %q/%q: %w", basic.InstrumentCode, basic.TradeDate, err)
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
		DailyBasics:      int64(len(fixture.DailyBasics)),
		FinancialMetrics: int64(len(fixture.FinancialMetrics)),
		IndexSnapshots:   int64(len(fixture.IndexSnapshots)),
	}
}

// Equal 判断数据库快照是否仍与该 fixture 的数量完全一致。
func (counts Counts) Equal(other Counts) bool {
	return counts == other
}
