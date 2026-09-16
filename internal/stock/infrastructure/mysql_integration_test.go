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

// TestMySQLStockOverviewReader 只在显式提供测试数据库时验证真实聚合查询。
func TestMySQLStockOverviewReader(t *testing.T) {
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
	overview, err := reader.ReadStockOverview(context.Background(), "000001.SZ")
	if err != nil {
		t.Fatalf("ReadStockOverview() error = %v", err)
	}
	if overview.Symbol != "000001.SZ" || overview.Name != "平安银行" || overview.Industry != "银行" {
		t.Fatalf("identity = %#v, want seeded stock identity", overview)
	}
	if overview.Quote.AsOf != demo.SeedAsOf || overview.Quote.Last != "10.31" {
		t.Fatalf("quote = %#v, want same-day latest quote", overview.Quote)
	}
	if overview.PETTM.Value == nil || overview.PB.Value == nil || overview.ROE.Value == nil || overview.MarketCap.Value == nil {
		t.Fatalf("metrics = %#v, want four seeded values", overview)
	}
	if overview.PETTM.Basis != "ttm" || overview.ROE.Basis != "latest_report" || overview.PB.Basis != "latest_daily_basic" {
		t.Fatalf("metric basis = %#v/%#v/%#v, want explicit basis", overview.PETTM, overview.PB, overview.ROE)
	}
	if len(overview.Sparkline) != 20 || overview.Sparkline[len(overview.Sparkline)-1].TradeDate != demo.SeedAsOf {
		t.Fatalf("sparkline = %d points, want 20 ending at %s", len(overview.Sparkline), demo.SeedAsOf)
	}
	lastPoint := overview.Sparkline[len(overview.Sparkline)-1]
	if lastPoint.Open == "" || lastPoint.High == "" || lastPoint.Low == "" || lastPoint.Close == "" {
		t.Fatalf("sparkline OHLC = %#v, want seeded daily bar values", lastPoint)
	}
	_, err = reader.ReadStockOverview(context.Background(), "999999.SZ")
	if !errors.Is(err, stock.ErrInstrumentNotFound) {
		t.Fatalf("unknown stock error = %v, want ErrInstrumentNotFound", err)
	}
}
