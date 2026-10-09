package types

import (
	"fmt"
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

// Valid reports whether the date was constructed from a valid supported calendar date.
func (date TradingDate) Valid() bool {
	_, err := ParseTradingDate(date.value)
	return err == nil
}

// String returns the canonical date or an explicit marker for the invalid zero value.
func (date TradingDate) String() string {
	if !date.Valid() {
		return "<invalid-trading-date>"
	}
	return date.value
}

// DatabaseString returns the YYYY-MM-DD representation for a SQL DATE column.
func (date TradingDate) DatabaseString() (string, error) {
	if !date.Valid() {
		return "", fmt.Errorf("format MySQL DATE: invalid trading date")
	}
	return date.value, nil
}

// TushareString returns the YYYYMMDD representation expected by Tushare.
func (date TradingDate) TushareString() (string, error) {
	if !date.Valid() {
		return "", fmt.Errorf("format Tushare date: invalid trading date")
	}
	return date.value[:4] + date.value[5:7] + date.value[8:10], nil
}
