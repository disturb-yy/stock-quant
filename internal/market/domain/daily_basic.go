package domain

import (
	"errors"
	"strings"
)

// DailyBasic 是某只股票某个交易日的估值与规模快照。
type DailyBasic struct {
	InstrumentCode string
	TradeDate      string
	MarketCap      string
	PB             string
}

// Validate 检查 daily basic 的最小领域约束。
func (basic DailyBasic) Validate() error {
	if strings.TrimSpace(basic.InstrumentCode) == "" {
		return errors.New("daily basic instrument code is required")
	}
	if strings.TrimSpace(basic.TradeDate) == "" {
		return errors.New("daily basic trade date is required")
	}
	if strings.TrimSpace(basic.MarketCap) == "" && strings.TrimSpace(basic.PB) == "" {
		return errors.New("daily basic must contain market cap or pb")
	}
	return nil
}
