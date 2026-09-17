package stock

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	stockdomain "github.com/disturb-yy/stock-quant/internal/stock/domain"
)

const defaultValuationRange = stockdomain.RangeFiveYears

const (
	ValuationRangeThreeYears = stockdomain.RangeThreeYears
	ValuationRangeFiveYears  = stockdomain.RangeFiveYears
)

var (
	// ErrValuationInstrumentNotFound 表示估值查询的股票身份不存在。
	ErrValuationInstrumentNotFound = errors.New("valuation instrument not found")
)

// ValuationRequest 是股票估值查询的应用层请求。
type ValuationRequest struct {
	Symbol string
	Range  stockdomain.ValuationRange
}

// ValuationSnapshot 是基础设施返回的估值原始聚合结果。
type ValuationSnapshot struct {
	Symbol       string
	Name         string
	SeedVersion  string
	SourceAsOf   string
	Observations []stockdomain.ValuationObservation
	Industries   []ValuationIndustrySnapshot
}

// ValuationIndustrySnapshot 是目标股票的一个既有行业映射及其成员估值。
type ValuationIndustrySnapshot struct {
	Code    string
	Name    string
	Members []ValuationMemberSnapshot
}

// ValuationMemberSnapshot 是行业成员身份及其估值历史。
type ValuationMemberSnapshot struct {
	InstrumentCode string
	Name           string
	Observations   []stockdomain.ValuationObservation
}

// ValuationReader 是股票估值查询所需的最小数据访问边界。
type ValuationReader interface {
	ReadStockValuation(context.Context, ValuationRequest) (ValuationSnapshot, error)
}

// ValuationSource 是估值数据的模式和 Provider 标识。
type ValuationSource struct {
	Mode     string
	Provider string
}

// ValuationValidationError 表示估值查询参数不符合契约。
type ValuationValidationError struct {
	Fields map[string]string
}

func (err *ValuationValidationError) Error() string {
	return "stock valuation request parameters are invalid"
}

// Details 返回可安全暴露的参数诊断字段。
func (err *ValuationValidationError) Details() map[string]any {
	fields := make(map[string]any, len(err.Fields))
	for field, message := range err.Fields {
		fields[field] = message
	}
	return map[string]any{"fields": fields}
}

// ValuationService 编排股票估值、历史分位和行业比较查询。
type ValuationService struct {
	reader ValuationReader
	source ValuationSource
}

// NewValuationService 创建股票估值查询服务。
func NewValuationService(reader ValuationReader, source ValuationSource) (*ValuationService, error) {
	if reader == nil {
		return nil, errors.New("stock valuation reader is required")
	}
	if strings.TrimSpace(source.Mode) == "" || strings.TrimSpace(source.Provider) == "" {
		return nil, errors.New("stock valuation source is required")
	}
	return &ValuationService{reader: reader, source: source}, nil
}

// Valuation 读取目标股票估值，并在服务端完成历史和同业计算。
func (service *ValuationService) Valuation(ctx context.Context, request ValuationRequest) (StockValuation, error) {
	normalized, err := normalizeValuationRequest(request)
	if err != nil {
		return StockValuation{}, err
	}
	snapshot, err := service.reader.ReadStockValuation(ctx, normalized)
	if err != nil {
		return StockValuation{}, fmt.Errorf("read stock valuation %q: %w", normalized.Symbol, err)
	}
	if err := validateValuationSnapshot(snapshot, normalized.Symbol); err != nil {
		return StockValuation{}, err
	}
	from, to, asOf, err := valuationBounds(snapshot.Observations, normalized.Range)
	if err != nil {
		return StockValuation{}, err
	}
	metrics, err := buildValuationMetrics(snapshot.Observations, from, to)
	if err != nil {
		return StockValuation{}, fmt.Errorf("calculate stock valuation %q: %w", normalized.Symbol, err)
	}
	comparisons, err := buildIndustryComparisons(snapshot.Industries, metrics, normalized.Symbol, asOf)
	if err != nil {
		return StockValuation{}, fmt.Errorf("calculate stock valuation industries %q: %w", normalized.Symbol, err)
	}
	return StockValuation{
		Symbol: snapshot.Symbol, Name: snapshot.Name, RequestedRange: normalized.Range,
		EffectiveRange: effectiveValuationRange(snapshot.Observations), AsOf: optionalString(asOf),
		Metrics: metrics, IndustryComparisons: comparisons,
		Source: ValuationDataSource{Mode: service.source.Mode, Provider: service.source.Provider, SeedVersion: snapshot.SeedVersion, AsOf: snapshot.SourceAsOf},
	}, nil
}

func normalizeValuationRequest(request ValuationRequest) (ValuationRequest, error) {
	fields := make(map[string]string)
	if strings.TrimSpace(request.Symbol) != request.Symbol || !symbolPattern.MatchString(request.Symbol) {
		fields["symbol"] = "必须是 Markets 返回的股票代码"
	}
	requestedRange := request.Range
	if requestedRange == "" {
		requestedRange = defaultValuationRange
	} else if requestedRange != stockdomain.RangeThreeYears && requestedRange != stockdomain.RangeFiveYears {
		fields["range"] = "必须是 3y 或 5y"
	}
	if len(fields) > 0 {
		return ValuationRequest{}, &ValuationValidationError{Fields: fields}
	}
	return ValuationRequest{Symbol: request.Symbol, Range: requestedRange}, nil
}

func validateValuationSnapshot(snapshot ValuationSnapshot, symbol string) error {
	if snapshot.Symbol != symbol {
		return fmt.Errorf("stock valuation identity mismatch: requested %q, got %q", symbol, snapshot.Symbol)
	}
	if strings.TrimSpace(snapshot.Name) == "" || strings.TrimSpace(snapshot.SeedVersion) == "" || strings.TrimSpace(snapshot.SourceAsOf) == "" {
		return errors.New("stock valuation name, seed version and source as-of are required")
	}
	for _, observation := range snapshot.Observations {
		if observation.InstrumentCode != symbol {
			return fmt.Errorf("stock valuation observation identity mismatch: requested %q, got %q", symbol, observation.InstrumentCode)
		}
		if err := observation.Validate(); err != nil {
			return fmt.Errorf("validate stock valuation observation: %w", err)
		}
	}
	for _, industry := range snapshot.Industries {
		if strings.TrimSpace(industry.Code) == "" || strings.TrimSpace(industry.Name) == "" {
			return errors.New("stock valuation industry code and name are required")
		}
		for _, member := range industry.Members {
			if strings.TrimSpace(member.InstrumentCode) == "" || strings.TrimSpace(member.Name) == "" {
				return errors.New("stock valuation member identity is required")
			}
			for _, observation := range member.Observations {
				if observation.InstrumentCode != member.InstrumentCode {
					return fmt.Errorf("stock valuation member identity mismatch: member %q, observation %q", member.InstrumentCode, observation.InstrumentCode)
				}
				if err := observation.Validate(); err != nil {
					return fmt.Errorf("validate stock valuation member observation: %w", err)
				}
			}
		}
	}
	return nil
}

func valuationBounds(observations []stockdomain.ValuationObservation, requestedRange stockdomain.ValuationRange) (string, string, string, error) {
	if len(observations) == 0 {
		return "", "", "", nil
	}
	latest := observations[0].AsOf
	for _, observation := range observations[1:] {
		if observation.AsOf > latest {
			latest = observation.AsOf
		}
	}
	latestDate, err := time.Parse("2006-01-02", latest)
	if err != nil {
		return "", "", "", fmt.Errorf("parse latest valuation date: %w", err)
	}
	years := 5
	if requestedRange == stockdomain.RangeThreeYears {
		years = 3
	}
	return latestDate.AddDate(-years, 0, 0).Format("2006-01-02"), latest, latest, nil
}

func effectiveValuationRange(observations []stockdomain.ValuationObservation) FinancialEffectiveRange {
	if len(observations) == 0 {
		return FinancialEffectiveRange{}
	}
	from, to := observations[0].AsOf, observations[0].AsOf
	for _, observation := range observations[1:] {
		if observation.AsOf < from {
			from = observation.AsOf
		}
		if observation.AsOf > to {
			to = observation.AsOf
		}
	}
	return FinancialEffectiveRange{From: &from, To: &to}
}

type valuationMetricKey string

const (
	valuationMetricPE valuationMetricKey = "pe_ttm"
	valuationMetricPB valuationMetricKey = "pb"
	valuationMetricPS valuationMetricKey = "ps_ttm"
)

func buildValuationMetrics(observations []stockdomain.ValuationObservation, from, to string) (ValuationMetrics, error) {
	pe, err := analyzeMetric(observations, from, to, valuationMetricPE)
	if err != nil {
		return ValuationMetrics{}, err
	}
	pb, err := analyzeMetric(observations, from, to, valuationMetricPB)
	if err != nil {
		return ValuationMetrics{}, err
	}
	ps, err := analyzeMetric(observations, from, to, valuationMetricPS)
	if err != nil {
		return ValuationMetrics{}, err
	}
	return ValuationMetrics{PETTM: valuationMetricResponse(pe), PB: valuationMetricResponse(pb), PSTTM: valuationMetricResponse(ps)}, nil
}

func analyzeMetric(observations []stockdomain.ValuationObservation, from, to string, key valuationMetricKey) (stockdomain.ValuationMetricResult, error) {
	values := make([]stockdomain.ValuationMetricObservation, 0, len(observations))
	for _, observation := range observations {
		values = append(values, metricObservation(observation, key))
	}
	return stockdomain.AnalyzeValuationMetric(values, from, to)
}

func metricObservation(observation stockdomain.ValuationObservation, key valuationMetricKey) stockdomain.ValuationMetricObservation {
	switch key {
	case valuationMetricPB:
		value := observation.PB
		value.AsOf = observation.AsOf
		return value
	case valuationMetricPS:
		value := observation.PSTTM
		value.AsOf = observation.AsOf
		return value
	default:
		value := observation.PETTM
		value.AsOf = observation.AsOf
		return value
	}
}

func valuationMetricResponse(result stockdomain.ValuationMetricResult) ValuationMetric {
	history := make([]ValuationHistoryPoint, 0, len(result.History))
	for _, point := range result.History {
		history = append(history, ValuationHistoryPoint{AsOf: point.AsOf, Value: point.Value})
	}
	return ValuationMetric{
		Current: ValuationCurrent{Value: result.Current.Value, AsOf: result.Current.AsOf, Basis: result.Current.Basis},
		History: history,
		Percentile: ValuationPercentile{
			Value: result.Percentile.Value, SampleSize: result.Percentile.SampleSize,
			RangeFrom: result.Percentile.RangeFrom, RangeTo: result.Percentile.RangeTo, Method: result.Percentile.Method,
		},
		Position: result.Position,
	}
}

func buildIndustryComparisons(industries []ValuationIndustrySnapshot, metrics ValuationMetrics, symbol, asOf string) ([]IndustryComparison, error) {
	ordered := append([]ValuationIndustrySnapshot(nil), industries...)
	sort.SliceStable(ordered, func(left, right int) bool { return ordered[left].Code < ordered[right].Code })
	comparisons := make([]IndustryComparison, 0, len(ordered))
	for _, industry := range ordered {
		pe, err := industryMetricMedian(industry, valuationMetricPE, metrics.PETTM, symbol, asOf)
		if err != nil {
			return nil, err
		}
		pb, err := industryMetricMedian(industry, valuationMetricPB, metrics.PB, symbol, asOf)
		if err != nil {
			return nil, err
		}
		ps, err := industryMetricMedian(industry, valuationMetricPS, metrics.PSTTM, symbol, asOf)
		if err != nil {
			return nil, err
		}
		comparison := IndustryComparison{
			Industry: Industry{Code: industry.Code, Name: industry.Name},
			Metrics:  IndustryComparisonMetrics{PETTM: pe, PB: pb, PSTTM: ps},
		}
		if asOf != "" {
			comparison.AsOf = &asOf
		}
		comparisons = append(comparisons, comparison)
	}
	return comparisons, nil
}

func industryMetricMedian(industry ValuationIndustrySnapshot, key valuationMetricKey, target ValuationMetric, symbol, asOf string) (IndustryMetric, error) {
	values := make([]stockdomain.ValuationPeerObservation, 0)
	for _, member := range industry.Members {
		for _, observation := range member.Observations {
			metric := metricObservation(observation, key)
			values = append(values, stockdomain.ValuationPeerObservation{InstrumentCode: member.InstrumentCode, AsOf: observation.AsOf, Value: metric.Value, Basis: metric.Basis})
		}
	}
	median, err := stockdomain.CalculateIndustryMedian(values, symbol, asOf, stringPointerValue(target.Current.Basis), target.Current.Value)
	if err != nil {
		return IndustryMetric{}, err
	}
	return IndustryMetric{Value: median.Value, SampleSize: median.SampleSize}, nil
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// StockValuation 是股票估值研究 API 响应。
type StockValuation struct {
	Symbol              string                     `json:"symbol"`
	Name                string                     `json:"name"`
	RequestedRange      stockdomain.ValuationRange `json:"requested_range"`
	EffectiveRange      FinancialEffectiveRange    `json:"effective_range"`
	AsOf                *string                    `json:"as_of"`
	Metrics             ValuationMetrics           `json:"metrics"`
	IndustryComparisons []IndustryComparison       `json:"industry_comparisons"`
	Source              ValuationDataSource        `json:"source"`
}

// ValuationMetrics 是三项估值指标，字段始终存在。
type ValuationMetrics struct {
	PETTM ValuationMetric `json:"pe_ttm"`
	PB    ValuationMetric `json:"pb"`
	PSTTM ValuationMetric `json:"ps_ttm"`
}

// ValuationMetric 是当前值、历史序列、分位和位置。
type ValuationMetric struct {
	Current    ValuationCurrent               `json:"current"`
	History    []ValuationHistoryPoint        `json:"history"`
	Percentile ValuationPercentile            `json:"percentile"`
	Position   *stockdomain.ValuationPosition `json:"position"`
}

// ValuationCurrent 是指标当前值及其来源口径。
type ValuationCurrent struct {
	Value *string `json:"value"`
	AsOf  *string `json:"as_of"`
	Basis *string `json:"basis"`
}

// ValuationHistoryPoint 是一条按日期升序的历史正数观测。
type ValuationHistoryPoint struct {
	AsOf  string `json:"as_of"`
	Value string `json:"value"`
}

// ValuationPercentile 是服务端固定规则计算的历史分位。
type ValuationPercentile struct {
	Value      *string `json:"value"`
	SampleSize int     `json:"sample_size"`
	RangeFrom  *string `json:"range_from"`
	RangeTo    *string `json:"range_to"`
	Method     string  `json:"method"`
}

// IndustryComparison 是一个既有行业映射的同日同行比较。
type IndustryComparison struct {
	Industry Industry                  `json:"industry"`
	AsOf     *string                   `json:"as_of"`
	Metrics  IndustryComparisonMetrics `json:"metrics"`
}

// Industry 是行业身份。
type Industry struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// IndustryComparisonMetrics 是按指标分别计算的同行中位数。
type IndustryComparisonMetrics struct {
	PETTM IndustryMetric `json:"pe_ttm"`
	PB    IndustryMetric `json:"pb"`
	PSTTM IndustryMetric `json:"ps_ttm"`
}

// IndustryMetric 是同行中位数及有效样本数。
type IndustryMetric struct {
	Value      *string `json:"value"`
	SampleSize int     `json:"sample_size"`
}

// ValuationDataSource 描述估值来源模式、Provider、Seed 和截至日期。
type ValuationDataSource struct {
	Mode        string `json:"mode"`
	Provider    string `json:"provider"`
	SeedVersion string `json:"seed_version"`
	AsOf        string `json:"as_of"`
}
