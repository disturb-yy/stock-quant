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

// TestMySQLStockFinancialsReader 只在显式提供测试数据库时验证真实财务报告查询。
func TestMySQLStockFinancialsReader(t *testing.T) {
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
	reader, err := NewMySQLFinancialsReader(database)
	if err != nil {
		t.Fatalf("NewMySQLFinancialsReader() error = %v", err)
	}

	annual, err := reader.ReadStockFinancials(context.Background(), stock.FinancialsRequest{Symbol: "000001.SZ", Period: stock.FinancialPeriodAnnual, Range: stock.FinancialRangeFiveYears})
	if err != nil {
		t.Fatalf("ReadStockFinancials annual: %v", err)
	}
	if annual.Symbol != "000001.SZ" || annual.Name != "平安银行" || annual.SeedVersion != demo.SeedVersion || annual.AsOf != demo.SeedAsOf {
		t.Fatalf("annual snapshot identity/source = %#v, want seeded identity/source", annual)
	}
	if len(annual.Reports) != 6 || annual.Reports[0].PeriodEnd != "2018-12-31" || annual.Reports[len(annual.Reports)-1].PeriodEnd != "2023-12-31" {
		t.Fatalf("annual reports = %d/%q..%q, want all seeded reports in ascending order", len(annual.Reports), annual.Reports[0].PeriodEnd, annual.Reports[len(annual.Reports)-1].PeriodEnd)
	}
	quarterly, err := reader.ReadStockFinancials(context.Background(), stock.FinancialsRequest{Symbol: "000001.SZ", Period: stock.FinancialPeriodQuarterly, Range: stock.FinancialRangeFiveYears})
	if err != nil {
		t.Fatalf("ReadStockFinancials quarterly: %v", err)
	}
	if len(quarterly.Reports) != 21 || quarterly.Reports[0].FiscalQuarter != "Q1" || quarterly.Reports[len(quarterly.Reports)-1].PeriodEnd != "2024-03-31" {
		t.Fatalf("quarterly reports = %d, want 21 seeded periods through 2024Q1", len(quarterly.Reports))
	}
	empty, err := reader.ReadStockFinancials(context.Background(), stock.FinancialsRequest{Symbol: "300750.SZ", Period: stock.FinancialPeriodAnnual, Range: stock.FinancialRangeFiveYears})
	if err != nil {
		t.Fatalf("ReadStockFinancials empty result: %v", err)
	}
	if empty.Symbol != "300750.SZ" || len(empty.Reports) != 0 {
		t.Fatalf("empty snapshot = %#v, want existing stock with no reports", empty)
	}
	_, err = reader.ReadStockFinancials(context.Background(), stock.FinancialsRequest{Symbol: "999999.SZ", Period: stock.FinancialPeriodAnnual, Range: stock.FinancialRangeFiveYears})
	if !errors.Is(err, stock.ErrFinancialsInstrumentNotFound) {
		t.Fatalf("unknown stock error = %v, want ErrFinancialsInstrumentNotFound", err)
	}
}
