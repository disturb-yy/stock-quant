package infrastructure

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/disturb-yy/stock-quant/internal/demo"
)

const stockValuationMigration = "0007_stock_valuation"

const createStockValuationSQL = `
CREATE TABLE IF NOT EXISTS stock_valuation_snapshots (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    instrument_code VARCHAR(32) NOT NULL,
    trade_date DATE NOT NULL,
    pe_ttm DECIMAL(20,6) NULL,
    pe_ttm_basis VARCHAR(32) NULL,
    pb DECIMAL(20,6) NULL,
    pb_basis VARCHAR(32) NULL,
    ps_ttm DECIMAL(20,6) NULL,
    ps_ttm_basis VARCHAR(32) NULL,
    provider VARCHAR(64) NOT NULL,
    seed_version VARCHAR(64) NOT NULL,
    as_of DATE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_stock_valuation_instrument_date (instrument_code, trade_date),
    INDEX idx_stock_valuation_query (instrument_code, trade_date),
    INDEX idx_stock_valuation_trade_date (trade_date, instrument_code),
    CONSTRAINT fk_stock_valuation_instrument FOREIGN KEY (instrument_code) REFERENCES instruments (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

type stockValuationSchemaMigration struct {
	db *sql.DB
}

func (migration stockValuationSchemaMigration) Name() string {
	return stockValuationMigration
}

func (migration stockValuationSchemaMigration) Up(ctx context.Context) error {
	var applied int
	if err := migration.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, stockValuationMigration).Scan(&applied); err != nil {
		return fmt.Errorf("check migration %q: %w", stockValuationMigration, err)
	}
	if applied > 0 {
		return nil
	}
	if _, err := migration.db.ExecContext(ctx, createStockValuationSQL); err != nil {
		return fmt.Errorf("create stock valuation snapshots table: %w", err)
	}
	if _, err := migration.db.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES (?)`, stockValuationMigration); err != nil {
		return fmt.Errorf("record migration %q: %w", stockValuationMigration, err)
	}
	return nil
}

func seedValuationSnapshots(ctx context.Context, tx *sql.Tx, fixture demo.Fixture) error {
	for _, snapshot := range fixture.ValuationSnapshots {
		_, err := tx.ExecContext(ctx, `
            INSERT INTO stock_valuation_snapshots (
                instrument_code, trade_date, pe_ttm, pe_ttm_basis, pb, pb_basis, ps_ttm, ps_ttm_basis,
                provider, seed_version, as_of
            ) VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?)
            ON DUPLICATE KEY UPDATE
                pe_ttm = VALUES(pe_ttm), pe_ttm_basis = VALUES(pe_ttm_basis),
                pb = VALUES(pb), pb_basis = VALUES(pb_basis),
                ps_ttm = VALUES(ps_ttm), ps_ttm_basis = VALUES(ps_ttm_basis),
                provider = VALUES(provider), seed_version = VALUES(seed_version), as_of = VALUES(as_of)`,
			snapshot.InstrumentCode, snapshot.AsOf,
			valuationValue(snapshot.PETTM.Value), snapshot.PETTM.Basis,
			valuationValue(snapshot.PB.Value), snapshot.PB.Basis,
			valuationValue(snapshot.PSTTM.Value), snapshot.PSTTM.Basis,
			"mysql-demo-fixture", fixture.Version, fixture.AsOf)
		if err != nil {
			return fmt.Errorf("upsert valuation snapshot %q/%q: %w", snapshot.InstrumentCode, snapshot.AsOf, err)
		}
	}
	return nil
}

func valuationValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
