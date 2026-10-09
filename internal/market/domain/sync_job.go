package domain

import (
	"encoding/json"
	"errors"
	"time"

	"stock-quant/internal/shared/types"
)

var ErrInvalidSyncJobTransition = errors.New("invalid sync job status transition")

// SyncJobStatus is the persisted state of one idempotent market-data task.
type SyncJobStatus string

const (
	SyncJobPending SyncJobStatus = "PENDING"
	SyncJobRunning SyncJobStatus = "RUNNING"
	SyncJobSuccess SyncJobStatus = "SUCCESS"
	SyncJobFailed  SyncJobStatus = "FAILED"
	SyncJobBlocked SyncJobStatus = "BLOCKED"
)

// SyncJob is one request to synchronize a provider endpoint and optional market date.
type SyncJob struct {
	JobID         string
	TaskKey       string
	APIName       string
	TradeDate     *types.TradingDate
	Status        SyncJobStatus
	Attempts      int
	RequestedAt   time.Time
	ExpectedRows  *int
	ReceivedRows  *int
	QualityReport json.RawMessage
	LastError     *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
