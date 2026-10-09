package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"stock-quant/internal/shared/types"
)

// RunStatus is the persisted state of one backtest execution.
type RunStatus string

const (
	RunPending RunStatus = "PENDING"
	RunRunning RunStatus = "RUNNING"
	RunSuccess RunStatus = "SUCCESS"
	RunFailed  RunStatus = "FAILED"
	RunBlocked RunStatus = "BLOCKED"
)

var ErrInvalidStatusTransition = errors.New("invalid backtest run status transition")

// Run describes an immutable backtest request and its persisted outcome.
type Run struct {
	RunID           string
	RunKey          string
	StrategyID      string
	StrategyVersion string
	StartDate       types.TradingDate
	EndDate         types.TradingDate
	ConfigHash      string
	SnapshotHash    string
	Mode            string
	Status          RunStatus
	Metrics         json.RawMessage
	Risk            json.RawMessage
	ErrorMessage    *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CanonicalRunKey hashes the immutable inputs that identify a backtest.
func CanonicalRunKey(run Run) string {
	identity := struct {
		StrategyID      string `json:"strategy_id"`
		StrategyVersion string `json:"strategy_version"`
		StartDate       string `json:"start_date"`
		EndDate         string `json:"end_date"`
		ConfigHash      string `json:"config_hash"`
		SnapshotHash    string `json:"snapshot_hash"`
		Mode            string `json:"mode"`
	}{
		StrategyID: run.StrategyID, StrategyVersion: run.StrategyVersion,
		StartDate: run.StartDate.String(), EndDate: run.EndDate.String(),
		ConfigHash: run.ConfigHash, SnapshotHash: run.SnapshotHash, Mode: run.Mode,
	}
	encoded, _ := json.Marshal(identity)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}
