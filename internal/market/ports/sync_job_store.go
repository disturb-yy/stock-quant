package ports

import (
	"context"
	"encoding/json"
	"time"

	"stock-quant/internal/market/domain"
)

// SyncJobStore persists idempotent task identity and guarded state transitions.
type SyncJobStore interface {
	CreateOrGet(ctx context.Context, job domain.SyncJob) (domain.SyncJob, bool, error)
	MarkRunning(ctx context.Context, jobID string, at time.Time) error
	MarkSucceeded(ctx context.Context, jobID string, receivedRows int, quality json.RawMessage, at time.Time) error
	MarkFailed(ctx context.Context, jobID, code, message string, at time.Time) error
	MarkBlocked(ctx context.Context, jobID, code, message string, at time.Time) error
	Find(ctx context.Context, jobID string) (domain.SyncJob, error)
}
