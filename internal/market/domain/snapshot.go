package domain

import "stock-quant/internal/shared/types"

// SnapshotReference locates an immutable local market-data file.
type SnapshotReference struct {
	Kind   string
	Path   string
	SHA256 string
	Format string
}

// Snapshot identifies the market-data state used for one trading date.
type Snapshot struct {
	AsOf         types.TradingDate
	SnapshotHash string
	Reference    SnapshotReference
}
