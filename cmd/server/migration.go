package main

import (
	"context"
	"fmt"
	"log/slog"

	"example.com/stock-ddd/internal/migration"
)

func initializeMigrations(applicationLogger *slog.Logger) error {
	runner, err := migration.NewRunner()
	if err != nil {
		return fmt.Errorf("initialize migration runner: %w", err)
	}
	if err := runner.Run(context.Background()); err != nil {
		return fmt.Errorf("run database migrations: %w", err)
	}

	applicationLogger.Info("database migration runner initialized", "migration_count", runner.Count())
	return nil
}
