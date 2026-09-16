package stock

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrInstrumentNotFound 表示详情查询的股票身份不存在。
	ErrInstrumentNotFound = errors.New("instrument not found")
	// ErrQuoteUnavailable 表示详情所需的同日最新行情不可用。
	ErrQuoteUnavailable = errors.New("stock quote unavailable")
	symbolPattern       = regexp.MustCompile(`^[A-Za-z0-9]{1,16}\.[A-Za-z]{2,8}$`)
)

// MetricSnapshot 是一个指标及其口径、日期的原始快照。
type MetricSnapshot struct {
	Value *string
	AsOf  string
	Basis string
}

// QuoteSnapshot 是同一交易日的价格快照。
type QuoteSnapshot struct {
	Last      string
	Change    string
	ChangePct string
	AsOf      string
}

// SparklinePoint 是走势图的一个交易日 OHLC 点。
type SparklinePoint struct {
	TradeDate string `json:"trade_date"`
	Open      string `json:"open"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Close     string `json:"close"`
}

// OverviewSnapshot 是基础设施返回的股票详情聚合结果。
type OverviewSnapshot struct {
	Symbol    string
	Name      string
	Industry  string
	Quote     QuoteSnapshot
	MarketCap MetricSnapshot
	PETTM     MetricSnapshot
	PB        MetricSnapshot
	ROE       MetricSnapshot
	Sparkline []SparklinePoint
}

// StockOverview 是股票详情概览 API 响应。
type StockOverview struct {
	Symbol    string            `json:"symbol"`
	Name      string            `json:"name"`
	Industry  string            `json:"industry"`
	Quote     QuoteOverview     `json:"quote"`
	Metrics   MetricsOverview   `json:"metrics"`
	Sparkline SparklineOverview `json:"sparkline"`
}

// QuoteOverview 是股票详情的行情响应。
type QuoteOverview struct {
	Last      string `json:"last"`
	Change    string `json:"change"`
	ChangePct string `json:"change_pct"`
	AsOf      string `json:"as_of"`
}

// MetricOverview 是带日期和口径的指标响应。
type MetricOverview struct {
	Value *string `json:"value"`
	AsOf  *string `json:"as_of"`
	Basis *string `json:"basis"`
}

// MetricsOverview 是股票详情的关键指标响应。
type MetricsOverview struct {
	MarketCap MetricOverview `json:"market_cap"`
	PETTM     MetricOverview `json:"pe_ttm"`
	PB        MetricOverview `json:"pb"`
	ROE       MetricOverview `json:"roe"`
}

// SparklineOverview 是股票详情的 20 日走势图响应。
type SparklineOverview struct {
	Period string           `json:"period"`
	Points []SparklinePoint `json:"points"`
}

// OverviewReader 是股票详情聚合所需的最小数据访问边界。
type OverviewReader interface {
	ReadStockOverview(context.Context, string) (OverviewSnapshot, error)
}

// OverviewService 编排股票详情概览用例。
type OverviewService struct {
	reader OverviewReader
}

// NewOverviewService 创建股票详情概览服务。
func NewOverviewService(reader OverviewReader) (*OverviewService, error) {
	if reader == nil {
		return nil, errors.New("stock overview reader is required")
	}
	return &OverviewService{reader: reader}, nil
}

// Overview 按 Markets 返回的原始 code 查询股票详情。
func (service *OverviewService) Overview(ctx context.Context, symbol string) (StockOverview, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return StockOverview{}, err
	}
	snapshot, err := service.reader.ReadStockOverview(ctx, normalized)
	if err != nil {
		return StockOverview{}, fmt.Errorf("read stock overview %q: %w", normalized, err)
	}
	if snapshot.Symbol != normalized {
		return StockOverview{}, fmt.Errorf("stock overview identity mismatch: requested %q, got %q", normalized, snapshot.Symbol)
	}
	return overviewFromSnapshot(snapshot), nil
}

func normalizeSymbol(symbol string) (string, error) {
	if strings.TrimSpace(symbol) != symbol || !symbolPattern.MatchString(symbol) {
		return "", &SymbolValidationError{Message: "symbol 必须是 Markets 返回的股票代码"}
	}
	return symbol, nil
}

// SymbolValidationError 表示详情 symbol 参数不合法。
type SymbolValidationError struct {
	Message string
}

func (err *SymbolValidationError) Error() string {
	return "stock symbol is invalid"
}

// Details 返回统一错误模型所需的安全字段。
func (err *SymbolValidationError) Details() map[string]any {
	return map[string]any{"fields": map[string]string{"symbol": err.Message}}
}

func overviewFromSnapshot(snapshot OverviewSnapshot) StockOverview {
	return StockOverview{
		Symbol: snapshot.Symbol, Name: snapshot.Name, Industry: snapshot.Industry,
		Quote: QuoteOverview{Last: snapshot.Quote.Last, Change: snapshot.Quote.Change, ChangePct: snapshot.Quote.ChangePct, AsOf: snapshot.Quote.AsOf},
		Metrics: MetricsOverview{
			MarketCap: metricOverview(snapshot.MarketCap), PETTM: metricOverview(snapshot.PETTM),
			PB: metricOverview(snapshot.PB), ROE: metricOverview(snapshot.ROE),
		},
		Sparkline: SparklineOverview{Period: "20d", Points: snapshot.Sparkline},
	}
}

func metricOverview(snapshot MetricSnapshot) MetricOverview {
	return MetricOverview{Value: snapshot.Value, AsOf: optionalString(snapshot.AsOf), Basis: optionalString(snapshot.Basis)}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
