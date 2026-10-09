package domain

import (
	"time"

	"stock-quant/internal/shared/types"
)

// Stock describes the stable identity and listing interval of a security.
type Stock struct {
	TSCode     string
	Name       string
	Exchange   string
	Market     string
	ListDate   types.TradingDate
	DelistDate *types.TradingDate
	ListStatus string
	UpdatedAt  time.Time
}

// ListedOn 判断日期是否处于股票的上市区间内，上市日和退市日均包含在内。
func (stock Stock) ListedOn(date types.TradingDate) bool {
	if !date.Valid() || !stock.ListDate.Valid() || date.String() < stock.ListDate.String() {
		return false
	}
	if stock.DelistDate == nil {
		return true
	}
	return stock.DelistDate.Valid() && date.String() <= stock.DelistDate.String()
}

// TradeCalendar describes whether an exchange was open on a calendar date.
type TradeCalendar struct {
	Exchange     string
	Date         types.TradingDate
	IsOpen       bool
	PretradeDate *types.TradingDate
}

// DailyPrice stores an unadjusted daily bar, with amount in yuan and volume in lots.
type DailyPrice struct {
	TSCode     string
	TradeDate  types.TradingDate
	Open       types.Decimal
	High       types.Decimal
	Low        types.Decimal
	Close      types.Decimal
	AmountYuan types.AmountYuan
	VolumeLot  types.Decimal
	SourceHash string
	Revision   int
	FetchedAt  time.Time
}

// AdjFactor stores the provider's positive adjustment factor for one security date.
type AdjFactor struct {
	TSCode     string
	TradeDate  types.TradingDate
	Factor     types.Decimal
	SourceHash string
	Revision   int
	FetchedAt  time.Time
}
