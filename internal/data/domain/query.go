package domain

import (
	"context"
	"strings"
	"time"
)

const DefaultRecentTradingDays = 30

type StockIdentifier struct {
	Symbol string
	Market string
}

func (identifier StockIdentifier) Qualified() string {
	return identifier.Symbol + "." + identifier.Market
}

func ParseStockIdentifier(raw string) (StockIdentifier, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 2 || len(parts[0]) != 6 || len(parts[1]) != 2 || !digitsOnly(parts[0]) {
		return StockIdentifier{}, ErrInvalidStockSymbol
	}
	market := parts[1]
	if market != "SH" && market != "SZ" && market != "BJ" {
		return StockIdentifier{}, ErrInvalidStockSymbol
	}
	if !knownMarketCode(market, parts[0]) {
		return StockIdentifier{}, ErrStockNotFound
	}
	return StockIdentifier{Symbol: parts[0], Market: market}, nil
}

func ParseOptionalDateRange(start, end *string) (DateRange, error) {
	if start == nil && end == nil {
		return DateRange{}, nil
	}
	if start == nil || end == nil || *start == "" || *end == "" {
		return DateRange{}, ErrInvalidDateRange
	}
	return ParseDateRange(*start, *end)
}

type StockQueryResult struct {
	Symbol       string
	BasicInfo    *Instrument
	DailyBars    []DailyBar
	Availability StockDataAvailability
	Source       *SourceProvenance
	UpdatedAt    *time.Time
	DataAsOf     *time.Time
}

type StockDataAvailability struct {
	BasicInfo string
	DailyBars string
}

const (
	AvailabilityAvailable = "available"
	AvailabilityEmpty     = "empty"
)

type StockQueryRepository interface {
	FindInstrument(context.Context, StockIdentifier) (*Instrument, error)
	FindDailyBars(context.Context, StockIdentifier, DateRange, int) ([]DailyBar, error)
}

func digitsOnly(value string) bool {
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func knownMarketCode(market, symbol string) bool {
	if market == "SH" {
		return hasPrefix(symbol, "600", "601", "603", "605", "688", "689")
	}
	if market == "SZ" {
		return hasPrefix(symbol, "000", "001", "002", "003", "300", "301")
	}
	return symbol[0] == '4' || symbol[0] == '8' || symbol[0] == '9'
}

func hasPrefix(value string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}
