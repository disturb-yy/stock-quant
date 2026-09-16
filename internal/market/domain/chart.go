package domain

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// ChartAdjustment 是日线价格的复权口径。
type ChartAdjustment string

const (
	AdjustmentNone ChartAdjustment = "none"
	AdjustmentQFQ  ChartAdjustment = "qfq"
	AdjustmentHFQ  ChartAdjustment = "hfq"
)

// AdjustmentFactor 是某只股票某个交易日的复权因子。
type AdjustmentFactor struct {
	InstrumentCode string
	TradeDate      string
	QFQFactor      string
	HFQFactor      string
}

// Validate 检查复权因子的最小领域约束。
func (factor AdjustmentFactor) Validate() error {
	if strings.TrimSpace(factor.InstrumentCode) == "" || strings.TrimSpace(factor.TradeDate) == "" {
		return errors.New("adjustment factor identity is required")
	}
	if strings.TrimSpace(factor.QFQFactor) == "" || strings.TrimSpace(factor.HFQFactor) == "" {
		return errors.New("adjustment factors are required")
	}
	for name, value := range map[string]string{"qfq": factor.QFQFactor, "hfq": factor.HFQFactor} {
		parsed, err := parseDecimal(value)
		if err != nil || parsed.Sign() <= 0 {
			return fmt.Errorf("%s adjustment factor must be positive decimal", name)
		}
	}
	return nil
}

// BarWithAdjustment 是带原始 OHLCV 与复权因子的日线。
type BarWithAdjustment struct {
	Bar       DailyBar
	QFQFactor string
	HFQFactor string
}

// ChartBar 是完成复权但尚未附加指标的日线。
type ChartBar struct {
	TradeDate string
	Open      string
	High      string
	Low       string
	Close     string
	Volume    int64
	MA5       *string
	MA20      *string
}

// BenchmarkBar 是基准指数的一条原始收盘点。
type BenchmarkBar struct {
	TradeDate string
	Close     string
}

// BenchmarkPoint 是与股票共同交易日对齐后的基准表现。
type BenchmarkPoint struct {
	TradeDate          string
	Close              string
	StockReturnPct     string
	BenchmarkReturnPct string
	RelativeReturnPct  string
}

// ApplyAdjustment 按指定口径调整 OHLC，成交量保持原值。
func ApplyAdjustment(source BarWithAdjustment, adjustment ChartAdjustment) (ChartBar, error) {
	factor := "1"
	switch adjustment {
	case AdjustmentNone:
	case AdjustmentQFQ:
		factor = source.QFQFactor
	case AdjustmentHFQ:
		factor = source.HFQFactor
	default:
		return ChartBar{}, fmt.Errorf("unsupported chart adjustment %q", adjustment)
	}
	parsedFactor, err := parseDecimal(factor)
	if err != nil || parsedFactor.Sign() <= 0 {
		return ChartBar{}, errors.New("adjustment factor is required and must be positive")
	}
	open, err := adjustMoney(source.Bar.Open, parsedFactor)
	if err != nil {
		return ChartBar{}, fmt.Errorf("adjust open price: %w", err)
	}
	high, err := adjustMoney(source.Bar.High, parsedFactor)
	if err != nil {
		return ChartBar{}, fmt.Errorf("adjust high price: %w", err)
	}
	low, err := adjustMoney(source.Bar.Low, parsedFactor)
	if err != nil {
		return ChartBar{}, fmt.Errorf("adjust low price: %w", err)
	}
	close, err := adjustMoney(source.Bar.Close, parsedFactor)
	if err != nil {
		return ChartBar{}, fmt.Errorf("adjust close price: %w", err)
	}
	return ChartBar{
		TradeDate: source.Bar.TradeDate,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    source.Bar.Volume,
	}, nil
}

// CalculateMovingAverages 为响应中的收盘价附加 MA5 与 MA20。
func CalculateMovingAverages(bars []ChartBar) error {
	for index := range bars {
		bars[index].MA5 = nil
		bars[index].MA20 = nil
		if index >= 4 {
			value, err := averageClose(bars[index-4 : index+1])
			if err != nil {
				return fmt.Errorf("calculate ma5 at %q: %w", bars[index].TradeDate, err)
			}
			bars[index].MA5 = &value
		}
		if index >= 19 {
			value, err := averageClose(bars[index-19 : index+1])
			if err != nil {
				return fmt.Errorf("calculate ma20 at %q: %w", bars[index].TradeDate, err)
			}
			bars[index].MA20 = &value
		}
	}
	return nil
}

// BuildBenchmarkPoints 按共同交易日计算股票、基准和相对收益。
func BuildBenchmarkPoints(stock []ChartBar, benchmark []BenchmarkBar) ([]BenchmarkPoint, error) {
	benchmarkByDate := make(map[string]string, len(benchmark))
	for _, point := range benchmark {
		if _, exists := benchmarkByDate[point.TradeDate]; exists {
			return nil, fmt.Errorf("duplicate benchmark trade date %q", point.TradeDate)
		}
		if _, err := parsePositiveDecimal(point.Close); err != nil {
			return nil, fmt.Errorf("benchmark close at %q: %w", point.TradeDate, err)
		}
		benchmarkByDate[point.TradeDate] = point.Close
	}
	common := make([]ChartBar, 0)
	for _, bar := range stock {
		if _, ok := benchmarkByDate[bar.TradeDate]; ok {
			common = append(common, bar)
		}
	}
	if len(common) == 0 {
		return []BenchmarkPoint{}, nil
	}
	stockBase, err := parsePositiveDecimal(common[0].Close)
	if err != nil {
		return nil, fmt.Errorf("stock close at %q: %w", common[0].TradeDate, err)
	}
	benchmarkBase, err := parsePositiveDecimal(benchmarkByDate[common[0].TradeDate])
	if err != nil {
		return nil, fmt.Errorf("benchmark base close: %w", err)
	}
	points := make([]BenchmarkPoint, 0, len(common))
	for _, bar := range common {
		stockClose, err := parsePositiveDecimal(bar.Close)
		if err != nil {
			return nil, fmt.Errorf("stock close at %q: %w", bar.TradeDate, err)
		}
		benchmarkClose, err := parsePositiveDecimal(benchmarkByDate[bar.TradeDate])
		if err != nil {
			return nil, fmt.Errorf("benchmark close at %q: %w", bar.TradeDate, err)
		}
		stockReturn := returnPercentage(stockClose, stockBase)
		benchmarkReturn := returnPercentage(benchmarkClose, benchmarkBase)
		points = append(points, BenchmarkPoint{
			TradeDate:          bar.TradeDate,
			Close:              formatDecimal(benchmarkClose, 2),
			StockReturnPct:     formatDecimal(stockReturn, 2),
			BenchmarkReturnPct: formatDecimal(benchmarkReturn, 2),
			RelativeReturnPct:  formatDecimal(new(big.Rat).Sub(stockReturn, benchmarkReturn), 2),
		})
	}
	return points, nil
}

func averageClose(bars []ChartBar) (string, error) {
	total := new(big.Rat)
	for _, bar := range bars {
		value, err := parseDecimal(bar.Close)
		if err != nil {
			return "", err
		}
		total.Add(total, value)
	}
	total.Quo(total, new(big.Rat).SetInt64(int64(len(bars))))
	return formatDecimal(total, 2), nil
}

func parsePositiveDecimal(value string) (*big.Rat, error) {
	parsed, err := parseDecimal(value)
	if err != nil || parsed.Sign() <= 0 {
		return nil, errors.New("must be a positive decimal")
	}
	return parsed, nil
}

func parseDecimal(value string) (*big.Rat, error) {
	parsed, ok := new(big.Rat).SetString(strings.TrimSpace(value))
	if !ok {
		return nil, fmt.Errorf("invalid decimal %q", value)
	}
	return parsed, nil
}

func adjustMoney(value string, factor *big.Rat) (string, error) {
	parsed, err := parseDecimal(value)
	if err != nil {
		return "", err
	}
	return formatDecimal(new(big.Rat).Mul(parsed, factor), 2), nil
}

func returnPercentage(value, base *big.Rat) *big.Rat {
	percent := new(big.Rat).Sub(new(big.Rat).Quo(value, base), big.NewRat(1, 1))
	return percent.Mul(percent, big.NewRat(100, 1))
}

func formatDecimal(value *big.Rat, precision int) string {
	return value.FloatString(precision)
}
