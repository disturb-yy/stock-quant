package main

import (
	"context"
	"fmt"
	"log/slog"
)

type migrationStore interface {
	Migrate(context.Context) error
}

func initializeMigrations(ctx context.Context, store migrationStore, applicationLogger *slog.Logger, additionalStores ...migrationStore) error {
	stores := append([]migrationStore{store}, additionalStores...)
	for index, currentStore := range stores {
		if currentStore == nil {
			return fmt.Errorf("migration store %d is required", index+1)
		}
		if err := currentStore.Migrate(ctx); err != nil {
			return fmt.Errorf("run database migrations for store %d: %w", index+1, err)
		}
	}

	applicationLogger.Info("database migration runners initialized", "migration_store_count", len(stores))
	return nil
}
