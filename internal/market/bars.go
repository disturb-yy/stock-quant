package market

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/disturb-yy/stock-quant/internal/market/domain"
)

const (
	defaultBarsTimeframe = "1d"
	defaultBarsAdjust    = string(domain.AdjustmentNone)
	defaultBarsRange     = "120d"
	benchmarkCSI300      = "000300.SH"
)

var (
	// ErrBarsInstrumentNotFound 表示股票身份不存在。
	ErrBarsInstrumentNotFound = errors.New("stock bars instrument not found")
	// ErrBarsBenchmarkUnavailable 表示请求的基准数据依赖不可用。
	ErrBarsBenchmarkUnavailable = errors.New("stock bars benchmark unavailable")
	barsSymbolPattern           = regexp.MustCompile(`^[A-Za-z0-9]{1,16}\.[A-Za-z]{2,8}$`)
)

// BarsRequest 是股票日线查询的应用层请求。
type BarsRequest struct {
	Symbol    string
	Timeframe string
	Adjust    string
	Range     string
	From      string
	To        string
	Benchmark string
}

// BarsSnapshot 是基础设施返回的原始日线、复权因子与基准数据。
type BarsSnapshot struct {
	Symbol        string
	Name          string
	SeedVersion   string
	Bars          []domain.BarWithAdjustment
	BenchmarkCode string
	BenchmarkName string
	Benchmark     []domain.BenchmarkBar
}

// BarsReader 是股票日线查询所需的最小数据访问边界。
type BarsReader interface {
	ReadStockBars(context.Context, BarsRequest) (BarsSnapshot, error)
}

// StockBar 是股票行情响应中的一条日线。
type StockBar struct {
	TradeDate string  `json:"trade_date"`
	Open      string  `json:"open"`
	High      string  `json:"high"`
	Low       string  `json:"low"`
	Close     string  `json:"close"`
	Volume    int64   `json:"volume"`
	MA5       *string `json:"ma5"`
	MA20      *string `json:"ma20"`
}

// EffectiveRange 表示实际返回数据的首尾交易日。
type EffectiveRange struct {
	From *string `json:"from"`
	To   *string `json:"to"`
}

// BenchmarkOverview 是股票行情响应中的基准序列。
type BenchmarkOverview struct {
	Code   string           `json:"code"`
	Name   string           `json:"name"`
	Points []BenchmarkPoint `json:"points"`
}

// BenchmarkPoint 是基准收盘与共同日期收益率。
type BenchmarkPoint struct {
	TradeDate          string `json:"trade_date"`
	Close              string `json:"close"`
	StockReturnPct     string `json:"stock_return_pct"`
	BenchmarkReturnPct string `json:"benchmark_return_pct"`
	RelativeReturnPct  string `json:"relative_return_pct"`
}

// StockBars 是 GET /api/v1/stocks/{symbol}/bars 的成功响应。
type StockBars struct {
	Symbol         string                 `json:"symbol"`
	Name           string                 `json:"name"`
	Timeframe      string                 `json:"timeframe"`
	Adjust         domain.ChartAdjustment `json:"adjust"`
	EffectiveRange EffectiveRange         `json:"effective_range"`
	Bars           []StockBar             `json:"bars"`
	Benchmark      *BenchmarkOverview     `json:"benchmark"`
	Source         DataSource             `json:"source"`
}

// BarsValidationError 表示股票行情请求参数不符合稳定契约。
type BarsValidationError struct {
	Fields map[string]string
}

func (err *BarsValidationError) Error() string {
	return "stock bars request parameters are invalid"
}

// Details 返回可直接放入统一错误响应的安全诊断字段。
func (err *BarsValidationError) Details() map[string]any {
	fields := make(map[string]any, len(err.Fields))
	for field, message := range err.Fields {
		fields[field] = message
	}
	return map[string]any{"fields": fields}
}

// BarsService 编排日线读取、复权、均线和基准表现计算。
type BarsService struct {
	reader    BarsReader
	selection ProviderSelection
}

// NewBarsService 创建股票日线查询服务。
func NewBarsService(reader BarsReader, selection ProviderSelection) (*BarsService, error) {
	if reader == nil {
		return nil, errors.New("stock bars reader is required")
	}
	if strings.TrimSpace(selection.Provider) == "" {
		return nil, errors.New("stock bars provider is required")
	}
	return &BarsService{reader: reader, selection: selection}, nil
}

// Bars 读取并计算一只股票的研究型日线。
func (service *BarsService) Bars(ctx context.Context, request BarsRequest) (StockBars, error) {
	normalized, err := normalizeBarsRequest(request)
	if err != nil {
		return StockBars{}, err
	}
	snapshot, err := service.reader.ReadStockBars(ctx, normalized)
	if err != nil {
		return StockBars{}, fmt.Errorf("read stock bars %q: %w", normalized.Symbol, err)
	}
	if err := validateBarsSnapshot(snapshot, normalized.Symbol); err != nil {
		return StockBars{}, err
	}
	resultBars, err := buildStockBars(snapshot.Bars, normalized.Adjust)
	if err != nil {
		return StockBars{}, fmt.Errorf("build stock bars %q: %w", normalized.Symbol, err)
	}
	result := StockBars{
		Symbol:         snapshot.Symbol,
		Name:           snapshot.Name,
		Timeframe:      normalized.Timeframe,
		Adjust:         domain.ChartAdjustment(normalized.Adjust),
		EffectiveRange: effectiveBarsRange(resultBars),
		Bars:           resultBars,
		Source:         DataSource{Mode: service.selection.Mode, Provider: service.selection.Provider, SeedVersion: snapshot.SeedVersion},
	}
	if normalized.Benchmark != "" {
		result.Benchmark, err = buildBenchmarkOverview(snapshot, resultBars, normalized.Benchmark)
		if err != nil {
			return StockBars{}, err
		}
	}
	return result, nil
}

func normalizeBarsRequest(request BarsRequest) (BarsRequest, error) {
	fields := make(map[string]string)
	if strings.TrimSpace(request.Symbol) != request.Symbol || !barsSymbolPattern.MatchString(request.Symbol) {
		fields["symbol"] = "必须是 Markets 返回的股票代码"
	}
	timeframe := request.Timeframe
	if timeframe == "" {
		timeframe = defaultBarsTimeframe
	} else if timeframe != defaultBarsTimeframe {
		fields["timeframe"] = "必须是 1d"
	}
	adjust := request.Adjust
	if adjust == "" {
		adjust = defaultBarsAdjust
	} else if adjust != string(domain.AdjustmentNone) && adjust != string(domain.AdjustmentQFQ) && adjust != string(domain.AdjustmentHFQ) {
		fields["adjust"] = "必须是 none、qfq 或 hfq"
	}
	chartRange := request.Range
	if chartRange == "" && request.From == "" && request.To == "" {
		chartRange = defaultBarsRange
	}
	if chartRange != "" && request.From != "" || chartRange != "" && request.To != "" {
		fields["range"] = "不能与 from/to 同时提供"
	}
	if request.From == "" && request.To != "" {
		fields["from"] = "必须与 to 同时提供"
	}
	if request.From != "" && request.To == "" {
		fields["to"] = "必须与 from 同时提供"
	}
	if request.From != "" && request.To != "" {
		if !isDate(request.From) {
			fields["from"] = "必须是 YYYY-MM-DD"
		}
		if !isDate(request.To) {
			fields["to"] = "必须是 YYYY-MM-DD"
		}
		if isDate(request.From) && isDate(request.To) && request.From > request.To {
			fields["from"] = "不能晚于 to"
		}
	}
	if chartRange != "" && chartRange != "20d" && chartRange != "60d" && chartRange != "120d" && chartRange != "all" {
		fields["range"] = "必须是 20d、60d、120d 或 all"
	}
	if request.Benchmark != "" && request.Benchmark != benchmarkCSI300 {
		fields["benchmark"] = "当前只支持 000300.SH"
	}
	if len(fields) > 0 {
		return BarsRequest{}, &BarsValidationError{Fields: fields}
	}
	return BarsRequest{Symbol: request.Symbol, Timeframe: timeframe, Adjust: adjust, Range: chartRange, From: request.From, To: request.To, Benchmark: request.Benchmark}, nil
}

func isDate(value string) bool {
	parsed, err := time.Parse("2006-01-02", value)
	return err == nil && parsed.Format("2006-01-02") == value
}

func validateBarsSnapshot(snapshot BarsSnapshot, symbol string) error {
	if snapshot.Symbol != symbol {
		return fmt.Errorf("stock bars identity mismatch: requested %q, got %q", symbol, snapshot.Symbol)
	}
	if strings.TrimSpace(snapshot.Name) == "" || strings.TrimSpace(snapshot.SeedVersion) == "" {
		return errors.New("stock bars name and seed version are required")
	}
	previousDate := ""
	for _, bar := range snapshot.Bars {
		if err := bar.Bar.Validate(); err != nil {
			return fmt.Errorf("validate stock bar %q: %w", bar.Bar.TradeDate, err)
		}
		if previousDate != "" && bar.Bar.TradeDate <= previousDate {
			return fmt.Errorf("stock bars must be unique and ascending by trade date")
		}
		previousDate = bar.Bar.TradeDate
	}
	return nil
}

func buildStockBars(source []domain.BarWithAdjustment, adjustment string) ([]StockBar, error) {
	result := make([]domain.ChartBar, 0, len(source))
	for _, bar := range source {
		adjusted, err := domain.ApplyAdjustment(bar, domain.ChartAdjustment(adjustment))
		if err != nil {
			return nil, fmt.Errorf("adjust %q: %w", bar.Bar.TradeDate, err)
		}
		result = append(result, adjusted)
	}
	if err := domain.CalculateMovingAverages(result); err != nil {
		return nil, err
	}
	bars := make([]StockBar, 0, len(result))
	for _, bar := range result {
		bars = append(bars, StockBar{TradeDate: bar.TradeDate, Open: bar.Open, High: bar.High, Low: bar.Low, Close: bar.Close, Volume: bar.Volume, MA5: bar.MA5, MA20: bar.MA20})
	}
	return bars, nil
}

func effectiveBarsRange(bars []StockBar) EffectiveRange {
	if len(bars) == 0 {
		return EffectiveRange{}
	}
	from, to := bars[0].TradeDate, bars[len(bars)-1].TradeDate
	return EffectiveRange{From: &from, To: &to}
}

func buildBenchmarkOverview(snapshot BarsSnapshot, stock []StockBar, code string) (*BenchmarkOverview, error) {
	if snapshot.BenchmarkCode != code || strings.TrimSpace(snapshot.BenchmarkName) == "" {
		return nil, fmt.Errorf("%w: %s", ErrBarsBenchmarkUnavailable, code)
	}
	chartBars := make([]domain.ChartBar, 0, len(stock))
	for _, bar := range stock {
		chartBars = append(chartBars, domain.ChartBar{TradeDate: bar.TradeDate, Close: bar.Close})
	}
	points, err := domain.BuildBenchmarkPoints(chartBars, snapshot.Benchmark)
	if err != nil {
		return nil, fmt.Errorf("build benchmark points: %w", err)
	}
	result := &BenchmarkOverview{Code: code, Name: snapshot.BenchmarkName, Points: make([]BenchmarkPoint, 0, len(points))}
	for _, point := range points {
		result.Points = append(result.Points, BenchmarkPoint{TradeDate: point.TradeDate, Close: point.Close, StockReturnPct: point.StockReturnPct, BenchmarkReturnPct: point.BenchmarkReturnPct, RelativeReturnPct: point.RelativeReturnPct})
	}
	return result, nil
}
