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
