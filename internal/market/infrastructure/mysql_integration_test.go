package infrastructure

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/disturb-yy/stock-quant/internal/demo"
	demoinfrastructure "github.com/disturb-yy/stock-quant/internal/demo/infrastructure"
	"github.com/disturb-yy/stock-quant/internal/market"
)

// TestMySQLOverviewReader 只在显式提供测试数据库时验证真实查询和聚合结果。
func TestMySQLOverviewReader(t *testing.T) {
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
	if err := store.SeedDemo(context.Background(), demo.DemoFixture()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	reader, err := NewMySQLOverviewReader(database)
	if err != nil {
		t.Fatalf("NewMySQLOverviewReader() error = %v", err)
	}
	snapshot, err := reader.ReadOverviewSnapshot(context.Background())
	if err != nil {
		t.Fatalf("ReadOverviewSnapshot() error = %v", err)
	}
	if len(snapshot.Indices) != 4 {
		t.Fatalf("index count = %d, want 4", len(snapshot.Indices))
	}
	if snapshot.Breadth.Advancing != 2 || snapshot.Breadth.Declining != 1 || snapshot.Breadth.Unchanged != 0 {
		t.Fatalf("breadth = %#v, want 2/1/0", snapshot.Breadth)
	}
	if snapshot.Breadth.TurnoverAmount != "12002494200.00" {
		t.Fatalf("turnover = %q, want %q", snapshot.Breadth.TurnoverAmount, "12002494200.00")
	}
	service, err := market.NewOverviewService(reader, market.ProviderSelection{Mode: market.ModeDemo, Provider: market.DemoProviderName})
	if err != nil {
		t.Fatalf("NewOverviewService() error = %v", err)
	}
	if _, err := service.Overview(context.Background()); err != nil {
		t.Fatalf("Overview() error = %v", err)
	}
}
