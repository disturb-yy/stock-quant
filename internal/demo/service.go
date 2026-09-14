package demo

import (
	"context"
	"fmt"

	"github.com/disturb-yy/stock-quant/internal/market"
)

// SampleStock 是状态接口返回的样本股票摘要。
type SampleStock struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Exchange string `json:"exchange"`
	Status   string `json:"status"`
}

// StoreSnapshot 是 Infrastructure 返回的只读数据库快照。
type StoreSnapshot struct {
	SeedVersion string
	AsOf        *string
	Counts      Counts
	Samples     []SampleStock
}

// DemoStatus 是 GET /api/v1/dev/demo-status 的成功响应。
type DemoStatus struct {
	Mode         market.ProviderMode `json:"mode"`
	Provider     string              `json:"provider"`
	SeedVersion  string              `json:"seed_version"`
	AsOf         *string             `json:"as_of"`
	Counts       Counts              `json:"counts"`
	SampleStocks []SampleStock       `json:"sample_stocks"`
}

// StatusStore 是状态查询所需的最小存储边界。
type StatusStore interface {
	ReadDemoSnapshot(context.Context) (StoreSnapshot, error)
}

// SeedStore 是演示数据写入所需的最小存储边界。
type SeedStore interface {
	SeedDemo(context.Context, Fixture) error
}

// StatusReader 是 HTTP 层依赖的状态查询接口。
type StatusReader interface {
	DemoStatus(context.Context) (DemoStatus, error)
}

// Service 编排演示状态查询。
type Service struct {
	store             StatusStore
	requestedProvider string
}

// NewService 创建演示状态服务。
func NewService(store StatusStore, requestedProvider string) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("demo status store is required")
	}
	return &Service{store: store, requestedProvider: requestedProvider}, nil
}

// DemoStatus 读取真实数据库快照并解析 demo/real/fallback 模式。
func (service *Service) DemoStatus(ctx context.Context) (DemoStatus, error) {
	snapshot, err := service.store.ReadDemoSnapshot(ctx)
	if err != nil {
		return DemoStatus{}, fmt.Errorf("read demo status snapshot: %w", err)
	}
	fixtureReady := snapshot.SeedVersion == SeedVersion && snapshot.AsOf != nil && *snapshot.AsOf == SeedAsOf && snapshot.Counts.Equal(DemoFixture().DataCounts())
	selection := market.SelectProvider(service.requestedProvider, fixtureReady)
	return DemoStatus{
		Mode:         selection.Mode,
		Provider:     selection.Provider,
		SeedVersion:  snapshot.SeedVersion,
		AsOf:         snapshot.AsOf,
		Counts:       snapshot.Counts,
		SampleStocks: snapshot.Samples,
	}, nil
}

// Seeder 编排一个版本化 fixture 的写入。
type Seeder struct {
	store SeedStore
}

// NewSeeder 创建演示数据 seed 用例。
func NewSeeder(store SeedStore) (*Seeder, error) {
	if store == nil {
		return nil, fmt.Errorf("demo seed store is required")
	}
	return &Seeder{store: store}, nil
}

// Seed 写入确定的 fixture；重复调用由 Infrastructure 的唯一键保证幂等。
func (seeder *Seeder) Seed(ctx context.Context, fixture Fixture) error {
	if err := fixture.Validate(); err != nil {
		return fmt.Errorf("validate demo fixture: %w", err)
	}
	if err := seeder.store.SeedDemo(ctx, fixture); err != nil {
		return fmt.Errorf("seed demo fixture: %w", err)
	}
	return nil
}
