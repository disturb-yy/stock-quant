// Package infrastructure 提供演示数据的 MySQL 连接、migration 和存储实现。
package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/disturb-yy/stock-quant/internal/demo"
	"github.com/disturb-yy/stock-quant/internal/market"
	"github.com/disturb-yy/stock-quant/internal/migration"
	stockdomain "github.com/disturb-yy/stock-quant/internal/stock/domain"
	"github.com/disturb-yy/stock-quant/pkg/config"
	"github.com/go-sql-driver/mysql"
)

const (
	migrationName            = "0001_demo_seed"
	marketOverviewMigration  = "0002_market_overview"
	marketSectorsMigration   = "0003_market_sectors"
	stockOverviewMigration   = "0004_stock_overview"
	stockBarsMigration       = "0005_stock_bars"
	stockFinancialsMigration = "0006_stock_financials"
)

const createSchemaMigrationsSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    name VARCHAR(191) NOT NULL PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

var demoSchemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS instruments (
        code VARCHAR(32) NOT NULL PRIMARY KEY,
        name VARCHAR(128) NOT NULL,
        exchange VARCHAR(16) NOT NULL,
        status VARCHAR(32) NOT NULL,
        as_of DATE NOT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
        INDEX idx_instruments_exchange (exchange),
        INDEX idx_instruments_status (status)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS daily_bars (
        id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
        instrument_code VARCHAR(32) NOT NULL,
        trade_date DATE NOT NULL,
        open_price DECIMAL(20,6) NOT NULL,
        high_price DECIMAL(20,6) NOT NULL,
        low_price DECIMAL(20,6) NOT NULL,
        close_price DECIMAL(20,6) NOT NULL,
		volume BIGINT UNSIGNED NOT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
        UNIQUE KEY uk_daily_bars_instrument_date (instrument_code, trade_date),
		CONSTRAINT fk_daily_bars_instrument FOREIGN KEY (instrument_code) REFERENCES instruments (code)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS financial_metrics (
        id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
        instrument_code VARCHAR(32) NOT NULL,
        metric_date DATE NOT NULL,
        metric_name VARCHAR(64) NOT NULL,
        metric_value DECIMAL(20,6) NOT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
        UNIQUE KEY uk_financial_metrics_instrument_date_name (instrument_code, metric_date, metric_name),
        CONSTRAINT fk_financial_metrics_instrument FOREIGN KEY (instrument_code) REFERENCES instruments (code)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS demo_seed_metadata (
        seed_name VARCHAR(64) NOT NULL PRIMARY KEY,
        seed_version VARCHAR(64) NOT NULL,
        as_of DATE NOT NULL,
        provider VARCHAR(64) NOT NULL,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
}

const marketOverviewSchemaSQL = `
ALTER TABLE daily_bars
    ADD COLUMN turnover_amount DECIMAL(20,6) NOT NULL DEFAULT 0 AFTER volume,
    ADD INDEX idx_daily_bars_trade_date (trade_date, instrument_code)`

const createIndexSnapshotsSQL = `
CREATE TABLE IF NOT EXISTS index_snapshots (
    code VARCHAR(32) NOT NULL,
    name VARCHAR(128) NOT NULL,
    trade_date DATE NOT NULL,
    observed_at DATETIME NOT NULL,
    close_price DECIMAL(20,6) NOT NULL,
    change_amount DECIMAL(20,6) NOT NULL,
    change_percent DECIMAL(20,6) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (code, trade_date),
    INDEX idx_index_snapshots_trade_date (trade_date, code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

const createMarketSectorsSQL = `
CREATE TABLE IF NOT EXISTS sector_categories (
    code VARCHAR(32) NOT NULL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

const createSectorMembershipsSQL = `
CREATE TABLE IF NOT EXISTS sector_memberships (
    sector_code VARCHAR(32) NOT NULL,
    instrument_code VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (sector_code, instrument_code),
    INDEX idx_sector_memberships_instrument (instrument_code, sector_code),
    CONSTRAINT fk_sector_memberships_sector FOREIGN KEY (sector_code) REFERENCES sector_categories (code),
    CONSTRAINT fk_sector_memberships_instrument FOREIGN KEY (instrument_code) REFERENCES instruments (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

const createStockOverviewSQL = `
CREATE TABLE IF NOT EXISTS daily_basic (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    instrument_code VARCHAR(32) NOT NULL,
    trade_date DATE NOT NULL,
    market_cap DECIMAL(20,6) NULL,
    pb DECIMAL(20,6) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_daily_basic_instrument_date (instrument_code, trade_date),
    INDEX idx_daily_basic_trade_date (trade_date, instrument_code),
    CONSTRAINT fk_daily_basic_instrument FOREIGN KEY (instrument_code) REFERENCES instruments (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

const createStockBarsSQL = `
CREATE TABLE IF NOT EXISTS daily_adjustment_factors (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    instrument_code VARCHAR(32) NOT NULL,
    trade_date DATE NOT NULL,
    qfq_factor DECIMAL(20,8) NOT NULL,
    hfq_factor DECIMAL(20,8) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_daily_adjustment_factors_instrument_date (instrument_code, trade_date),
    CONSTRAINT fk_daily_adjustment_factors_instrument FOREIGN KEY (instrument_code) REFERENCES instruments (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

const createStockFinancialsSQL = `
CREATE TABLE IF NOT EXISTS stock_financial_reports (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    instrument_code VARCHAR(32) NOT NULL,
    period VARCHAR(16) NOT NULL,
    period_end DATE NOT NULL,
    fiscal_year SMALLINT UNSIGNED NOT NULL,
    fiscal_quarter VARCHAR(2) NULL,
    published_at DATE NULL,
    reporting_currency CHAR(3) NOT NULL,
    amount_unit VARCHAR(16) NOT NULL,
    revenue DECIMAL(24,6) NULL,
    gross_profit DECIMAL(24,6) NULL,
    operating_profit DECIMAL(24,6) NULL,
    net_profit DECIMAL(24,6) NULL,
    cash_and_equivalents DECIMAL(24,6) NULL,
    accounts_receivable DECIMAL(24,6) NULL,
    inventory DECIMAL(24,6) NULL,
    current_assets DECIMAL(24,6) NULL,
    current_liabilities DECIMAL(24,6) NULL,
    total_assets DECIMAL(24,6) NULL,
    total_liabilities DECIMAL(24,6) NULL,
    total_equity DECIMAL(24,6) NULL,
    operating_cash_flow DECIMAL(24,6) NULL,
    capital_expenditure DECIMAL(24,6) NULL,
    investing_cash_flow DECIMAL(24,6) NULL,
    financing_cash_flow DECIMAL(24,6) NULL,
    net_cash_change DECIMAL(24,6) NULL,
    provider VARCHAR(64) NOT NULL,
    seed_version VARCHAR(64) NOT NULL,
    as_of DATE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_stock_financial_reports_identity (instrument_code, period, period_end),
    INDEX idx_stock_financial_reports_query (instrument_code, period, period_end),
    CONSTRAINT fk_stock_financial_reports_instrument FOREIGN KEY (instrument_code) REFERENCES instruments (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

// Open 连接并探测 MySQL；密码只进入 Driver DSN，不写入错误信息。
func Open(ctx context.Context, settings config.Database) (*sql.DB, error) {
	if ctx == nil {
		return nil, errors.New("database context is required")
	}
	driverConfig := mysql.Config{
		User:                 settings.User,
		Passwd:               settings.Password,
		Net:                  "tcp",
		Addr:                 net.JoinHostPort(settings.Host, settings.Port),
		DBName:               settings.Name,
		ParseTime:            true,
		Loc:                  time.UTC,
		Timeout:              5 * time.Second,
		ReadTimeout:          5 * time.Second,
		WriteTimeout:         5 * time.Second,
		AllowNativePasswords: true,
	}
	db, err := sql.Open("mysql", driverConfig.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open MySQL connection: %w", err)
	}
	pingContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingContext); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping MySQL: %w", err)
	}
	return db, nil
}

// Store 是 fixture 数据和状态元数据的 MySQL 存储实现。
type Store struct {
	db *sql.DB
}

// NewStore 创建 MySQL 存储。
func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("database connection is required")
	}
	return &Store{db: db}, nil
}

// Close 关闭底层数据库连接池。
func (store *Store) Close() error {
	return store.db.Close()
}

// Migrate 按版本执行演示数据 schema migration。
func (store *Store) Migrate(ctx context.Context) error {
	runner, err := migration.NewRunner(
		schemaMigration{db: store.db},
		marketOverviewSchemaMigration{db: store.db},
		marketSectorsSchemaMigration{db: store.db},
		stockOverviewSchemaMigration{db: store.db},
		stockBarsSchemaMigration{db: store.db},
		stockFinancialsSchemaMigration{db: store.db},
		stockValuationSchemaMigration{db: store.db},
	)
	if err != nil {
		return fmt.Errorf("create demo migration runner: %w", err)
	}
	if err := runner.Run(ctx); err != nil {
		return fmt.Errorf("run demo migrations: %w", err)
	}
	return nil
}

type schemaMigration struct {
	db *sql.DB
}

type marketOverviewSchemaMigration struct {
	db *sql.DB
}

type marketSectorsSchemaMigration struct {
	db *sql.DB
}

type stockOverviewSchemaMigration struct {
	db *sql.DB
}

type stockBarsSchemaMigration struct {
	db *sql.DB
}

type stockFinancialsSchemaMigration struct {
	db *sql.DB
}

func (migration marketOverviewSchemaMigration) Name() string {
	return marketOverviewMigration
}

func (migration marketOverviewSchemaMigration) Up(ctx context.Context) error {
	var applied int
	if err := migration.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, marketOverviewMigration).Scan(&applied); err != nil {
		return fmt.Errorf("check migration %q: %w", marketOverviewMigration, err)
	}
	if applied > 0 {
		return nil
	}
	if _, err := migration.db.ExecContext(ctx, marketOverviewSchemaSQL); err != nil {
		return fmt.Errorf("add market overview daily bar fields: %w", err)
	}
	if _, err := migration.db.ExecContext(ctx, createIndexSnapshotsSQL); err != nil {
		return fmt.Errorf("create index snapshots table: %w", err)
	}
	if _, err := migration.db.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES (?)`, marketOverviewMigration); err != nil {
		return fmt.Errorf("record migration %q: %w", marketOverviewMigration, err)
	}
	return nil
}

func (migration marketSectorsSchemaMigration) Name() string {
	return marketSectorsMigration
}

func (migration marketSectorsSchemaMigration) Up(ctx context.Context) error {
	var applied int
	if err := migration.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, marketSectorsMigration).Scan(&applied); err != nil {
		return fmt.Errorf("check migration %q: %w", marketSectorsMigration, err)
	}
	if applied > 0 {
		return nil
	}
	if _, err := migration.db.ExecContext(ctx, createMarketSectorsSQL); err != nil {
		return fmt.Errorf("create sector categories table: %w", err)
	}
	if _, err := migration.db.ExecContext(ctx, createSectorMembershipsSQL); err != nil {
		return fmt.Errorf("create sector memberships table: %w", err)
	}
	if _, err := migration.db.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES (?)`, marketSectorsMigration); err != nil {
		return fmt.Errorf("record migration %q: %w", marketSectorsMigration, err)
	}
	return nil
}

func (migration stockOverviewSchemaMigration) Name() string {
	return stockOverviewMigration
}

func (migration stockOverviewSchemaMigration) Up(ctx context.Context) error {
	var applied int
	if err := migration.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, stockOverviewMigration).Scan(&applied); err != nil {
		return fmt.Errorf("check migration %q: %w", stockOverviewMigration, err)
	}
	if applied > 0 {
		return nil
	}
	if _, err := migration.db.ExecContext(ctx, `ALTER TABLE financial_metrics ADD COLUMN basis VARCHAR(32) NOT NULL DEFAULT 'latest_report' AFTER metric_name`); err != nil {
		return fmt.Errorf("add financial metric basis: %w", err)
	}
	if _, err := migration.db.ExecContext(ctx, createStockOverviewSQL); err != nil {
		return fmt.Errorf("create daily basic table: %w", err)
	}
	if _, err := migration.db.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES (?)`, stockOverviewMigration); err != nil {
		return fmt.Errorf("record migration %q: %w", stockOverviewMigration, err)
	}
	return nil
}

func (migration stockBarsSchemaMigration) Name() string {
	return stockBarsMigration
}

func (migration stockBarsSchemaMigration) Up(ctx context.Context) error {
	var applied int
	if err := migration.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, stockBarsMigration).Scan(&applied); err != nil {
		return fmt.Errorf("check migration %q: %w", stockBarsMigration, err)
	}
	if applied > 0 {
		return nil
	}
	if _, err := migration.db.ExecContext(ctx, createStockBarsSQL); err != nil {
		return fmt.Errorf("create stock bars adjustment factors: %w", err)
	}
	if _, err := migration.db.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES (?)`, stockBarsMigration); err != nil {
		return fmt.Errorf("record migration %q: %w", stockBarsMigration, err)
	}
	return nil
}

func (migration stockFinancialsSchemaMigration) Name() string {
	return stockFinancialsMigration
}

func (migration stockFinancialsSchemaMigration) Up(ctx context.Context) error {
	var applied int
	if err := migration.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, stockFinancialsMigration).Scan(&applied); err != nil {
		return fmt.Errorf("check migration %q: %w", stockFinancialsMigration, err)
	}
	if applied > 0 {
		return nil
	}
	if _, err := migration.db.ExecContext(ctx, createStockFinancialsSQL); err != nil {
		return fmt.Errorf("create stock financial reports table: %w", err)
	}
	if _, err := migration.db.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES (?)`, stockFinancialsMigration); err != nil {
		return fmt.Errorf("record migration %q: %w", stockFinancialsMigration, err)
	}
	return nil
}

func (migration schemaMigration) Name() string {
	return migrationName
}

func (migration schemaMigration) Up(ctx context.Context) error {
	tx, err := migration.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin schema migration: %w", err)
	}
	rollback := true
	defer func() {
		if rollback {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, createSchemaMigrationsSQL); err != nil {
		return fmt.Errorf("create schema migrations table: %w", err)
	}
	var applied int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, migrationName).Scan(&applied); err != nil {
		return fmt.Errorf("check migration %q: %w", migrationName, err)
	}
	if applied > 0 {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit existing migration %q: %w", migrationName, err)
		}
		rollback = false
		return nil
	}
	for index, statement := range demoSchemaStatements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply demo schema statement %d: %w", index+1, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES (?)`, migrationName); err != nil {
		return fmt.Errorf("record migration %q: %w", migrationName, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %q: %w", migrationName, err)
	}
	rollback = false
	return nil
}

// SeedDemo 幂等写入 fixture 各类数据和版本元数据。
func (store *Store) SeedDemo(ctx context.Context, fixture demo.Fixture) error {
	if err := fixture.Validate(); err != nil {
		return fmt.Errorf("validate fixture in MySQL store: %w", err)
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin demo seed: %w", err)
	}
	rollback := true
	defer func() {
		if rollback {
			_ = tx.Rollback()
		}
	}()
	if err := seedInstruments(ctx, tx, fixture); err != nil {
		return err
	}
	if err := seedSectors(ctx, tx, fixture); err != nil {
		return err
	}
	if err := seedSectorMemberships(ctx, tx, fixture); err != nil {
		return err
	}
	if err := seedDailyBars(ctx, tx, fixture); err != nil {
		return err
	}
	if err := seedAdjustmentFactors(ctx, tx, fixture); err != nil {
		return err
	}
	if err := seedDailyBasics(ctx, tx, fixture); err != nil {
		return err
	}
	if err := seedIndexSnapshots(ctx, tx, fixture); err != nil {
		return err
	}
	if err := seedFinancialMetrics(ctx, tx, fixture); err != nil {
		return err
	}
	if err := seedFinancialReports(ctx, tx, fixture); err != nil {
		return err
	}
	if err := seedValuationSnapshots(ctx, tx, fixture); err != nil {
		return err
	}
	if err := seedMetadata(ctx, tx, fixture); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit demo seed: %w", err)
	}
	rollback = false
	return nil
}

func seedInstruments(ctx context.Context, tx *sql.Tx, fixture demo.Fixture) error {
	for _, instrument := range fixture.Instruments {
		_, err := tx.ExecContext(ctx, `
            INSERT INTO instruments (code, name, exchange, status, as_of)
            VALUES (?, ?, ?, ?, ?)
            ON DUPLICATE KEY UPDATE name = VALUES(name), exchange = VALUES(exchange), status = VALUES(status), as_of = VALUES(as_of)`,
			instrument.Code, instrument.Name, instrument.Exchange, instrument.Status, instrument.AsOf)
		if err != nil {
			return fmt.Errorf("upsert instrument %q: %w", instrument.Code, err)
		}
	}
	return nil
}

func seedSectors(ctx context.Context, tx *sql.Tx, fixture demo.Fixture) error {
	for _, sector := range fixture.Sectors {
		_, err := tx.ExecContext(ctx, `
            INSERT INTO sector_categories (code, name)
            VALUES (?, ?)
            ON DUPLICATE KEY UPDATE name = VALUES(name)`,
			sector.Code, sector.Name)
		if err != nil {
			return fmt.Errorf("upsert sector %q: %w", sector.Code, err)
		}
	}
	return nil
}

func seedSectorMemberships(ctx context.Context, tx *sql.Tx, fixture demo.Fixture) error {
	for _, membership := range fixture.SectorMemberships {
		_, err := tx.ExecContext(ctx, `
            INSERT INTO sector_memberships (sector_code, instrument_code)
            VALUES (?, ?)
            ON DUPLICATE KEY UPDATE sector_code = VALUES(sector_code)`,
			membership.SectorCode, membership.InstrumentCode)
		if err != nil {
			return fmt.Errorf("upsert sector membership %q/%q: %w", membership.SectorCode, membership.InstrumentCode, err)
		}
	}
	return nil
}

func seedDailyBars(ctx context.Context, tx *sql.Tx, fixture demo.Fixture) error {
	for _, bar := range fixture.DailyBars {
		_, err := tx.ExecContext(ctx, `
            INSERT INTO daily_bars (instrument_code, trade_date, open_price, high_price, low_price, close_price, volume, turnover_amount)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE open_price = VALUES(open_price), high_price = VALUES(high_price), low_price = VALUES(low_price), close_price = VALUES(close_price), volume = VALUES(volume), turnover_amount = VALUES(turnover_amount)`,
			bar.InstrumentCode, bar.TradeDate, bar.Open, bar.High, bar.Low, bar.Close, bar.Volume, bar.TurnoverAmount)
		if err != nil {
			return fmt.Errorf("upsert daily bar %q/%q: %w", bar.InstrumentCode, bar.TradeDate, err)
		}
	}
	return nil
}

func seedAdjustmentFactors(ctx context.Context, tx *sql.Tx, fixture demo.Fixture) error {
	for _, factor := range fixture.AdjustmentFactors {
		_, err := tx.ExecContext(ctx, `
            INSERT INTO daily_adjustment_factors (instrument_code, trade_date, qfq_factor, hfq_factor)
            VALUES (?, ?, ?, ?)
            ON DUPLICATE KEY UPDATE qfq_factor = VALUES(qfq_factor), hfq_factor = VALUES(hfq_factor)`,
			factor.InstrumentCode, factor.TradeDate, factor.QFQFactor, factor.HFQFactor)
		if err != nil {
			return fmt.Errorf("upsert adjustment factor %q/%q: %w", factor.InstrumentCode, factor.TradeDate, err)
		}
	}
	return nil
}

func seedDailyBasics(ctx context.Context, tx *sql.Tx, fixture demo.Fixture) error {
	for _, basic := range fixture.DailyBasics {
		_, err := tx.ExecContext(ctx, `
            INSERT INTO daily_basic (instrument_code, trade_date, market_cap, pb)
            VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''))
            ON DUPLICATE KEY UPDATE market_cap = VALUES(market_cap), pb = VALUES(pb)`,
			basic.InstrumentCode, basic.TradeDate, basic.MarketCap, basic.PB)
		if err != nil {
			return fmt.Errorf("upsert daily basic %q/%q: %w", basic.InstrumentCode, basic.TradeDate, err)
		}
	}
	return nil
}

func seedIndexSnapshots(ctx context.Context, tx *sql.Tx, fixture demo.Fixture) error {
	for _, index := range fixture.IndexSnapshots {
		observedAt, err := mysqlDateTime(index.ObservedAt)
		if err != nil {
			return fmt.Errorf("parse index snapshot observation %q/%q: %w", index.Code, index.TradeDate, err)
		}
		_, err = tx.ExecContext(ctx, `
            INSERT INTO index_snapshots (code, name, trade_date, observed_at, close_price, change_amount, change_percent)
            VALUES (?, ?, ?, ?, ?, ?, ?)
            ON DUPLICATE KEY UPDATE name = VALUES(name), observed_at = VALUES(observed_at), close_price = VALUES(close_price), change_amount = VALUES(change_amount), change_percent = VALUES(change_percent)`,
			index.Code, index.Name, index.TradeDate, observedAt, index.Close, index.Change, index.ChangePercent)
		if err != nil {
			return fmt.Errorf("upsert index snapshot %q/%q: %w", index.Code, index.TradeDate, err)
		}
	}
	return nil
}

func mysqlDateTime(value string) (string, error) {
	observedAt, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return "", fmt.Errorf("parse RFC3339 timestamp: %w", err)
	}
	return observedAt.UTC().Format("2006-01-02 15:04:05"), nil
}

func seedFinancialMetrics(ctx context.Context, tx *sql.Tx, fixture demo.Fixture) error {
	for _, metric := range fixture.FinancialMetrics {
		_, err := tx.ExecContext(ctx, `
            INSERT INTO financial_metrics (instrument_code, metric_date, metric_name, basis, metric_value)
            VALUES (?, ?, ?, ?, ?)
            ON DUPLICATE KEY UPDATE basis = VALUES(basis), metric_value = VALUES(metric_value)`,
			metric.InstrumentCode, metric.MetricDate, metric.MetricName, metric.Basis, metric.MetricValue)
		if err != nil {
			return fmt.Errorf("upsert financial metric %q/%q: %w", metric.InstrumentCode, metric.MetricName, err)
		}
	}
	return nil
}

func seedFinancialReports(ctx context.Context, tx *sql.Tx, fixture demo.Fixture) error {
	for _, report := range fixture.FinancialReports {
		_, err := tx.ExecContext(ctx, `
            INSERT INTO stock_financial_reports (
                instrument_code, period, period_end, fiscal_year, fiscal_quarter, published_at,
                reporting_currency, amount_unit, revenue, gross_profit, operating_profit, net_profit,
                cash_and_equivalents, accounts_receivable, inventory, current_assets, current_liabilities,
                total_assets, total_liabilities, total_equity, operating_cash_flow, capital_expenditure,
                investing_cash_flow, financing_cash_flow, net_cash_change, provider, seed_version, as_of
            ) VALUES (?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''),
                NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''),
                NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?)
            ON DUPLICATE KEY UPDATE
                fiscal_year = VALUES(fiscal_year), fiscal_quarter = VALUES(fiscal_quarter), published_at = VALUES(published_at),
                reporting_currency = VALUES(reporting_currency), amount_unit = VALUES(amount_unit), revenue = VALUES(revenue),
                gross_profit = VALUES(gross_profit), operating_profit = VALUES(operating_profit), net_profit = VALUES(net_profit),
                cash_and_equivalents = VALUES(cash_and_equivalents), accounts_receivable = VALUES(accounts_receivable), inventory = VALUES(inventory),
                current_assets = VALUES(current_assets), current_liabilities = VALUES(current_liabilities), total_assets = VALUES(total_assets),
                total_liabilities = VALUES(total_liabilities), total_equity = VALUES(total_equity), operating_cash_flow = VALUES(operating_cash_flow),
                capital_expenditure = VALUES(capital_expenditure), investing_cash_flow = VALUES(investing_cash_flow), financing_cash_flow = VALUES(financing_cash_flow),
                net_cash_change = VALUES(net_cash_change), provider = VALUES(provider), seed_version = VALUES(seed_version), as_of = VALUES(as_of)`,
			report.InstrumentCode, report.Period, report.PeriodEnd, report.FiscalYear, report.FiscalQuarter, financialReportPublishedAt(report),
			"CNY", "CNY", financialReportValue(report.Income.Revenue), financialReportValue(report.Income.GrossProfit), financialReportValue(report.Income.OperatingProfit), financialReportValue(report.Income.NetProfit),
			financialReportValue(report.Balance.CashAndEquivalents), financialReportValue(report.Balance.AccountsReceivable), financialReportValue(report.Balance.Inventory), financialReportValue(report.Balance.CurrentAssets), financialReportValue(report.Balance.CurrentLiabilities),
			financialReportValue(report.Balance.TotalAssets), financialReportValue(report.Balance.TotalLiabilities), financialReportValue(report.Balance.TotalEquity), financialReportValue(report.CashFlow.OperatingCashFlow), financialReportValue(report.CashFlow.CapitalExpenditure),
			financialReportValue(report.CashFlow.InvestingCashFlow), financialReportValue(report.CashFlow.FinancingCashFlow), financialReportValue(report.CashFlow.NetCashChange), market.DemoProviderName, fixture.Version, fixture.AsOf)
		if err != nil {
			return fmt.Errorf("upsert financial report %q/%q/%q: %w", report.InstrumentCode, report.Period, report.PeriodEnd, err)
		}
	}
	return nil
}

func financialReportValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func financialReportPublishedAt(report stockdomain.FinancialReport) string {
	if report.PublishedAt == nil {
		return ""
	}
	return *report.PublishedAt
}

func seedMetadata(ctx context.Context, tx *sql.Tx, fixture demo.Fixture) error {
	if _, err := tx.ExecContext(ctx, `
        INSERT INTO demo_seed_metadata (seed_name, seed_version, as_of, provider)
        VALUES (?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE seed_version = VALUES(seed_version), as_of = VALUES(as_of), provider = VALUES(provider)`,
		demo.SeedName, fixture.Version, fixture.AsOf, market.DemoProviderName); err != nil {
		return fmt.Errorf("upsert demo seed metadata: %w", err)
	}
	return nil
}

// ReadDemoSnapshot 读取开发状态接口所需的数据库快照。
func (store *Store) ReadDemoSnapshot(ctx context.Context) (demo.StoreSnapshot, error) {
	snapshot := demo.StoreSnapshot{Samples: make([]demo.SampleStock, 0, 5)}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM instruments`).Scan(&snapshot.Counts.Instruments); err != nil {
		return demo.StoreSnapshot{}, fmt.Errorf("count instruments: %w", err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM daily_bars`).Scan(&snapshot.Counts.DailyBars); err != nil {
		return demo.StoreSnapshot{}, fmt.Errorf("count daily bars: %w", err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM daily_basic`).Scan(&snapshot.Counts.DailyBasics); err != nil {
		return demo.StoreSnapshot{}, fmt.Errorf("count daily basic: %w", err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM financial_metrics`).Scan(&snapshot.Counts.FinancialMetrics); err != nil {
		return demo.StoreSnapshot{}, fmt.Errorf("count financial metrics: %w", err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM stock_financial_reports`).Scan(&snapshot.Counts.FinancialReports); err != nil {
		return demo.StoreSnapshot{}, fmt.Errorf("count financial reports: %w", err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM stock_valuation_snapshots`).Scan(&snapshot.Counts.ValuationSnapshots); err != nil {
		return demo.StoreSnapshot{}, fmt.Errorf("count valuation snapshots: %w", err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM index_snapshots`).Scan(&snapshot.Counts.IndexSnapshots); err != nil {
		return demo.StoreSnapshot{}, fmt.Errorf("count index snapshots: %w", err)
	}
	var asOf string
	err := store.db.QueryRowContext(ctx, `SELECT seed_version, DATE_FORMAT(as_of, '%Y-%m-%d') FROM demo_seed_metadata WHERE seed_name = ?`, demo.SeedName).Scan(&snapshot.SeedVersion, &asOf)
	if errors.Is(err, sql.ErrNoRows) {
		return readSamples(ctx, store.db, snapshot)
	}
	if err != nil {
		return demo.StoreSnapshot{}, fmt.Errorf("read demo seed metadata: %w", err)
	}
	snapshot.AsOf = &asOf
	return readSamples(ctx, store.db, snapshot)
}

func readSamples(ctx context.Context, db *sql.DB, snapshot demo.StoreSnapshot) (demo.StoreSnapshot, error) {
	rows, err := db.QueryContext(ctx, `SELECT code, name, exchange, status FROM instruments ORDER BY code LIMIT 5`)
	if err != nil {
		return demo.StoreSnapshot{}, fmt.Errorf("read sample instruments: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sample demo.SampleStock
		if err := rows.Scan(&sample.Code, &sample.Name, &sample.Exchange, &sample.Status); err != nil {
			return demo.StoreSnapshot{}, fmt.Errorf("scan sample instrument: %w", err)
		}
		snapshot.Samples = append(snapshot.Samples, sample)
	}
	if err := rows.Err(); err != nil {
		return demo.StoreSnapshot{}, fmt.Errorf("iterate sample instruments: %w", err)
	}
	return snapshot, nil
}
