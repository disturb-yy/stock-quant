package domain

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"
)

// ValuationRange 复用股票财务查询的 3y/5y 范围值对象。
type ValuationRange = FinancialRange

// ValuationMetricObservation 是单项估值在一个观察日的原始快照。
type ValuationMetricObservation struct {
	AsOf  string
	Value *string
	Basis string
}

// ValuationObservation 是同一股票、同一观察日的三项估值快照。
type ValuationObservation struct {
	InstrumentCode string
	AsOf           string
	PETTM          ValuationMetricObservation
	PB             ValuationMetricObservation
	PSTTM          ValuationMetricObservation
}

// Validate 检查估值快照的身份、日期和 nullable 数值口径。
func (observation ValuationObservation) Validate() error {
	if strings.TrimSpace(observation.InstrumentCode) == "" {
		return errors.New("valuation instrument code is required")
	}
	if _, err := time.Parse("2006-01-02", observation.AsOf); err != nil {
		return fmt.Errorf("valuation observation date is invalid: %w", err)
	}
	for name, metric := range map[string]ValuationMetricObservation{
		"pe_ttm": observation.PETTM,
		"pb":     observation.PB,
		"ps_ttm": observation.PSTTM,
	} {
		if metric.AsOf != observation.AsOf {
			return fmt.Errorf("valuation %s date does not match snapshot date", name)
		}
		if metric.Value != nil && strings.TrimSpace(*metric.Value) == "" {
			return fmt.Errorf("valuation %s value cannot be blank", name)
		}
		if metric.Value != nil && strings.TrimSpace(metric.Basis) == "" {
			return fmt.Errorf("valuation %s basis is required when value is present", name)
		}
	}
	return nil
}

// ValuationCurrent 是当前指标及其实际日期、口径。
type ValuationCurrent struct {
	Value *string
	AsOf  *string
	Basis *string
}

// ValuationHistoryPoint 是用于历史分布的正数观察点。
type ValuationHistoryPoint struct {
	AsOf  string
	Value string
}

// ValuationPercentile 是 inclusive rank 分位结果。
type ValuationPercentile struct {
	Value      *string
	SampleSize int
	RangeFrom  *string
	RangeTo    *string
	Method     string
}

// ValuationPosition 是历史分布位置，不代表投资评级。
type ValuationPosition string

const (
	PositionLow    ValuationPosition = "low"
	PositionMiddle ValuationPosition = "middle"
	PositionHigh   ValuationPosition = "high"
)

// ValuationMetricResult 是一项估值的领域计算结果。
type ValuationMetricResult struct {
	Current    ValuationCurrent
	History    []ValuationHistoryPoint
	Percentile ValuationPercentile
	Position   *ValuationPosition
}

// AnalyzeValuationMetric 按当前值的口径计算历史正数样本和 inclusive rank。
func AnalyzeValuationMetric(observations []ValuationMetricObservation, from, to string) (ValuationMetricResult, error) {
	if err := validateDateRange(from, to); err != nil {
		return ValuationMetricResult{}, err
	}
	sorted := append([]ValuationMetricObservation(nil), observations...)
	sort.SliceStable(sorted, func(left, right int) bool { return sorted[left].AsOf < sorted[right].AsOf })
	current, err := latestMetricObservation(sorted)
	if err != nil {
		return ValuationMetricResult{}, err
	}
	result := ValuationMetricResult{
		History: make([]ValuationHistoryPoint, 0),
		Percentile: ValuationPercentile{
			SampleSize: 0,
			Method:     "inclusive_rank",
		},
	}
	if current == nil {
		return result, nil
	}
	currentRat, err := parseValuationValue(*current.Value)
	if err != nil {
		return ValuationMetricResult{}, fmt.Errorf("parse current valuation: %w", err)
	}
	formattedCurrent := formatValuationValue(currentRat)
	currentAsOf, currentBasis := current.AsOf, current.Basis
	result.Current = ValuationCurrent{Value: &formattedCurrent, AsOf: &currentAsOf, Basis: &currentBasis}
	for _, observation := range sorted {
		if !inDateRange(observation.AsOf, from, to) || observation.Basis != current.Basis || observation.Value == nil {
			continue
		}
		value, err := parseValuationValue(*observation.Value)
		if err != nil {
			return ValuationMetricResult{}, fmt.Errorf("parse valuation on %s: %w", observation.AsOf, err)
		}
		if value.Sign() <= 0 {
			continue
		}
		result.History = append(result.History, ValuationHistoryPoint{AsOf: observation.AsOf, Value: formatValuationValue(value)})
	}
	if len(result.History) == 0 || currentRat.Sign() <= 0 {
		return result, nil
	}
	result.Percentile.SampleSize = len(result.History)
	fromDate, toDate := result.History[0].AsOf, result.History[len(result.History)-1].AsOf
	result.Percentile.RangeFrom = &fromDate
	result.Percentile.RangeTo = &toDate
	lessOrEqual := 0
	for _, point := range result.History {
		value, err := parseValuationValue(point.Value)
		if err != nil {
			return ValuationMetricResult{}, fmt.Errorf("parse formatted valuation on %s: %w", point.AsOf, err)
		}
		if value.Cmp(currentRat) <= 0 {
			lessOrEqual++
		}
	}
	percentile := new(big.Rat).SetFrac(big.NewInt(int64(lessOrEqual*100)), big.NewInt(int64(len(result.History))))
	formattedPercentile := formatValuationValue(percentile)
	result.Percentile.Value = &formattedPercentile
	position := positionForPercentile(percentile)
	result.Position = &position
	return result, nil
}

func latestMetricObservation(observations []ValuationMetricObservation) (*ValuationMetricObservation, error) {
	for index := len(observations) - 1; index >= 0; index-- {
		observation := observations[index]
		if _, err := time.Parse("2006-01-02", observation.AsOf); err != nil {
			return nil, fmt.Errorf("valuation observation date is invalid: %w", err)
		}
		if observation.Value == nil {
			continue
		}
		if strings.TrimSpace(observation.Basis) == "" {
			return nil, errors.New("valuation basis is required when current value is present")
		}
		if strings.TrimSpace(*observation.Value) == "" {
			return nil, errors.New("valuation value cannot be blank")
		}
		copy := observation
		return &copy, nil
	}
	for _, observation := range observations {
		if _, err := time.Parse("2006-01-02", observation.AsOf); err != nil {
			return nil, fmt.Errorf("valuation observation date is invalid: %w", err)
		}
	}
	return nil, nil
}

func validateDateRange(from, to string) error {
	if from == "" && to == "" {
		return nil
	}
	fromDate, err := time.Parse("2006-01-02", from)
	if err != nil {
		return fmt.Errorf("valuation range from is invalid: %w", err)
	}
	toDate, err := time.Parse("2006-01-02", to)
	if err != nil {
		return fmt.Errorf("valuation range to is invalid: %w", err)
	}
	if fromDate.After(toDate) {
		return errors.New("valuation range is reversed")
	}
	return nil
}

func inDateRange(value, from, to string) bool {
	return (from == "" || value >= from) && (to == "" || value <= to)
}

func parseValuationValue(value string) (*big.Rat, error) {
	parsed, ok := new(big.Rat).SetString(strings.TrimSpace(value))
	if !ok {
		return nil, fmt.Errorf("%q is not a decimal", value)
	}
	return parsed, nil
}

func formatValuationValue(value *big.Rat) string {
	return value.FloatString(2)
}

func positionForPercentile(value *big.Rat) ValuationPosition {
	if value.Cmp(big.NewRat(30, 1)) < 0 {
		return PositionLow
	}
	if value.Cmp(big.NewRat(70, 1)) <= 0 {
		return PositionMiddle
	}
	return PositionHigh
}

// ValuationPeerObservation 是一个行业成员在某日的单项估值。
type ValuationPeerObservation struct {
	InstrumentCode string
	AsOf           string
	Value          *string
	Basis          string
}

// ValuationMedian 是同日同行中位数及其有效样本数。
type ValuationMedian struct {
	Value      *string
	SampleSize int
}

// CalculateIndustryMedian 计算同日、同口径、排除目标股票后的正数中位数。
func CalculateIndustryMedian(values []ValuationPeerObservation, targetCode, asOf, basis string, targetCurrent *string) (ValuationMedian, error) {
	result := ValuationMedian{}
	var target *big.Rat
	if targetCurrent != nil {
		parsed, err := parseValuationValue(*targetCurrent)
		if err != nil {
			return result, fmt.Errorf("parse target valuation: %w", err)
		}
		target = parsed
	}
	positive := make([]*big.Rat, 0, len(values))
	for _, value := range values {
		if value.InstrumentCode == targetCode || value.AsOf != asOf || value.Basis != basis || value.Value == nil {
			continue
		}
		parsed, err := parseValuationValue(*value.Value)
		if err != nil {
			return ValuationMedian{}, fmt.Errorf("parse peer valuation: %w", err)
		}
		if parsed.Sign() > 0 {
			positive = append(positive, parsed)
		}
	}
	result.SampleSize = len(positive)
	if target == nil || target.Sign() <= 0 || len(positive) < 2 {
		return result, nil
	}
	sort.Slice(positive, func(left, right int) bool { return positive[left].Cmp(positive[right]) < 0 })
	var median *big.Rat
	middle := len(positive) / 2
	if len(positive)%2 == 1 {
		median = positive[middle]
	} else {
		median = new(big.Rat).Quo(new(big.Rat).Add(positive[middle-1], positive[middle]), big.NewRat(2, 1))
	}
	formatted := formatValuationValue(median)
	result.Value = &formatted
	return result, nil
}
