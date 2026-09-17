package infrastructure

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/disturb-yy/stock-quant/internal/demo"
	demoinfrastructure "github.com/disturb-yy/stock-quant/internal/demo/infrastructure"
	"github.com/disturb-yy/stock-quant/internal/screener"
	"github.com/disturb-yy/stock-quant/internal/screener/domain"
	_ "github.com/go-sql-driver/mysql"
)

// TestMySQLScreenerReaderAndService 只在显式提供测试数据库时验证真实快照查询。
func TestMySQLScreenerReaderAndService(t *testing.T) {
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
		t.Fatalf("initialize demo store: %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migration: %v", err)
	}
	fixture := demo.DemoFixture()
	if err := store.SeedDemo(context.Background(), fixture); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := store.SeedDemo(context.Background(), fixture); err != nil {
		t.Fatalf("repeat seed: %v", err)
	}
	reader, err := NewMySQLReader(database, domain.Source{Mode: "demo", Provider: "mysql-demo-fixture"})
	if err != nil {
		t.Fatalf("NewMySQLReader() error = %v", err)
	}
	input, err := reader.ReadSnapshot(context.Background(), []string{"technical.volume", "valuation.pe_ttm"})
	if err != nil {
		t.Fatalf("ReadSnapshot() error = %v", err)
	}
	if input.Universe.ID != domain.ActiveAShareUniverse || len(input.Eligible) != len(fixture.Instruments) || input.Source.SeedVersion != demo.SeedVersion || input.Snapshot.AsOf != demo.SeedAsOf {
		t.Fatalf("snapshot = %#v, want seeded universe/source", input)
	}
	service, err := screener.NewService(reader)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	result, err := service.Run(context.Background(), screener.ScreenerRunRequest{Spec: domain.ScreenerSpec{UniverseID: domain.ActiveAShareUniverse, Filters: []domain.Filter{{FieldID: "valuation.pe_ttm", Operator: domain.OperatorLessEqual, Value: "15"}}, Ranking: domain.Ranking{FieldID: "technical.volume", Direction: "desc"}, TopN: 3}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.MatchedCount == 0 || result.ReturnedCount != 3 || result.Results[0].Ranking.Value == nil {
		t.Fatalf("result = %#v, want real ranked rows", result)
	}
	empty, err := service.Run(context.Background(), screener.ScreenerRunRequest{Spec: domain.ScreenerSpec{UniverseID: domain.ActiveAShareUniverse, Ranking: domain.Ranking{FieldID: "technical.close", Direction: "desc"}, Filters: []domain.Filter{{FieldID: "technical.close", Operator: domain.OperatorGreater, Value: "999999"}}, TopN: 1}})
	if err != nil {
		t.Fatalf("empty Run() error = %v", err)
	}
	if empty.MatchedCount != 0 || empty.Results == nil || len(empty.Results) != 0 {
		t.Fatalf("empty result = %#v, want 200-compatible empty result", empty)
	}
}
