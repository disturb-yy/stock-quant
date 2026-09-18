package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

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
	savedStore, err := NewMySQLSavedScreenerStore(database)
	if err != nil {
		t.Fatalf("initialize saved screener store: %v", err)
	}
	if err := savedStore.Migrate(context.Background()); err != nil {
		t.Fatalf("saved screener migration: %v", err)
	}
	if err := savedStore.Migrate(context.Background()); err != nil {
		t.Fatalf("repeat saved screener migration: %v", err)
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
	savedService, err := screener.NewSavedScreenerService(savedStore)
	if err != nil {
		t.Fatalf("NewSavedScreenerService() error = %v", err)
	}
	name := fmt.Sprintf("集成测试方案-%d", time.Now().UnixNano())
	description := "仅保存规范化条件"
	saved, err := savedService.Create(context.Background(), domain.SavedScreenerInput{
		Name: name, Description: &description,
		Spec: domain.ScreenerSpec{UniverseID: domain.ActiveAShareUniverse, Filters: []domain.Filter{{FieldID: "valuation.pe_ttm", Operator: domain.OperatorLessEqual, Value: "15"}}, Ranking: domain.Ranking{FieldID: "technical.volume", Direction: "desc"}, TopN: 3},
	})
	if err != nil {
		t.Fatalf("create saved screener: %v", err)
	}
	loaded, err := savedService.Get(context.Background(), saved.ID)
	if err != nil {
		t.Fatalf("get saved screener: %v", err)
	}
	if loaded.Version != 1 || loaded.Name != name || loaded.Spec.Filters[0].Value != "15" {
		t.Fatalf("loaded saved screener = %#v, want canonical version 1", loaded)
	}
	updatedDescription := "更新后的条件"
	updated, err := savedService.Update(context.Background(), saved.ID, saved.Version, domain.SavedScreenerInput{
		Name: name, Description: &updatedDescription,
		Spec: domain.ScreenerSpec{UniverseID: domain.ActiveAShareUniverse, Filters: []domain.Filter{}, Ranking: domain.Ranking{FieldID: "technical.close", Direction: "asc"}, TopN: 1},
	})
	if err != nil {
		t.Fatalf("update saved screener: %v", err)
	}
	if updated.Version != 2 || updated.Description == nil || *updated.Description != updatedDescription {
		t.Fatalf("updated saved screener = %#v, want version 2", updated)
	}
	_, err = savedService.Update(context.Background(), saved.ID, saved.Version, domain.SavedScreenerInput{
		Name: name, Spec: updated.Spec,
	})
	var conflict *domain.SavedScreenerVersionConflictError
	if !errors.As(err, &conflict) || conflict.CurrentVersion != 2 {
		t.Fatalf("stale update error = %v, want current version conflict", err)
	}
	var versionCount int
	if err := database.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM t_screener_version WHERE screener_id = ?`, saved.ID).Scan(&versionCount); err != nil {
		t.Fatalf("count saved screener versions: %v", err)
	}
	if versionCount != 2 {
		t.Fatalf("saved screener version count = %d, want 2 after stale update", versionCount)
	}
}
