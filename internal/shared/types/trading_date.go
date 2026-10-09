package types

import (
	"fmt"
	"math"
	"time"
)

const tradingDateLayout = "2006-01-02"

// TradingDate is a calendar date without a time or time zone.
type TradingDate struct {
	value string
}

// ParseTradingDate parses a canonical YYYY-MM-DD date used by APIs and MySQL DATE.
func ParseTradingDate(value string) (TradingDate, error) {
	parsed, err := time.Parse(tradingDateLayout, value)
	if err != nil || parsed.Year() < 1 || parsed.Format(tradingDateLayout) != value {
		return TradingDate{}, fmt.Errorf("parse trading date %q: expected a valid YYYY-MM-DD date", value)
	}
	return TradingDate{value: value}, nil
}

// ParseTushareDate parses a Tushare YYYYMMDD trading date.
func ParseTushareDate(value string) (TradingDate, error) {
	if len(value) != 8 {
		return TradingDate{}, fmt.Errorf("parse Tushare trading date %q: expected YYYYMMDD", value)
	}
	parsed, err := time.Parse("20060102", value)
	if err != nil || parsed.Year() < 1 {
		return TradingDate{}, fmt.Errorf("parse Tushare trading date %q: invalid calendar date", value)
	}
	return ParseTradingDate(parsed.Format(tradingDateLayout))
}

// TradingDateFromTime extracts a calendar date using the caller's explicit location.
func TradingDateFromTime(value time.Time, location *time.Location) (TradingDate, error) {
	if location == nil {
		return TradingDate{}, fmt.Errorf("convert timestamp to trading date: location is required")
	}
	return ParseTradingDate(value.In(location).Format(tradingDateLayout))
}

// String returns the canonical YYYY-MM-DD representation, also accepted by MySQL DATE.
func (date TradingDate) String() string {
	return date.value
}

// DatabaseString returns the YYYY-MM-DD representation for a SQL DATE column.
func (date TradingDate) DatabaseString() string {
	return date.value
}

// TushareString returns the YYYYMMDD representation expected by Tushare.
func (date TradingDate) TushareString() string {
	if date.value == "" {
		return ""
	}
	return date.value[:4] + date.value[5:7] + date.value[8:10]
}

// AmountYuan stores a finite monetary value measured in Chinese yuan.
type AmountYuan struct {
	value float64
}

// NewAmountYuan constructs an amount without rounding or restricting its sign.
func NewAmountYuan(value float64) (AmountYuan, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return AmountYuan{}, fmt.Errorf("amount in yuan must be finite")
	}
	return AmountYuan{value: value}, nil
}

// Float64 returns the amount in yuan without changing its precision.
func (amount AmountYuan) Float64() float64 {
	return amount.value
}
