package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/disturb-yy/stock-quant/internal/pool"
	"github.com/disturb-yy/stock-quant/internal/pool/domain"
	_ "github.com/go-sql-driver/mysql"
)

// TestMySQLStockPoolStore 只在显式测试 DSN 下验证真实迁移、Seed 与持久化。
func TestMySQLStockPoolStore(t *testing.T) {
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
	store, err := NewMySQLStockPoolStore(database)
	if err != nil {
		t.Fatalf("NewMySQLStockPoolStore() error = %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("first migration: %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
	if err := store.SeedDemo(context.Background()); err != nil {
		t.Fatalf("first demo seed: %v", err)
	}
	if err := store.SeedDemo(context.Background()); err != nil {
		t.Fatalf("repeat demo seed: %v", err)
	}
	service, err := pool.NewService(store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	name := fmt.Sprintf("集成测试池-%d", time.Now().UnixNano())
	created, err := service.Create(context.Background(), domain.StockPoolInput{Name: name + "-1"})
	if err != nil {
		t.Fatalf("create stock pool: %v", err)
	}
	if created.ID < 1 || created.Source != domain.SourceManual || created.MemberCount != 0 || created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatalf("created pool = %#v, want persisted manual pool", created)
	}
	newer, err := service.Create(context.Background(), domain.StockPoolInput{Name: name + "-2"})
	if err != nil {
		t.Fatalf("create second stock pool: %v", err)
	}
	list, err := service.List(context.Background(), pool.StockPoolListRequest{Search: name, Page: 1, PageSize: 1})
	if err != nil {
		t.Fatalf("search stock pool: %v", err)
	}
	if list.Pagination.Total != 2 || len(list.Data) != 1 || list.Data[0].ID != newer.ID {
		t.Fatalf("list = %#v, want newer pool on first page", list)
	}
	secondPage, err := service.List(context.Background(), pool.StockPoolListRequest{Search: name, Page: 2, PageSize: 1})
	if err != nil {
		t.Fatalf("read second page: %v", err)
	}
	if len(secondPage.Data) != 1 || secondPage.Data[0].ID != created.ID {
		t.Fatalf("second page = %#v, want older pool", secondPage)
	}
	loaded, err := service.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get stock pool: %v", err)
	}
	if loaded.Name != name+"-1" || loaded.Source != domain.SourceManual || loaded.MemberCount != 0 {
		t.Fatalf("loaded pool = %#v, want persisted metadata", loaded)
	}
}
