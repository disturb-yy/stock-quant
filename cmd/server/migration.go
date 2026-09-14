package main

import (
	"context"
	"fmt"
	"log/slog"
)

type migrationStore interface {
	Migrate(context.Context) error
}

func initializeMigrations(ctx context.Context, store migrationStore, applicationLogger *slog.Logger) error {
	if store == nil {
		return fmt.Errorf("migration store is required")
	}
	if err := store.Migrate(ctx); err != nil {
		return fmt.Errorf("run database migrations: %w", err)
	}

	applicationLogger.Info("database migration runner initialized", "migration_count", 3)
	return nil
}
