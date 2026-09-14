package market

import (
	"context"
	"fmt"
	"strings"

	"github.com/disturb-yy/stock-quant/internal/market/domain"
)

// SectorSnapshot 是 Provider 返回的行业成分行情快照。
type SectorSnapshot struct {
	SeedVersion string
	AsOf        string
	Components  []domain.SectorComponent
}

// SectorReader 是行业表现查询所需的最小数据访问边界。
type SectorReader interface {
	ReadSectorSnapshot(context.Context) (SectorSnapshot, error)
}

// MarketSectors 是 GET /api/v1/markets/sectors 的成功响应。
type MarketSectors struct {
	AsOf    string           `json:"as_of"`
	Source  DataSource       `json:"source"`
	Sectors []SectorOverview `json:"sectors"`
}

// SectorOverview 是前端展示用的行业表现。
type SectorOverview struct {
	Code           string               `json:"code"`
	Name           string               `json:"name"`
	ChangePercent  string               `json:"change_percent"`
	ComponentCount int64                `json:"component_count"`
	Leader         SectorLeaderOverview `json:"leader"`
}

// SectorLeaderOverview 是行业内涨幅最高的股票。
type SectorLeaderOverview struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	ChangePercent string `json:"change_percent"`
}

// SectorService 编排行业表现查询和结果完整性校验。
type SectorService struct {
	reader    SectorReader
	selection ProviderSelection
}

// NewSectorService 创建行业表现查询服务。
func NewSectorService(reader SectorReader, selection ProviderSelection) (*SectorService, error) {
	if reader == nil {
		return nil, fmt.Errorf("market sector reader is required")
	}
	if strings.TrimSpace(selection.Provider) == "" {
		return nil, fmt.Errorf("market sector provider is required")
	}
	return &SectorService{reader: reader, selection: selection}, nil
}

// Sectors 读取最新交易日并聚合行业表现。
func (service *SectorService) Sectors(ctx context.Context) (MarketSectors, error) {
	snapshot, err := service.reader.ReadSectorSnapshot(ctx)
	if err != nil {
		return MarketSectors{}, fmt.Errorf("read market sector snapshot: %w", err)
	}
	if err := validateSectorSnapshot(snapshot); err != nil {
		return MarketSectors{}, fmt.Errorf("validate market sector snapshot: %w", err)
	}
	performances, err := domain.AggregateSectorPerformances(snapshot.Components)
	if err != nil {
		return MarketSectors{}, fmt.Errorf("aggregate market sector performance: %w", err)
	}
	return buildMarketSectors(snapshot, performances, service.selection), nil
}

func validateSectorSnapshot(snapshot SectorSnapshot) error {
	if strings.TrimSpace(snapshot.AsOf) == "" || strings.TrimSpace(snapshot.SeedVersion) == "" {
		return fmt.Errorf("sector as-of and seed version are required")
	}
	if len(snapshot.Components) == 0 {
		return fmt.Errorf("sector components are empty")
	}
	for _, component := range snapshot.Components {
		if err := component.Validate(); err != nil {
			return err
		}
		if component.TradeDate != snapshot.AsOf {
			return fmt.Errorf("sector component %q has trade date %q, want %q", component.InstrumentCode, component.TradeDate, snapshot.AsOf)
		}
	}
	return nil
}

func buildMarketSectors(snapshot SectorSnapshot, performances []domain.SectorPerformance, selection ProviderSelection) MarketSectors {
	result := MarketSectors{
		AsOf: snapshot.AsOf,
		Source: DataSource{
			Mode:        selection.Mode,
			Provider:    selection.Provider,
			SeedVersion: snapshot.SeedVersion,
		},
		Sectors: make([]SectorOverview, 0, len(performances)),
	}
	for _, performance := range performances {
		result.Sectors = append(result.Sectors, SectorOverview{
			Code:           performance.Code,
			Name:           performance.Name,
			ChangePercent:  performance.ChangePercent,
			ComponentCount: performance.ComponentCount,
			Leader: SectorLeaderOverview{
				Code:          performance.Leader.Code,
				Name:          performance.Leader.Name,
				ChangePercent: performance.Leader.ChangePercent,
			},
		})
	}
	return result
}
