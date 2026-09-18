package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/disturb-yy/stock-quant/internal/pool"
	"github.com/disturb-yy/stock-quant/internal/pool/domain"
	stockinfrastructure "github.com/disturb-yy/stock-quant/internal/stock/infrastructure"
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
	if _, err := database.ExecContext(context.Background(), `
        CREATE TABLE IF NOT EXISTS instruments (
            code VARCHAR(32) NOT NULL PRIMARY KEY,
            name VARCHAR(128) NOT NULL,
            exchange VARCHAR(16) NOT NULL,
            status VARCHAR(32) NOT NULL,
            as_of DATE NOT NULL
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
		t.Fatalf("create instrument fixture: %v", err)
	}
	if _, err := database.ExecContext(context.Background(), `
        INSERT INTO instruments (code, name, exchange, status, as_of)
        VALUES ('000001.SZ', '平安银行', 'SZSE', 'active', '2024-06-28')
        ON DUPLICATE KEY UPDATE name = VALUES(name)`); err != nil {
		t.Fatalf("seed instrument fixture: %v", err)
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
	stockReader, err := stockinfrastructure.NewMySQLOverviewReader(database)
	if err != nil {
		t.Fatalf("NewMySQLOverviewReader() error = %v", err)
	}
	service, err := pool.NewService(store, stockReader)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	demoPools, err := service.List(context.Background(), pool.StockPoolListRequest{Search: "演示手工股票池"})
	if err != nil || len(demoPools.Data) != 1 || demoPools.Data[0].MemberCount != 1 {
		t.Fatalf("demo pools = %#v, error = %v, want one member", demoPools, err)
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
	added, err := service.AddMember(context.Background(), created.ID, "000001.SZ")
	if err != nil || added.Member.Symbol != "000001.SZ" || added.Member.Name != "平安银行" || added.MemberCount != 1 {
		t.Fatalf("added member = %#v, error = %v", added, err)
	}
	if _, err := service.AddMember(context.Background(), created.ID, "000001.SZ"); !errors.Is(err, domain.ErrStockPoolMemberConflict) {
		t.Fatalf("duplicate member error = %v, want conflict", err)
	}
	members, err := service.ListMembers(context.Background(), created.ID, pool.StockPoolMemberListRequest{Page: 1, PageSize: 20})
	if err != nil || len(members.Data) != 1 || members.Data[0].Symbol != "000001.SZ" {
		t.Fatalf("members = %#v, error = %v", members, err)
	}
	deleted, err := service.DeleteMember(context.Background(), created.ID, "000001.SZ")
	if err != nil || deleted.MemberCount != 0 || deleted.Symbol != "000001.SZ" {
		t.Fatalf("deleted member = %#v, error = %v", deleted, err)
	}
	if _, err := service.DeleteMember(context.Background(), created.ID, "000001.SZ"); !errors.Is(err, domain.ErrStockPoolMemberNotFound) {
		t.Fatalf("missing member error = %v, want not found", err)
	}
}
