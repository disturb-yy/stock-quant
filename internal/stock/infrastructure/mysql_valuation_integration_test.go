package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/disturb-yy/stock-quant/internal/demo"
	demoinfrastructure "github.com/disturb-yy/stock-quant/internal/demo/infrastructure"
	"github.com/disturb-yy/stock-quant/internal/stock"
	_ "github.com/go-sql-driver/mysql"
)

// TestMySQLStockValuationReader 只在显式提供测试数据库时验证真实估值读取、计算和空态。
func TestMySQLStockValuationReader(t *testing.T) {
	dsn := os.Getenv("STOCK_QUANT_TEST_DSN")
	if dsn == "" {
		t.Skip("set STOCK_QUANT_TEST_DSN to run the MySQL integration test")
	}
	database, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open test MySQL: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := database.PingContext(context.Background()); err != nil {
		t.Fatalf("ping test MySQL: %v", err)
	}
	store, err := demoinfrastructure.NewStore(database)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migration: %v", err)
	}
	fixture := demo.DemoFixture()
	if err := store.SeedDemo(context.Background(), fixture); err != nil {
		t.Fatalf("seed: %v", err)
	}
	reader, err := NewMySQLValuationReader(database)
	if err != nil {
		t.Fatalf("NewMySQLValuationReader() error = %v", err)
	}
	service, err := stock.NewValuationService(reader, stock.ValuationSource{Mode: "demo", Provider: "mysql-demo-fixture"})
	if err != nil {
		t.Fatalf("NewValuationService() error = %v", err)
	}
	result, err := service.Valuation(context.Background(), stock.ValuationRequest{Symbol: "000001.SZ", Range: stock.ValuationRangeThreeYears})
	if err != nil {
		t.Fatalf("valuation query: %v", err)
	}
	if result.Source.SeedVersion != demo.SeedVersion || result.AsOf == nil || *result.AsOf != demo.SeedAsOf || len(result.Metrics.PETTM.History) != 4 {
		t.Fatalf("valuation identity/range = %#v, want v8/latest/4 three-year history", result)
	}
	if len(result.IndustryComparisons) != 1 || result.IndustryComparisons[0].Metrics.PETTM.SampleSize != 3 || result.IndustryComparisons[0].Metrics.PETTM.Value == nil || *result.IndustryComparisons[0].Metrics.PETTM.Value != "6.00" {
		t.Fatalf("industry comparison = %#v, want three-peer median", result.IndustryComparisons)
	}
	negative, err := service.Valuation(context.Background(), stock.ValuationRequest{Symbol: "300750.SZ"})
	if err != nil {
		t.Fatalf("negative valuation query: %v", err)
	}
	if negative.Metrics.PETTM.Current.Value == nil || *negative.Metrics.PETTM.Current.Value != "-1.00" || negative.Metrics.PETTM.Percentile.Value != nil {
		t.Fatalf("negative PE = %#v, want raw current and null percentile", negative.Metrics.PETTM)
	}
	empty, err := service.Valuation(context.Background(), stock.ValuationRequest{Symbol: "000002.SZ"})
	if err != nil {
		t.Fatalf("empty valuation query: %v", err)
	}
	if empty.AsOf != nil || empty.EffectiveRange.From != nil || empty.Metrics.PETTM.Current.Value != nil {
		t.Fatalf("empty valuation = %#v, want 200-compatible empty structure", empty)
	}
	_, err = service.Valuation(context.Background(), stock.ValuationRequest{Symbol: "999999.SZ"})
	if !errors.Is(err, stock.ErrValuationInstrumentNotFound) {
		t.Fatalf("unknown stock error = %v, want ErrValuationInstrumentNotFound", err)
	}
}
