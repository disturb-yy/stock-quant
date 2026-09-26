package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/disturb-yy/stock-quant/internal/data/domain"
)

type fakeStockQueryRepository struct {
	instrument    *domain.Instrument
	bars          []domain.DailyBar
	instrumentErr error
	barsErr       error
	lastRange     domain.DateRange
	lastLimit     int
}

func (repository *fakeStockQueryRepository) FindInstrument(_ context.Context, _ domain.StockIdentifier) (*domain.Instrument, error) {
	return repository.instrument, repository.instrumentErr
}

func (repository *fakeStockQueryRepository) FindDailyBars(_ context.Context, _ domain.StockIdentifier,
	dateRange domain.DateRange, limit int) ([]domain.DailyBar, error) {
	repository.lastRange = dateRange
	repository.lastLimit = limit
	return repository.bars, repository.barsErr
}

func TestStockQueryServiceReturnsFullResultAndDefaultLimit(t *testing.T) {
	updatedAt := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	barUpdatedAt := updatedAt.Add(time.Minute)
	dataAsOf := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	repository := &fakeStockQueryRepository{
		instrument: &domain.Instrument{Symbol: "600519", Market: "SH", Name: "贵州茅台", Status: "normal",
			Source: domain.SourceProvenance{Provider: "mock", Mode: "mock"}, DataAsOf: &dataAsOf, UpdatedAt: updatedAt},
		bars: []domain.DailyBar{
			{Symbol: "600519", Market: "SH", TradeDate: time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC),
				Open: 2, High: 3, Low: 1, Close: 2.5, Volume: 10, Source: domain.SourceProvenance{Provider: "tushare", Mode: "external"}, UpdatedAt: barUpdatedAt},
			{Symbol: "600519", Market: "SH", TradeDate: time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC),
				Open: 1, High: 2, Low: 1, Close: 1.5, Volume: 8, Source: domain.SourceProvenance{Provider: "tushare", Mode: "external"}, UpdatedAt: barUpdatedAt},
		},
	}
	service, err := NewStockQueryService(repository)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.QueryStockData(context.Background(), StockQueryInput{Symbol: "600519.SH"})
	if err != nil {
		t.Fatal(err)
	}
	if repository.lastLimit != domain.DefaultRecentTradingDays {
		t.Fatalf("default limit = %d, want %d", repository.lastLimit, domain.DefaultRecentTradingDays)
	}
	if result.Symbol != "600519.SH" || result.BasicInfo == nil || len(result.DailyBars) != 2 {
		t.Fatalf("unexpected full result: %#v", result)
	}
	if !result.DailyBars[0].TradeDate.Before(result.DailyBars[1].TradeDate) {
		t.Fatalf("daily bars are not ascending: %#v", result.DailyBars)
	}
	if result.Source == nil || *result.Source != repository.instrument.Source {
		t.Fatalf("source = %#v, want basic info source %#v", result.Source, repository.instrument.Source)
	}
	if result.UpdatedAt == nil || !result.UpdatedAt.Equal(barUpdatedAt) {
		t.Fatalf("updated_at = %v, want %v", result.UpdatedAt, barUpdatedAt)
	}
	wantDataAsOf := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	if result.DataAsOf == nil || !result.DataAsOf.Equal(wantDataAsOf) {
		t.Fatalf("data_as_of = %v, want %v", result.DataAsOf, wantDataAsOf)
	}
}

func TestStockQueryServicePreservesPartialAndEmptyAvailability(t *testing.T) {
	instrument := &domain.Instrument{Symbol: "000001", Market: "SZ", Name: "平安银行", Status: "normal",
		Source: domain.SourceProvenance{Provider: "mock", Mode: "mock"}, UpdatedAt: time.Now().UTC()}
	tests := []struct {
		name          string
		instrument    *domain.Instrument
		bars          []domain.DailyBar
		wantBasic     string
		wantBars      string
		wantSourceNil bool
	}{
		{name: "basic only", instrument: instrument, wantBasic: domain.AvailabilityAvailable, wantBars: domain.AvailabilityEmpty},
		{name: "daily bars only", bars: []domain.DailyBar{{Symbol: "000001", Market: "SZ", TradeDate: time.Now().UTC(), UpdatedAt: time.Now().UTC()}}, wantBasic: domain.AvailabilityEmpty, wantBars: domain.AvailabilityAvailable},
		{name: "both empty", wantBasic: domain.AvailabilityEmpty, wantBars: domain.AvailabilityEmpty, wantSourceNil: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, err := NewStockQueryService(&fakeStockQueryRepository{instrument: test.instrument, bars: test.bars})
			if err != nil {
				t.Fatal(err)
			}
			result, err := service.QueryStockData(context.Background(), StockQueryInput{Symbol: "000001.SZ"})
			if err != nil {
				t.Fatal(err)
			}
			if result.Availability.BasicInfo != test.wantBasic || result.Availability.DailyBars != test.wantBars {
				t.Fatalf("availability = %#v, want basic=%s bars=%s", result.Availability, test.wantBasic, test.wantBars)
			}
			if result.DailyBars == nil {
				t.Fatal("daily_bars must be a non-nil empty slice when empty")
			}
			if test.wantSourceNil != (result.Source == nil) {
				t.Fatalf("source nil = %t, want %t", result.Source == nil, test.wantSourceNil)
			}
		})
	}
}

func TestStockQueryServiceValidatesInputAndPropagatesDependencyError(t *testing.T) {
	service, err := NewStockQueryService(&fakeStockQueryRepository{instrumentErr: domain.ErrDataSourceUnavailable})
	if err != nil {
		t.Fatal(err)
	}
	start := "2026-09-01"
	if _, err := service.QueryStockData(context.Background(), StockQueryInput{Symbol: "bad", StartDate: &start}); !errors.Is(err, domain.ErrInvalidStockSymbol) {
		t.Fatalf("invalid symbol error = %v", err)
	}
	if _, err := service.QueryStockData(context.Background(), StockQueryInput{Symbol: "600519.SH", StartDate: &start}); !errors.Is(err, domain.ErrInvalidDateRange) {
		t.Fatalf("missing end date error = %v", err)
	}
	if _, err := service.QueryStockData(context.Background(), StockQueryInput{Symbol: "600519.SH"}); !errors.Is(err, domain.ErrDataSourceUnavailable) {
		t.Fatalf("dependency error = %v", err)
	}
}
