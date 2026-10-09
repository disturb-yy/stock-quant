// Package ports defines screening persistence dependencies.
package ports

import (
	"context"
	"time"

	"stock-quant/internal/screening/domain"
)

// ScreeningStore persists idempotent screening runs and atomic result outcomes.
type ScreeningStore interface {
	CreateOrGet(ctx context.Context, run domain.RunMetadata) (domain.RunMetadata, bool, error)
	MarkRunning(ctx context.Context, runID string, at time.Time) error
	MarkFailed(ctx context.Context, runID, code, message string, at time.Time) error
	MarkBlocked(ctx context.Context, runID, code, message string, at time.Time) error
	Complete(ctx context.Context, runID string, outcome domain.ScreeningOutcome, at time.Time) error
	Find(ctx context.Context, runID string) (domain.RunMetadata, error)
	ListResults(ctx context.Context, runID string, limit, offset int) ([]domain.ScreeningResult, error)
}
