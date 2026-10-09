package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"stock-quant/db/migrations"
	"stock-quant/internal/app"
	mysqlinfra "stock-quant/internal/infrastructure/mysql"

	mysqldriver "github.com/go-sql-driver/mysql"
)

type healthStatus struct {
	Status string `json:"status"`
	Scope  string `json:"scope"`
}

type healthChecker interface {
	Check(context.Context) (app.HealthStatus, error)
}

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "migrate" {
		if err := runMigrationCommand(context.Background(), args[1:], os.Getenv("APP_ENV"), migrationDSNFromEnv(), os.Stdout); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := run(context.Background(), args, os.Stdout, app.HealthService{}); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output io.Writer, checker healthChecker) error {
	if len(args) != 1 || args[0] != "health" {
		return fmt.Errorf("usage: stockquant health")
	}

	status, err := checker.Check(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	response := healthStatus{Status: status.Status, Scope: status.Scope}
	if err := json.NewEncoder(output).Encode(response); err != nil {
		return fmt.Errorf("write health output: %w", err)
	}
	return nil
}

func validateMigrationCommand(action, appEnv string) error {
	if action != "up" && action != "down" {
		return fmt.Errorf("usage: stockquant migrate up|down")
	}
	if action == "down" {
		switch appEnv {
		case "development", "test":
		case "":
			return fmt.Errorf("migrate down requires APP_ENV=development or APP_ENV=test")
		default:
			return fmt.Errorf("migrate down is only allowed when APP_ENV=development or APP_ENV=test")
		}
	}
	return nil
}

func runMigrationCommand(ctx context.Context, args []string, appEnv, dsn string, output io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: stockquant migrate up|down")
	}
	action := args[0]
	if err := validateMigrationCommand(action, appEnv); err != nil {
		return err
	}
	if strings.TrimSpace(dsn) == "" {
		return fmt.Errorf("database connection is not configured; set MYSQL_DSN or DB_* variables")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open database connection: %w", err)
	}
	defer db.Close()
	items, err := mysqlinfra.LoadMigrations(migrations.FS)
	if err != nil {
		return fmt.Errorf("load database migrations: %w", err)
	}
	migrator := &mysqlinfra.Migrator{DB: db, Migrations: items}
	switch action {
	case "up":
		err = migrator.Up(ctx)
	case "down":
		err = migrator.Down(ctx)
	}
	if err != nil {
		return fmt.Errorf("migration %s failed: %w", action, err)
	}
	if _, err := fmt.Fprintf(output, "migration %s complete\n", action); err != nil {
		return fmt.Errorf("write migration result: %w", err)
	}
	return nil
}

func migrationDSNFromEnv() string {
	if dsn := strings.TrimSpace(os.Getenv("MYSQL_DSN")); dsn != "" {
		return dsn
	}
	port := 3306
	if value := strings.TrimSpace(os.Getenv("DB_PORT")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 65535 {
			return ""
		}
		port = parsed
	}
	cfg := mysqldriver.NewConfig()
	cfg.Net = "tcp"
	host := strings.TrimSpace(os.Getenv("DB_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}
	cfg.Addr = net.JoinHostPort(host, strconv.Itoa(port))
	cfg.DBName = strings.TrimSpace(os.Getenv("DB_NAME"))
	if cfg.DBName == "" {
		cfg.DBName = "stock_quant_dev"
	}
	cfg.User = strings.TrimSpace(os.Getenv("DB_USER"))
	if cfg.User == "" {
		cfg.User = "stock_quant"
	}
	cfg.Passwd = os.Getenv("DB_PASSWORD")
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	return cfg.FormatDSN()
}
