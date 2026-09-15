package domain

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// SignalType 是内置信号的稳定枚举。
type SignalType string

const (
	SignalVolumeSurge SignalType = "volume_surge"
	SignalBreakout    SignalType = "breakout"
	SignalNewHigh     SignalType = "new_high"
	SignalStrong      SignalType = "strong"
)

// SignalTypes 返回 API 可用的全部内置信号类型。
func SignalTypes() []SignalType {
	return []SignalType{SignalVolumeSurge, SignalBreakout, SignalNewHigh, SignalStrong}
}

// SignalRuleParameters 是领域规则使用的规范化参数。
type SignalRuleParameters struct {
	Window     int
	Multiple   *big.Rat
	TopPercent int
}

// SignalSeries 是一只股票按交易日升序排列的日行情。
type SignalSeries struct {
	InstrumentCode string
	InstrumentName string
	Bars           []DailyBar
}

// SignalMatch 是一只命中股票的领域结果。
type SignalMatch struct {
	InstrumentCode string
	InstrumentName string
	SignalType     SignalType
}

// ValidateSignalParameters 校验信号参数的领域约束。
func ValidateSignalParameters(signalType SignalType, params SignalRuleParameters) error {
	if !isSupportedSignalType(signalType) {
		return fmt.Errorf("unsupported signal type %q", signalType)
	}
	if params.Window != 20 && params.Window != 60 && params.Window != 120 {
		return fmt.Errorf("signal window must be one of 20, 60, 120")
	}
	switch signalType {
	case SignalVolumeSurge:
		if params.Multiple == nil || params.Multiple.Cmp(big.NewRat(0, 1)) <= 0 {
			return errors.New("volume surge multiple must be positive")
		}
		if params.TopPercent != 0 {
			return errors.New("volume surge does not accept top percent")
		}
	case SignalStrong:
		if params.TopPercent != 10 && params.TopPercent != 20 {
			return errors.New("strong top percent must be 10 or 20")
		}
		if params.Multiple != nil {
			return errors.New("strong does not accept multiple")
		}
	default:
		if params.Multiple != nil || params.TopPercent != 0 {
			return errors.New("signal type does not accept extra parameters")
		}
	}
	return nil
}

// ScanSignals 按最新一根日线计算指定的内置信号。
func ScanSignals(signalType SignalType, series []SignalSeries, params SignalRuleParameters) ([]SignalMatch, error) {
	if err := ValidateSignalParameters(signalType, params); err != nil {
		return nil, err
	}
	if len(series) == 0 {
		return nil, errors.New("signal series are required")
	}
	for _, item := range series {
		if err := validateSignalSeries(item, params.Window); err != nil {
			return nil, err
		}
	}
	if signalType == SignalStrong {
		return scanStrong(series, params)
	}

	matches := make([]SignalMatch, 0)
	for _, item := range series {
		matched, err := matchesSingle(signalType, item.Bars, params)
		if err != nil {
			return nil, fmt.Errorf("evaluate %q: %w", item.InstrumentCode, err)
		}
		if matched {
			matches = append(matches, SignalMatch{
				InstrumentCode: item.InstrumentCode,
				InstrumentName: item.InstrumentName,
				SignalType:     signalType,
			})
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].InstrumentCode < matches[j].InstrumentCode })
	return matches, nil
}

func isSupportedSignalType(signalType SignalType) bool {
	for _, supported := range SignalTypes() {
		if signalType == supported {
			return true
		}
	}
	return false
}

func validateSignalSeries(series SignalSeries, window int) error {
	if strings.TrimSpace(series.InstrumentCode) == "" || strings.TrimSpace(series.InstrumentName) == "" {
		return errors.New("signal series instrument identity is required")
	}
	if len(series.Bars) < window+1 {
		return fmt.Errorf("instrument %q has %d bars, want at least %d", series.InstrumentCode, len(series.Bars), window+1)
	}
	previousDate := ""
	for _, bar := range series.Bars {
		if bar.InstrumentCode != series.InstrumentCode {
			return fmt.Errorf("instrument %q contains bar for %q", series.InstrumentCode, bar.InstrumentCode)
		}
		if err := bar.Validate(); err != nil {
			return fmt.Errorf("validate bar %q: %w", series.InstrumentCode, err)
		}
		if previousDate != "" && bar.TradeDate <= previousDate {
			return fmt.Errorf("instrument %q bars are not in ascending date order", series.InstrumentCode)
		}
		previousDate = bar.TradeDate
	}
	return nil
}

func matchesSingle(signalType SignalType, bars []DailyBar, params SignalRuleParameters) (bool, error) {
	current := bars[len(bars)-1]
	previous := bars[len(bars)-params.Window-1 : len(bars)-1]
	switch signalType {
	case SignalVolumeSurge:
		return volumeSurge(current.Volume, previous, params.Multiple), nil
	case SignalBreakout:
		return priceBreaksLevel(current.Close, previous, func(bar DailyBar) string { return bar.High })
	case SignalNewHigh:
		return priceBreaksLevel(current.Close, previous, func(bar DailyBar) string { return bar.Close })
	default:
		return false, fmt.Errorf("unsupported single signal type %q", signalType)
	}
}

func volumeSurge(currentVolume int64, previous []DailyBar, multiple *big.Rat) bool {
	volumeTotal := new(big.Int)
	for _, bar := range previous {
		volumeTotal.Add(volumeTotal, big.NewInt(bar.Volume))
	}
	if volumeTotal.Sign() == 0 {
		return false
	}
	threshold := new(big.Rat).SetInt(volumeTotal)
	threshold.Quo(threshold, new(big.Rat).SetInt64(int64(len(previous))))
	threshold.Mul(threshold, multiple)
	return new(big.Rat).SetInt64(currentVolume).Cmp(threshold) >= 0
}

func priceBreaksLevel(current string, previous []DailyBar, value func(DailyBar) string) (bool, error) {
	currentPrice, err := positivePrice(current, "current price")
	if err != nil {
		return false, err
	}
	maximum := new(big.Rat)
	for index, bar := range previous {
		price, err := positivePrice(value(bar), "historical price")
		if err != nil {
			return false, err
		}
		if index == 0 || price.Cmp(maximum) > 0 {
			maximum = price
		}
	}
	return currentPrice.Cmp(maximum) > 0, nil
}

func positivePrice(value, label string) (*big.Rat, error) {
	price, ok := new(big.Rat).SetString(strings.TrimSpace(value))
	if !ok || price.Sign() <= 0 {
		return nil, fmt.Errorf("%s must be a positive decimal", label)
	}
	return price, nil
}

type strongCandidate struct {
	series     SignalSeries
	returnRate *big.Rat
}

func scanStrong(series []SignalSeries, params SignalRuleParameters) ([]SignalMatch, error) {
	candidates := make([]strongCandidate, 0, len(series))
	for _, item := range series {
		latest := item.Bars[len(item.Bars)-1]
		base := item.Bars[len(item.Bars)-params.Window-1]
		latestPrice, err := positivePrice(latest.Close, "latest close")
		if err != nil {
			return nil, fmt.Errorf("evaluate %q: %w", item.InstrumentCode, err)
		}
		basePrice, err := positivePrice(base.Close, "base close")
		if err != nil {
			return nil, fmt.Errorf("evaluate %q: %w", item.InstrumentCode, err)
		}
		returnRate := new(big.Rat).Quo(latestPrice, basePrice)
		returnRate.Sub(returnRate, big.NewRat(1, 1))
		candidates = append(candidates, strongCandidate{series: item, returnRate: returnRate})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].returnRate.Cmp(candidates[j].returnRate) == 0 {
			return candidates[i].series.InstrumentCode < candidates[j].series.InstrumentCode
		}
		return candidates[i].returnRate.Cmp(candidates[j].returnRate) > 0
	})
	count := (len(candidates)*params.TopPercent + 99) / 100
	if count < 1 {
		count = 1
	}
	matches := make([]SignalMatch, 0, count)
	for _, candidate := range candidates[:count] {
		matches = append(matches, SignalMatch{
			InstrumentCode: candidate.series.InstrumentCode,
			InstrumentName: candidate.series.InstrumentName,
			SignalType:     SignalStrong,
		})
	}
	return matches, nil
}
