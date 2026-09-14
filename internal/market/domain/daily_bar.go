// Package domain 定义行情领域模型。
package domain

import (
	"errors"
	"strings"
)

// DailyBar 是某只股票某个交易日的 OHLCV 数据。
type DailyBar struct {
	InstrumentCode string
	TradeDate      string
	Open           string
	High           string
	Low            string
	Close          string
	Volume         int64
}

// Validate 检查 DailyBar 的最小领域约束。
func (bar DailyBar) Validate() error {
	if strings.TrimSpace(bar.InstrumentCode) == "" {
		return errors.New("daily bar instrument code is required")
	}
	if strings.TrimSpace(bar.TradeDate) == "" {
		return errors.New("daily bar trade date is required")
	}
	for name, value := range map[string]string{
		"open": bar.Open, "high": bar.High, "low": bar.Low, "close": bar.Close,
	} {
		if strings.TrimSpace(value) == "" {
			return errors.New("daily bar " + name + " price is required")
		}
	}
	if bar.Volume < 0 {
		return errors.New("daily bar volume must not be negative")
	}
	return nil
}
