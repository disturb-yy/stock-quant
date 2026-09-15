package domain

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// RankingMetric 是股票排行支持的指标。
type RankingMetric string

const (
	RankingMetricGain           RankingMetric = "gain"
	RankingMetricLoss           RankingMetric = "loss"
	RankingMetricTurnoverAmount RankingMetric = "turnover_amount"
	RankingMetricTurnoverRate   RankingMetric = "turnover_rate"
)

// RankingObservation 是一只股票在同一观察点的排行原始数据。
type RankingObservation struct {
	InstrumentCode string
	InstrumentName string
	CurrentClose   string
	PreviousClose  string
	TurnoverAmount string
	TurnoverRate   string
}

// RankingResult 是排序后可供 Application 层转换为 API 的结果。
type RankingResult struct {
	Rank           int
	InstrumentCode string
	InstrumentName string
	Value          string
	Close          string
	Change         string
	ChangePercent  string
	TurnoverAmount string
	TurnoverRate   string
}

// NormalizeRankingMetric 将公开请求参数归一化为稳定的四类指标。
func NormalizeRankingMetric(value string) (RankingMetric, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "gain":
		return RankingMetricGain, nil
	case "loss":
		return RankingMetricLoss, nil
	case "turnover_amount":
		return RankingMetricTurnoverAmount, nil
	case "turnover_rate":
		return RankingMetricTurnoverRate, nil
	default:
		return "", fmt.Errorf("unsupported ranking metric %q", value)
	}
}

// RankObservations 按指标排序，并用股票代码作为稳定的二级排序键。
func RankObservations(metric RankingMetric, observations []RankingObservation) ([]RankingResult, error) {
	metric, err := NormalizeRankingMetric(string(metric))
	if err != nil {
		return nil, err
	}
	results := make([]RankingResult, 0, len(observations))
	seenCodes := make(map[string]struct{}, len(observations))
	for _, observation := range observations {
		result, value, err := buildRankingResult(metric, observation)
		if err != nil {
			return nil, fmt.Errorf("build ranking result for %q: %w", observation.InstrumentCode, err)
		}
		if _, exists := seenCodes[observation.InstrumentCode]; exists {
			return nil, fmt.Errorf("duplicate ranking instrument %q", observation.InstrumentCode)
		}
		seenCodes[observation.InstrumentCode] = struct{}{}
		result.Value = value
		results = append(results, result)
	}

	sort.SliceStable(results, func(left, right int) bool {
		leftValue := mustDecimal(results[left].Value)
		rightValue := mustDecimal(results[right].Value)
		comparison := leftValue.Cmp(rightValue)
		if comparison != 0 {
			if metric == RankingMetricLoss {
				return comparison < 0
			}
			return comparison > 0
		}
		return results[left].InstrumentCode < results[right].InstrumentCode
	})
	for index := range results {
		results[index].Rank = index + 1
	}
	return results, nil
}

func buildRankingResult(metric RankingMetric, observation RankingObservation) (RankingResult, string, error) {
	if strings.TrimSpace(observation.InstrumentCode) == "" || strings.TrimSpace(observation.InstrumentName) == "" {
		return RankingResult{}, "", errors.New("instrument identity is required")
	}
	current, err := positiveRankingDecimal(observation.CurrentClose, "current close")
	if err != nil {
		return RankingResult{}, "", err
	}
	previous, err := positiveRankingDecimal(observation.PreviousClose, "previous close")
	if err != nil {
		return RankingResult{}, "", err
	}
	change := new(big.Rat).Sub(current, previous)
	changePercent := new(big.Rat).Quo(change, previous)
	changePercent.Mul(changePercent, big.NewRat(100, 1))
	value := formatRankingDecimal(changePercent)
	if metric == RankingMetricTurnoverAmount {
		if _, err := nonNegativeRankingDecimal(observation.TurnoverAmount, "turnover amount"); err != nil {
			return RankingResult{}, "", err
		}
		value = strings.TrimSpace(observation.TurnoverAmount)
	}
	if metric == RankingMetricTurnoverRate {
		if _, err := nonNegativeRankingDecimal(observation.TurnoverRate, "turnover rate"); err != nil {
			return RankingResult{}, "", err
		}
		value = strings.TrimSpace(observation.TurnoverRate)
	}
	return RankingResult{
		InstrumentCode: observation.InstrumentCode,
		InstrumentName: observation.InstrumentName,
		Close:          strings.TrimSpace(observation.CurrentClose),
		Change:         formatRankingDecimal(change),
		ChangePercent:  formatRankingDecimal(changePercent),
		TurnoverAmount: strings.TrimSpace(observation.TurnoverAmount),
		TurnoverRate:   strings.TrimSpace(observation.TurnoverRate),
	}, value, nil
}

func positiveRankingDecimal(value, label string) (*big.Rat, error) {
	parsed, ok := new(big.Rat).SetString(strings.TrimSpace(value))
	if !ok || parsed.Sign() <= 0 {
		return nil, fmt.Errorf("%s must be a positive decimal", label)
	}
	return parsed, nil
}

func nonNegativeRankingDecimal(value, label string) (*big.Rat, error) {
	parsed, ok := new(big.Rat).SetString(strings.TrimSpace(value))
	if !ok || parsed.Sign() < 0 {
		return nil, fmt.Errorf("%s must be a non-negative decimal", label)
	}
	return parsed, nil
}

func mustDecimal(value string) *big.Rat {
	parsed, _ := new(big.Rat).SetString(value)
	return parsed
}

func formatRankingDecimal(value *big.Rat) string {
	scaled := new(big.Rat).Mul(value, big.NewRat(100, 1))
	quotient, remainder := new(big.Int).QuoRem(scaled.Num(), scaled.Denom(), new(big.Int))
	if remainder.Sign() != 0 {
		twiceRemainder := new(big.Int).Lsh(new(big.Int).Abs(remainder), 1)
		if twiceRemainder.Cmp(scaled.Denom()) >= 0 {
			if scaled.Num().Sign() < 0 {
				quotient.Sub(quotient, big.NewInt(1))
			} else {
				quotient.Add(quotient, big.NewInt(1))
			}
		}
	}
	sign := ""
	if quotient.Sign() < 0 {
		sign = "-"
	}
	absolute := new(big.Int).Abs(quotient)
	major := new(big.Int).Quo(absolute, big.NewInt(100))
	minor := new(big.Int).Mod(absolute, big.NewInt(100))
	return fmt.Sprintf("%s%s.%02d", sign, major.String(), minor.Int64())
}
