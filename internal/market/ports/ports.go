// Package ports defines the market domain's application-facing dependencies.
package ports

import (
	"context"

	"stock-quant/internal/market/domain"
	"stock-quant/internal/shared/types"
)

// SnapshotRepository loads an immutable, validated market snapshot.
type SnapshotRepository interface {
	Load(ctx context.Context, asOf types.TradingDate, lookback int) (domain.Snapshot, error)
}

// MarketDataProvider fetches market trading dates from an upstream provider.
type MarketDataProvider interface {
	TradingDates(ctx context.Context, from, through types.TradingDate) ([]types.TradingDate, error)
}

// StockRepository persists security identity and supports historical lookup regardless of current listing status.
type StockRepository interface {
	Upsert(ctx context.Context, stocks []domain.Stock) error
	Find(ctx context.Context, tsCode string) (domain.Stock, error)
}

// TradeCalendarRepository persists and reads exchange calendar ranges.
type TradeCalendarRepository interface {
	Upsert(ctx context.Context, dates []domain.TradeCalendar) error
	List(ctx context.Context, exchange string, from, through types.TradingDate) ([]domain.TradeCalendar, error)
}

// DailyPriceRepository persists unadjusted daily bars and reads deterministic ranges.
type DailyPriceRepository interface {
	Upsert(ctx context.Context, prices []domain.DailyPrice) error
	ListByCode(ctx context.Context, tsCode string, from, through types.TradingDate) ([]domain.DailyPrice, error)
	ListByDate(ctx context.Context, date types.TradingDate) ([]domain.DailyPrice, error)
}

// AdjFactorRepository persists and reads adjustment-factor ranges.
type AdjFactorRepository interface {
	Upsert(ctx context.Context, factors []domain.AdjFactor) error
	ListByCode(ctx context.Context, tsCode string, from, through types.TradingDate) ([]domain.AdjFactor, error)
	ListByDate(ctx context.Context, date types.TradingDate) ([]domain.AdjFactor, error)
}
