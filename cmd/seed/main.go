// Command seed 将版本化的开发 fixture 幂等写入本地 MySQL。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/disturb-yy/stock-quant/internal/demo"
	"github.com/disturb-yy/stock-quant/internal/demo/infrastructure"
	"github.com/disturb-yy/stock-quant/pkg/config"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "seed demo fixture: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	database, err := infrastructure.Open(ctx, config.LoadDatabase())
	if err != nil {
		return err
	}
	defer database.Close()
	store, err := infrastructure.NewStore(database)
	if err != nil {
		return err
	}
	if err := store.Migrate(ctx); err != nil {
		return err
	}
	fixture := demo.DemoFixture()
	seeder, err := demo.NewSeeder(store)
	if err != nil {
		return err
	}
	if err := seeder.Seed(ctx, fixture); err != nil {
		return err
	}
	snapshot, err := store.ReadDemoSnapshot(ctx)
	if err != nil {
		return fmt.Errorf("read seeded snapshot: %w", err)
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"seed_version":      snapshot.SeedVersion,
		"as_of":             snapshot.AsOf,
		"instruments":       snapshot.Counts.Instruments,
		"daily_bars":        snapshot.Counts.DailyBars,
		"financial_metrics": snapshot.Counts.FinancialMetrics,
		"index_snapshots":   snapshot.Counts.IndexSnapshots,
	})
}
