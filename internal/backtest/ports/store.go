package ports

import (
	"context"
	"time"

	"stock-quant/internal/backtest/domain"
)

// BacktestStore persists idempotent run metadata and terminal outcome.
type BacktestStore interface {
	CreateOrGet(ctx context.Context, run domain.Run) (domain.Run, bool, error)
	MarkRunning(ctx context.Context, runID string, at time.Time) error
	MarkFailed(ctx context.Context, runID, code, message string, at time.Time) error
	MarkBlocked(ctx context.Context, runID, code, message string, at time.Time) error
	Complete(ctx context.Context, runID string, metrics, risk []byte, at time.Time) error
	Find(ctx context.Context, runID string) (domain.Run, error)
}
