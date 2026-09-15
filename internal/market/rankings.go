package market

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/disturb-yy/stock-quant/internal/market/domain"
	"github.com/disturb-yy/stock-quant/pkg/api"
)

// RankingRequest 是股票排行查询的应用层请求。
type RankingRequest struct {
	Metric   string
	Page     int
	PageSize int
}

// RankingSnapshot 是排行查询所需的最新交易日快照。
type RankingSnapshot struct {
	SeedVersion  string
	AsOf         string
	Observations []domain.RankingObservation
}

// RankingReader 是股票排行查询所需的最小数据访问边界。
type RankingReader interface {
	ReadRankingSnapshot(context.Context) (RankingSnapshot, error)
}

// RankingValidationError 表示 metric 或应用层分页参数无效。
type RankingValidationError struct {
	Fields map[string]string
}

func (e *RankingValidationError) Error() string {
	return "ranking request parameters are invalid"
}

// Details 返回可直接放入统一错误响应的安全诊断字段。
func (e *RankingValidationError) Details() map[string]any {
	fields := make(map[string]any, len(e.Fields))
	for field, message := range e.Fields {
		fields[field] = message
	}
	return map[string]any{"fields": fields}
}

// RankingHistoryError 表示计算排行所需的前一交易日行情不足。
type RankingHistoryError struct {
	InstrumentCode string
	RequiredBars   int
	AvailableBars  int
}

func (e *RankingHistoryError) Error() string {
	return fmt.Sprintf("instrument %q has insufficient ranking history", e.InstrumentCode)
}

// Details 返回历史数据不足的安全诊断字段。
func (e *RankingHistoryError) Details() map[string]any {
	return map[string]any{
		"instrument_code": e.InstrumentCode,
		"required_bars":   e.RequiredBars,
		"available_bars":  e.AvailableBars,
	}
}

// ErrRankingNoData 表示数据库尚无可供排行的最新行情。
var ErrRankingNoData = errors.New("market ranking data is empty")

// RankingStock 是股票排行条目，包含 MKT-005 可复用的稳定股票身份字段。
type RankingStock struct {
	Rank           int    `json:"rank"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	Value          string `json:"value"`
	Close          string `json:"close"`
	Change         string `json:"change"`
	ChangePercent  string `json:"change_percent"`
	TurnoverAmount string `json:"turnover_amount"`
	TurnoverRate   string `json:"turnover_rate"`
}

// MarketRankings 是 GET /api/v1/markets/rankings 的成功响应。
type MarketRankings struct {
	Metric     domain.RankingMetric `json:"metric"`
	AsOf       string               `json:"as_of"`
	Source     DataSource           `json:"source"`
	Data       []RankingStock       `json:"data"`
	Pagination api.PaginationMeta   `json:"pagination"`
}

// RankingService 编排参数规范化、行情读取、排序和分页。
type RankingService struct {
	reader    RankingReader
	selection ProviderSelection
}

// NewRankingService 创建股票排行查询服务。
func NewRankingService(reader RankingReader, selection ProviderSelection) (*RankingService, error) {
	if reader == nil {
		return nil, errors.New("market ranking reader is required")
	}
	if strings.TrimSpace(selection.Provider) == "" {
		return nil, errors.New("market ranking provider is required")
	}
	return &RankingService{reader: reader, selection: selection}, nil
}

// Rank 按最新交易日执行排行查询。
func (service *RankingService) Rank(ctx context.Context, request RankingRequest) (MarketRankings, error) {
	metric, page, pageSize, err := normalizeRankingRequest(request)
	if err != nil {
		return MarketRankings{}, err
	}
	snapshot, err := service.reader.ReadRankingSnapshot(ctx)
	if err != nil {
		return MarketRankings{}, fmt.Errorf("read market ranking snapshot: %w", err)
	}
	if err := validateRankingSnapshot(snapshot); err != nil {
		return MarketRankings{}, err
	}
	ranked, err := domain.RankObservations(metric, snapshot.Observations)
	if err != nil {
		return MarketRankings{}, fmt.Errorf("rank market observations: %w", err)
	}
	return buildMarketRankings(snapshot, metric, ranked, page, pageSize, service.selection), nil
}

func normalizeRankingRequest(request RankingRequest) (domain.RankingMetric, int, int, error) {
	metric, err := domain.NormalizeRankingMetric(request.Metric)
	if err != nil {
		return "", 0, 0, &RankingValidationError{Fields: map[string]string{"metric": "必须是 gain、loss、turnover_amount 或 turnover_rate"}}
	}
	page := request.Page
	if page == 0 {
		page = api.DefaultPage
	}
	pageSize := request.PageSize
	if pageSize == 0 {
		pageSize = api.DefaultPageSize
	}
	fields := make(map[string]string)
	if page < api.DefaultPage {
		fields["page"] = "必须是大于等于 1 的整数"
	}
	if pageSize < 1 || pageSize > api.MaxPageSize {
		fields["page_size"] = fmt.Sprintf("必须是 1 到 %d 之间的整数", api.MaxPageSize)
	}
	if len(fields) > 0 {
		return "", 0, 0, &RankingValidationError{Fields: fields}
	}
	return metric, page, pageSize, nil
}

func validateRankingSnapshot(snapshot RankingSnapshot) error {
	if strings.TrimSpace(snapshot.AsOf) == "" || strings.TrimSpace(snapshot.SeedVersion) == "" {
		return errors.New("market ranking as-of and seed version are required")
	}
	for _, observation := range snapshot.Observations {
		if strings.TrimSpace(observation.InstrumentCode) == "" || strings.TrimSpace(observation.InstrumentName) == "" {
			return errors.New("market ranking instrument identity is incomplete")
		}
		if strings.TrimSpace(observation.CurrentClose) == "" || strings.TrimSpace(observation.PreviousClose) == "" {
			return &RankingHistoryError{InstrumentCode: observation.InstrumentCode, RequiredBars: 2, AvailableBars: 1}
		}
	}
	return nil
}

func buildMarketRankings(snapshot RankingSnapshot, metric domain.RankingMetric, ranked []domain.RankingResult, page, pageSize int, selection ProviderSelection) MarketRankings {
	total := int64(len(ranked))
	start := paginationStart(len(ranked), page, pageSize)
	end := start + pageSize
	if end > len(ranked) {
		end = len(ranked)
	}
	data := make([]RankingStock, 0, end-start)
	for _, result := range ranked[start:end] {
		data = append(data, RankingStock{
			Rank: result.Rank, Code: result.InstrumentCode, Name: result.InstrumentName,
			Value: result.Value, Close: result.Close, Change: result.Change,
			ChangePercent: result.ChangePercent, TurnoverAmount: result.TurnoverAmount,
			TurnoverRate: result.TurnoverRate,
		})
	}
	return MarketRankings{
		Metric: metric,
		AsOf:   snapshot.AsOf,
		Source: DataSource{Mode: selection.Mode, Provider: selection.Provider, SeedVersion: snapshot.SeedVersion},
		Data:   data,
		Pagination: api.PaginationMeta{
			Page: page, PageSize: pageSize, Total: total,
			TotalPages: totalPages(total, int64(pageSize)),
		},
	}
}

func paginationStart(total, page, pageSize int) int {
	if total == 0 || page-1 > total/pageSize {
		return total
	}
	start := (page - 1) * pageSize
	if start > total {
		return total
	}
	return start
}

func totalPages(total, pageSize int64) int64 {
	if total == 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}
