package domain

import (
	"fmt"
	"time"
)

type Instrument struct {
	Symbol    string
	Market    string
	Name      string
	Status    string
	Source    SourceProvenance
	DataAsOf  *time.Time
	UpdatedAt time.Time
}

func (instrument Instrument) Validate() error {
	if instrument.Symbol == "" || instrument.Name == "" || instrument.Status == "" {
		return fmt.Errorf("instrument fields: %w", ErrInvalidTask)
	}
	if instrument.Market != "SH" && instrument.Market != "SZ" && instrument.Market != "BJ" {
		return fmt.Errorf("instrument market: %w", ErrInvalidTask)
	}
	if !instrument.Source.Valid() {
		return fmt.Errorf("instrument source: %w", ErrInvalidTask)
	}
	return nil
}

type DailyBar struct {
	Symbol    string
	Market    string
	TradeDate time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	Source    SourceProvenance
	UpdatedAt time.Time
}

func (bar DailyBar) Validate() error {
	if bar.Symbol == "" || bar.TradeDate.IsZero() || bar.Open < 0 || bar.High < 0 ||
		bar.Low < 0 || bar.Close < 0 || bar.Volume < 0 {
		return fmt.Errorf("daily bar fields: %w", ErrInvalidTask)
	}
	if bar.Market != "SH" && bar.Market != "SZ" && bar.Market != "BJ" {
		return fmt.Errorf("daily bar market: %w", ErrInvalidTask)
	}
	if bar.Low > bar.High || bar.Open > bar.High || bar.Open < bar.Low ||
		bar.Close > bar.High || bar.Close < bar.Low {
		return fmt.Errorf("daily bar prices: %w", ErrInvalidTask)
	}
	if !bar.Source.Valid() {
		return fmt.Errorf("daily bar source: %w", ErrInvalidTask)
	}
	return nil
}

type BasicInfoBatch struct {
	Items    []Instrument
	DataAsOf *time.Time
}

func (batch BasicInfoBatch) Validate() error {
	for _, item := range batch.Items {
		if err := item.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type DailyBarBatch struct {
	Items    []DailyBar
	DataAsOf *time.Time
}

func (batch DailyBarBatch) Validate(dateRange DateRange) error {
	for _, item := range batch.Items {
		if err := item.Validate(); err != nil {
			return err
		}
		if !dateRange.Contains(item.TradeDate) {
			return fmt.Errorf("daily bar date: %w", ErrInvalidDateRange)
		}
	}
	return nil
}
