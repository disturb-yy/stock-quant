package application

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/disturb-yy/stock-quant/internal/data/domain"
)

type StockQueryInput struct {
	Symbol    string
	StartDate *string
	EndDate   *string
}

type StockQueryService struct {
	repository domain.StockQueryRepository
}

func NewStockQueryService(repository domain.StockQueryRepository) (*StockQueryService, error) {
	if repository == nil {
		return nil, fmt.Errorf("stock query repository is nil")
	}
	return &StockQueryService{repository: repository}, nil
}

func (service *StockQueryService) QueryStockData(ctx context.Context, input StockQueryInput) (domain.StockQueryResult, error) {
	identifier, err := domain.ParseStockIdentifier(input.Symbol)
	if err != nil {
		return domain.StockQueryResult{}, err
	}
	dateRange, err := domain.ParseOptionalDateRange(input.StartDate, input.EndDate)
	if err != nil {
		return domain.StockQueryResult{}, err
	}
	limit := 0
	if dateRange.Start == nil {
		limit = domain.DefaultRecentTradingDays
	}
	instrument, err := service.repository.FindInstrument(ctx, identifier)
	if err != nil {
		return domain.StockQueryResult{}, err
	}
	bars, err := service.repository.FindDailyBars(ctx, identifier, dateRange, limit)
	if err != nil {
		return domain.StockQueryResult{}, err
	}
	return assembleStockQueryResult(identifier, instrument, bars), nil
}

func assembleStockQueryResult(identifier domain.StockIdentifier, instrument *domain.Instrument,
	bars []domain.DailyBar) domain.StockQueryResult {
	if bars == nil {
		bars = make([]domain.DailyBar, 0)
	}
	sort.SliceStable(bars, func(i, j int) bool {
		return bars[i].TradeDate.Before(bars[j].TradeDate)
	})
	result := domain.StockQueryResult{
		Symbol:    identifier.Qualified(),
		BasicInfo: instrument,
		DailyBars: bars,
		Availability: domain.StockDataAvailability{
			BasicInfo: domain.AvailabilityEmpty,
			DailyBars: domain.AvailabilityEmpty,
		},
	}
	if instrument != nil {
		result.Availability.BasicInfo = domain.AvailabilityAvailable
	}
	if len(bars) > 0 {
		result.Availability.DailyBars = domain.AvailabilityAvailable
	}
	result.Source, result.UpdatedAt, result.DataAsOf = mergeStockMetadata(instrument, bars)
	return result
}

func mergeStockMetadata(instrument *domain.Instrument, bars []domain.DailyBar) (*domain.SourceProvenance, *time.Time, *time.Time) {
	var source *domain.SourceProvenance
	var updatedAt *time.Time
	var dataAsOf *time.Time
	if instrument != nil {
		value := instrument.Source
		source = &value
		updatedAt = latestTime(updatedAt, instrument.UpdatedAt)
		dataAsOf = latestPointer(dataAsOf, instrument.DataAsOf)
	}
	for index := range bars {
		bar := bars[index]
		updatedAt = latestTime(updatedAt, bar.UpdatedAt)
		dataAsOf = latestTime(dataAsOf, bar.TradeDate)
	}
	if instrument == nil {
		var sourceUpdatedAt time.Time
		for index := range bars {
			bar := bars[index]
			if source == nil || bar.UpdatedAt.After(sourceUpdatedAt) {
				value := bar.Source
				source = &value
				sourceUpdatedAt = bar.UpdatedAt
			}
		}
	}
	return source, updatedAt, dataAsOf
}

func latestTime(current *time.Time, candidate time.Time) *time.Time {
	if candidate.IsZero() || (current != nil && !candidate.After(*current)) {
		return current
	}
	value := candidate
	return &value
}

func latestPointer(current, candidate *time.Time) *time.Time {
	if candidate == nil {
		return current
	}
	return latestTime(current, *candidate)
}
