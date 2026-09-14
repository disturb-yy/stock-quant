package infrastructure

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/disturb-yy/stock-quant/internal/demo"
)

// TestMySQLMigrationAndSeedAreIdempotent 只在显式提供测试数据库时运行。
func TestMySQLMigrationAndSeedAreIdempotent(t *testing.T) {
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
	store, err := NewStore(database)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("first migration: %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("second migration: %v", err)
	}
	fixture := demo.DemoFixture()
	seeder, err := demo.NewSeeder(store)
	if err != nil {
		t.Fatalf("NewSeeder() error = %v", err)
	}
	if err := seeder.Seed(context.Background(), fixture); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	first, err := store.ReadDemoSnapshot(context.Background())
	if err != nil {
		t.Fatalf("read first snapshot: %v", err)
	}
	if err := seeder.Seed(context.Background(), fixture); err != nil {
		t.Fatalf("second seed: %v", err)
	}
	second, err := store.ReadDemoSnapshot(context.Background())
	if err != nil {
		t.Fatalf("read second snapshot: %v", err)
	}
	if first.SeedVersion != second.SeedVersion || first.Counts != second.Counts || len(first.Samples) != len(second.Samples) {
		t.Fatalf("snapshot changed after repeated seed: first=%#v second=%#v", first, second)
	}
	if first.Counts != fixture.DataCounts() {
		t.Fatalf("seed counts = %#v, want %#v", first.Counts, fixture.DataCounts())
	}
}
