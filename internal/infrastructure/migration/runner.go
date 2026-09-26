package migration

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	migratemysql "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"
)

type Direction string

const (
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"
)

// Run 使用调用方提供的 schema FS 执行迁移，避免共享层依赖具体 domain 的表语义。
func Run(ctx context.Context, db *gorm.DB, migrationFS fs.FS, direction Direction) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if db == nil {
		return fmt.Errorf("migration database is nil")
	}
	if migrationFS == nil {
		return fmt.Errorf("migration filesystem is nil")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get migration sql database: %w", err)
	}
	source, err := iofs.New(migrationFS, ".")
	if err != nil {
		return fmt.Errorf("open migrations: %w", err)
	}
	driver, err := migratemysql.WithInstance(sqlDB, &migratemysql.Config{})
	if err != nil {
		return fmt.Errorf("create mysql migration driver: %w", err)
	}
	migrator, err := migrate.NewWithInstance("iofs", source, "mysql", driver)
	if err != nil {
		return fmt.Errorf("create migration runner: %w", err)
	}
	defer migrator.Close()
	return runDirection(migrator, direction)
}

func runDirection(migrator *migrate.Migrate, direction Direction) error {
	switch direction {
	case DirectionUp:
		if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("migrate up: %w", err)
		}
	case DirectionDown:
		if err := migrator.Steps(-1); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("migrate down: %w", err)
		}
	default:
		return fmt.Errorf("unsupported migration direction %q", direction)
	}
	return nil
}
