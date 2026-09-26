package source

import (
	"context"
	"time"

	"github.com/disturb-yy/stock-quant/internal/data/domain"
)

const mockTimestamp = "2026-09-26T08:00:00Z"

type MockAdapter struct {
	basic domain.BasicInfoBatch
	bars  domain.DailyBarBatch
}

func NewMockAdapter() *MockAdapter {
	updatedAt := parseMockTime(mockTimestamp)
	dataAsOf := parseMockDate("2026-09-25")
	source := domain.SourceProvenance{Provider: "mock", Mode: "mock"}
	return &MockAdapter{
		basic: domain.BasicInfoBatch{
			Items: []domain.Instrument{
				{Symbol: "000001", Market: "SZ", Name: "平安银行", Status: "L", Source: source, DataAsOf: &dataAsOf, UpdatedAt: updatedAt},
				{Symbol: "600000", Market: "SH", Name: "浦发银行", Status: "L", Source: source, DataAsOf: &dataAsOf, UpdatedAt: updatedAt},
			},
			DataAsOf: &dataAsOf,
		},
		bars: domain.DailyBarBatch{
			Items: []domain.DailyBar{
				mockBar("000001", "SZ", "2026-09-24", 10.0, 10.5, 9.8, 10.2, 1200000, source, updatedAt),
				mockBar("000001", "SZ", "2026-09-25", 10.2, 10.8, 10.1, 10.6, 1300000, source, updatedAt),
				mockBar("600000", "SH", "2026-09-24", 8.0, 8.2, 7.9, 8.1, 800000, source, updatedAt),
				mockBar("600000", "SH", "2026-09-25", 8.1, 8.3, 8.0, 8.2, 850000, source, updatedAt),
			},
			DataAsOf: &dataAsOf,
		},
	}
}

func (adapter *MockAdapter) Provenance() domain.SourceProvenance {
	return domain.SourceProvenance{Provider: "mock", Mode: "mock"}
}

func (adapter *MockAdapter) FetchBasicInfo(ctx context.Context) (domain.BasicInfoBatch, error) {
	if err := ctx.Err(); err != nil {
		return domain.BasicInfoBatch{}, err
	}
	return cloneBasicBatch(adapter.basic), nil
}

func (adapter *MockAdapter) FetchDailyBars(ctx context.Context, dateRange domain.DateRange) (domain.DailyBarBatch, error) {
	if err := ctx.Err(); err != nil {
		return domain.DailyBarBatch{}, err
	}
	if err := dateRange.ValidateFor(domain.TargetDailyBars); err != nil {
		return domain.DailyBarBatch{}, err
	}
	result := domain.DailyBarBatch{}
	for _, bar := range adapter.bars.Items {
		if dateRange.Contains(bar.TradeDate) {
			result.Items = append(result.Items, bar)
			result.DataAsOf = latestDate(result.DataAsOf, bar.TradeDate)
		}
	}
	return result, nil
}

func mockBar(symbol, market, date string, open, high, low, close, volume float64,
	source domain.SourceProvenance, updatedAt time.Time) domain.DailyBar {
	return domain.DailyBar{Symbol: symbol, Market: market, TradeDate: parseMockDate(date), Open: open,
		High: high, Low: low, Close: close, Volume: volume, Source: source, UpdatedAt: updatedAt}
}

func cloneBasicBatch(batch domain.BasicInfoBatch) domain.BasicInfoBatch {
	items := append([]domain.Instrument(nil), batch.Items...)
	return domain.BasicInfoBatch{Items: items, DataAsOf: cloneTime(batch.DataAsOf)}
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func latestDate(current *time.Time, candidate time.Time) *time.Time {
	if current == nil || candidate.After(*current) {
		value := candidate
		return &value
	}
	return current
}

func parseMockDate(value string) time.Time {
	date, err := time.Parse(domain.DateLayout, value)
	if err != nil {
		panic(err)
	}
	return date
}

func parseMockTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
