package main

import (
	"context"
	"flag"
	"log"
	"time"

	runtimeconfig "github.com/disturb-yy/stock-quant/internal/config"
	dbmysql "github.com/disturb-yy/stock-quant/internal/infrastructure/database/mysql"
	"github.com/disturb-yy/stock-quant/internal/infrastructure/migration"
	"github.com/disturb-yy/stock-quant/migrations"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	direction := flag.String("direction", string(migration.DirectionUp), "migration direction: up or down")
	flag.Parse()
	dsn, err := runtimeconfig.LoadDatabaseDSNFromEnv()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := dbmysql.Open(ctx, dsn)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	if err := migration.Run(ctx, db, migrations.FS, migration.Direction(*direction)); err != nil {
		return err
	}
	return nil
}
