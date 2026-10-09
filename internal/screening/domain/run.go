package domain

import "stock-quant/internal/shared/types"

// RunMetadata identifies the strategy and immutable snapshot used by a screen run.
type RunMetadata struct {
	RunID           string
	AsOf            types.TradingDate
	StrategyID      string
	StrategyVersion string
	SnapshotHash    string
}
