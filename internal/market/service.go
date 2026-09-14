package market

import (
	"context"
	"fmt"
	"strings"

	"github.com/disturb-yy/stock-quant/internal/market/domain"
)

const (
	shanghaiIndexCode = "000001.SH"
	shenzhenIndexCode = "399001.SZ"
	chiNextIndexCode  = "399006.SZ"
	csi300IndexCode   = "000300.SH"
)

var overviewIndexOrder = []string{
	shanghaiIndexCode,
	shenzhenIndexCode,
	chiNextIndexCode,
	csi300IndexCode,
}

// OverviewSnapshot 是 Provider 返回的市场概览原始快照。
type OverviewSnapshot struct {
	SeedVersion string
	AsOf        string
	ObservedAt  string
	Indices     []domain.IndexSnapshot
	Breadth     domain.MarketBreadth
}

// OverviewReader 是市场概览查询所需的最小数据访问边界。
type OverviewReader interface {
	ReadOverviewSnapshot(context.Context) (OverviewSnapshot, error)
}

// DataSource 描述市场数据的来源，避免 Seed 数据被误认为实时数据。
type DataSource struct {
	Mode        ProviderMode `json:"mode"`
	Provider    string       `json:"provider"`
	SeedVersion string       `json:"seed_version"`
}

// IndexOverview 是前端展示用的指数快照。
type IndexOverview struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Close         string `json:"close"`
	Change        string `json:"change"`
	ChangePercent string `json:"change_percent"`
}

// BreadthOverview 是前端展示用的市场宽度统计。
type BreadthOverview struct {
	Advancing int64 `json:"advancing"`
	Declining int64 `json:"declining"`
	Unchanged int64 `json:"unchanged"`
}

// TurnoverOverview 是前端展示用的成交额统计。
type TurnoverOverview struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

// MarketOverview 是 GET /api/v1/markets/overview 的成功响应。
type MarketOverview struct {
	AsOf       string           `json:"as_of"`
	ObservedAt string           `json:"observed_at"`
	Source     DataSource       `json:"source"`
	Indices    []IndexOverview  `json:"indices"`
	Breadth    BreadthOverview  `json:"breadth"`
	Turnover   TurnoverOverview `json:"turnover"`
}

// OverviewService 编排市场概览查询和结果完整性校验。
type OverviewService struct {
	reader    OverviewReader
	selection ProviderSelection
}

// NewOverviewService 创建市场概览查询服务。
func NewOverviewService(reader OverviewReader, selection ProviderSelection) (*OverviewService, error) {
	if reader == nil {
		return nil, fmt.Errorf("market overview reader is required")
	}
	if strings.TrimSpace(selection.Provider) == "" {
		return nil, fmt.Errorf("market overview provider is required")
	}
	return &OverviewService{reader: reader, selection: selection}, nil
}

// Overview 读取并整理一个完整交易日的市场概览。
func (service *OverviewService) Overview(ctx context.Context) (MarketOverview, error) {
	snapshot, err := service.reader.ReadOverviewSnapshot(ctx)
	if err != nil {
		return MarketOverview{}, fmt.Errorf("read market overview snapshot: %w", err)
	}
	orderedIndices, err := validateOverviewSnapshot(snapshot)
	if err != nil {
		return MarketOverview{}, fmt.Errorf("validate market overview snapshot: %w", err)
	}
	return buildMarketOverview(snapshot, orderedIndices, service.selection), nil
}

func validateOverviewSnapshot(snapshot OverviewSnapshot) ([]domain.IndexSnapshot, error) {
	if strings.TrimSpace(snapshot.AsOf) == "" || strings.TrimSpace(snapshot.ObservedAt) == "" {
		return nil, fmt.Errorf("overview as-of and observed-at are required")
	}
	if strings.TrimSpace(snapshot.SeedVersion) == "" {
		return nil, fmt.Errorf("overview seed version is required")
	}
	if err := snapshot.Breadth.Validate(); err != nil {
		return nil, err
	}
	if snapshot.Breadth.TradeDate != snapshot.AsOf {
		return nil, fmt.Errorf("breadth trade date %q does not match as-of %q", snapshot.Breadth.TradeDate, snapshot.AsOf)
	}

	byCode := make(map[string]domain.IndexSnapshot, len(snapshot.Indices))
	for _, index := range snapshot.Indices {
		if err := index.Validate(); err != nil {
			return nil, err
		}
		if index.TradeDate != snapshot.AsOf || index.ObservedAt != snapshot.ObservedAt {
			return nil, fmt.Errorf("index %q has inconsistent observation point", index.Code)
		}
		if _, exists := byCode[index.Code]; exists {
			return nil, fmt.Errorf("duplicate index %q", index.Code)
		}
		byCode[index.Code] = index
	}

	ordered := make([]domain.IndexSnapshot, 0, len(overviewIndexOrder))
	for _, code := range overviewIndexOrder {
		index, exists := byCode[code]
		if !exists {
			return nil, fmt.Errorf("required index %q is missing", code)
		}
		ordered = append(ordered, index)
	}
	if len(byCode) != len(overviewIndexOrder) {
		return nil, fmt.Errorf("unexpected index count %d", len(byCode))
	}
	return ordered, nil
}

func buildMarketOverview(snapshot OverviewSnapshot, indices []domain.IndexSnapshot, selection ProviderSelection) MarketOverview {
	result := MarketOverview{
		AsOf:       snapshot.AsOf,
		ObservedAt: snapshot.ObservedAt,
		Source: DataSource{
			Mode:        selection.Mode,
			Provider:    selection.Provider,
			SeedVersion: snapshot.SeedVersion,
		},
		Indices:  make([]IndexOverview, 0, len(indices)),
		Breadth:  BreadthOverview{Advancing: snapshot.Breadth.Advancing, Declining: snapshot.Breadth.Declining, Unchanged: snapshot.Breadth.Unchanged},
		Turnover: TurnoverOverview{Amount: snapshot.Breadth.TurnoverAmount, Currency: "CNY"},
	}
	for _, index := range indices {
		result.Indices = append(result.Indices, IndexOverview{
			Code: index.Code, Name: index.Name, Close: index.Close,
			Change: index.Change, ChangePercent: index.ChangePercent,
		})
	}
	return result
}
