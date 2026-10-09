package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
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
	"stock-quant/internal/infrastructure/tushare"
	"stock-quant/internal/shared/types"

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
	if len(args) > 0 && args[0] == "sync" {
		if err := runSyncCommand(context.Background(), args[1:], os.Stdout); err != nil {
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

type initialMarketSyncRunner interface {
	Sync(context.Context, types.TradingDate, types.TradingDate) (app.InitialMarketSyncResult, error)
}

func runSyncCommand(ctx context.Context, args []string, output io.Writer) error {
	if len(args) == 0 || args[0] != "initial" {
		return fmt.Errorf("usage: stockquant sync initial --from-date YYYY-MM-DD --through-date YYYY-MM-DD")
	}
	return runInitialMarketSyncFromEnv(ctx, args[1:], output)
}

func runInitialMarketSyncCommand(ctx context.Context, args []string, output io.Writer, runner initialMarketSyncRunner) error {
	from, through, err := parseInitialMarketSyncArgs(args)
	if err != nil {
		return err
	}
	if runner == nil {
		return fmt.Errorf("initial market sync runner is required")
	}
	result, err := runner.Sync(ctx, from, through)
	if err != nil {
		return fmt.Errorf("initial market sync failed: %w", err)
	}
	if _, err := fmt.Fprintf(output, "initial market sync complete: stocks=%d calendars=%d\n", result.StockCount, result.CalendarCount); err != nil {
		return fmt.Errorf("write initial sync result: %w", err)
	}
	return nil
}

func parseInitialMarketSyncArgs(args []string) (types.TradingDate, types.TradingDate, error) {
	flags := flag.NewFlagSet("sync initial", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	fromText := flags.String("from-date", "", "inclusive calendar start date (YYYY-MM-DD)")
	throughText := flags.String("through-date", "", "inclusive calendar end date (YYYY-MM-DD)")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return types.TradingDate{}, types.TradingDate{}, fmt.Errorf("usage: stockquant sync initial --from-date YYYY-MM-DD --through-date YYYY-MM-DD")
	}
	if strings.TrimSpace(*fromText) == "" || strings.TrimSpace(*throughText) == "" {
		return types.TradingDate{}, types.TradingDate{}, fmt.Errorf("usage: stockquant sync initial --from-date YYYY-MM-DD --through-date YYYY-MM-DD")
	}
	from, err := types.ParseTradingDate(*fromText)
	if err != nil {
		return types.TradingDate{}, types.TradingDate{}, fmt.Errorf("invalid --from-date: %w", err)
	}
	through, err := types.ParseTradingDate(*throughText)
	if err != nil {
		return types.TradingDate{}, types.TradingDate{}, fmt.Errorf("invalid --through-date: %w", err)
	}
	if from.String() > through.String() {
		return types.TradingDate{}, types.TradingDate{}, fmt.Errorf("--from-date must be on or before --through-date")
	}
	return from, through, nil
}

func runInitialMarketSyncFromEnv(ctx context.Context, args []string, output io.Writer) error {
	_, _, err := parseInitialMarketSyncArgs(args)
	if err != nil {
		return err
	}
	if strings.TrimSpace(os.Getenv("DATA_PROVIDER")) != "tushare" {
		return fmt.Errorf("initial market sync requires DATA_PROVIDER=tushare")
	}
	token := strings.TrimSpace(os.Getenv("TUSHARE_TOKEN"))
	if token == "" {
		return fmt.Errorf("initial market sync requires TUSHARE_TOKEN")
	}
	dsn := migrationDSNFromEnv()
	if strings.TrimSpace(dsn) == "" {
		return fmt.Errorf("database connection is not configured; set MYSQL_DSN or DB_* variables")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open database connection: %w", err)
	}
	defer db.Close()
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	stockRepository, err := mysqlinfra.NewStockRepository(db)
	if err != nil {
		return err
	}
	calendarRepository, err := mysqlinfra.NewTradeCalendarRepository(db)
	if err != nil {
		return err
	}
	client, err := tushare.NewClient(tushare.ClientConfig{Token: token})
	if err != nil {
		return fmt.Errorf("configure Tushare client: %w", err)
	}
	provider := tushare.NewInitialMarketDataProvider(client)
	service := app.NewInitialMarketSync(provider, stockRepository, calendarRepository)
	return runInitialMarketSyncCommand(ctx, args, output, service)
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
