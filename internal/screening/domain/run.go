package domain

import (
	"encoding/json"
	"errors"
	"time"

	"stock-quant/internal/shared/types"
)

// RunStatus is the persisted state of one screening execution.
type RunStatus string

const (
	RunPending RunStatus = "PENDING"
	RunRunning RunStatus = "RUNNING"
	RunSuccess RunStatus = "SUCCESS"
	RunFailed  RunStatus = "FAILED"
	RunBlocked RunStatus = "BLOCKED"
)

var ErrInvalidStatusTransition = errors.New("invalid screening run status transition")

// RunMetadata identifies the strategy and immutable snapshot used by a screen run.
type RunMetadata struct {
	RunID            string
	RunKey           string
	AsOf             types.TradingDate
	StrategyID       string
	StrategyVersion  string
	ConfigHash       string
	SnapshotHash     string
	Status           RunStatus
	TotalUniverse    int
	TotalEligible    int
	ErrorCode        *string
	ErrorMessage     *string
	RejectionSummary json.RawMessage
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ScreeningResult is one persisted candidate, including raw factors and audit reasons.
type ScreeningResult struct {
	RunID        string
	TSCode       string
	FinalRank    *int
	TotalScore   *float64
	RawFactors   json.RawMessage
	FactorScores json.RawMessage
	Reasons      json.RawMessage
}

// ScreeningOutcome is committed together with the successful run state.
type ScreeningOutcome struct {
	TotalUniverse    int
	TotalEligible    int
	RejectionSummary json.RawMessage
	Results          []ScreeningResult
}
