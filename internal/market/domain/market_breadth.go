package domain

import (
	"errors"
	"strings"
)

// MarketBreadth 是某个交易日的市场宽度和成交额统计。
type MarketBreadth struct {
	TradeDate      string
	Advancing      int64
	Declining      int64
	Unchanged      int64
	TurnoverAmount string
}

// Validate 检查市场宽度统计的最小领域约束。
func (breadth MarketBreadth) Validate() error {
	if strings.TrimSpace(breadth.TradeDate) == "" {
		return errors.New("market breadth trade date is required")
	}
	if breadth.Advancing < 0 || breadth.Declining < 0 || breadth.Unchanged < 0 {
		return errors.New("market breadth counts must not be negative")
	}
	if strings.TrimSpace(breadth.TurnoverAmount) == "" {
		return errors.New("market breadth turnover amount is required")
	}
	return nil
}
