// Package infrastructure 提供股票池的 MySQL 持久化实现。
package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/disturb-yy/stock-quant/internal/pool/domain"
)

const (
	stockPoolMigration  = "0009_stock_pool"
	demoSeedKey         = "fnd-003-demo-v8-stock-pool"
	demoPoolName        = "演示手工股票池"
	demoPoolDescription = "用于验证股票池创建、搜索和详情的确定性手工股票池。"
)

const createStockPoolSQL = `
CREATE TABLE IF NOT EXISTS t_stock_pool (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500) NULL,
    source VARCHAR(16) NOT NULL,
    seed_key VARCHAR(100) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_t_stock_pool_seed_key (seed_key),
    INDEX idx_t_stock_pool_updated_at (updated_at, id),
    INDEX idx_t_stock_pool_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

// MySQLStockPoolStore 实现股票池持久化及迁移入口。
type MySQLStockPoolStore struct {
	db *sql.DB
}

// NewMySQLStockPoolStore 创建股票池 MySQL 存储。
func NewMySQLStockPoolStore(db *sql.DB) (*MySQLStockPoolStore, error) {
	if db == nil {
		return nil, errors.New("stock pool database connection is required")
	}
	return &MySQLStockPoolStore{db: db}, nil
}

// Migrate 幂等创建股票池表并记录升级版本。
func (store *MySQLStockPoolStore) Migrate(ctx context.Context) error {
	if err := ensureMigrationTable(ctx, store.db); err != nil {
		return err
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin stock pool migration: %w", err)
	}
	defer rollbackTransaction(tx)
	applied, err := migrationApplied(ctx, tx, stockPoolMigration)
	if err != nil {
		return err
	}
	if applied {
		return commitMigration(tx, stockPoolMigration)
	}
	if _, err := tx.ExecContext(ctx, createStockPoolSQL); err != nil {
		return fmt.Errorf("create t_stock_pool: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES (?)`, stockPoolMigration); err != nil {
		return fmt.Errorf("record migration %q: %w", stockPoolMigration, err)
	}
	return commitMigration(tx, stockPoolMigration)
}

// Create 持久化服务端固定为 manual 的股票池。
func (store *MySQLStockPoolStore) Create(ctx context.Context, pool domain.StockPool) (domain.StockPool, error) {
	if pool.Source != domain.SourceManual {
		return domain.StockPool{}, errors.New("stock pool source must be manual")
	}
	result, err := store.db.ExecContext(ctx, `
        INSERT INTO t_stock_pool (name, description, source)
        VALUES (?, ?, ?)`, pool.Name, nullableDescription(pool.Description), pool.Source)
	if err != nil {
		return domain.StockPool{}, fmt.Errorf("insert stock pool: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.StockPool{}, fmt.Errorf("read stock pool id: %w", err)
	}
	return store.Get(ctx, id)
}

// List 按名称字面包含搜索，并按更新时间、ID 倒序稳定分页。
func (store *MySQLStockPoolStore) List(ctx context.Context, search string, page, pageSize int) ([]domain.StockPool, int64, error) {
	filter, arguments := stockPoolFilter(search)
	total, err := countStockPools(ctx, store.db, filter, arguments)
	if err != nil {
		return nil, 0, err
	}
	return listStockPools(ctx, store.db, filter, arguments, page, pageSize, total)
}

// Get 读取一个股票池的真实概览数据。
func (store *MySQLStockPoolStore) Get(ctx context.Context, id int64) (domain.StockPool, error) {
	pool, err := readStockPool(ctx, store.db, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.StockPool{}, domain.ErrStockPoolNotFound
	}
	if err != nil {
		return domain.StockPool{}, fmt.Errorf("read stock pool: %w", err)
	}
	return pool, nil
}

// SeedDemo 幂等写入一个可搜索、可追溯的手工股票池。
func (store *MySQLStockPoolStore) SeedDemo(ctx context.Context) error {
	_, err := store.db.ExecContext(ctx, `
        INSERT INTO t_stock_pool (name, description, source, seed_key)
        VALUES (?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE id = id`, demoPoolName, demoPoolDescription, domain.SourceManual, demoSeedKey)
	if err != nil {
		return fmt.Errorf("seed demo stock pool: %w", err)
	}
	return nil
}

func ensureMigrationTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS schema_migrations (
            name VARCHAR(128) NOT NULL PRIMARY KEY,
            applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		return fmt.Errorf("create schema migrations table for stock pools: %w", err)
	}
	return nil
}

func migrationApplied(ctx context.Context, tx *sql.Tx, name string) (bool, error) {
	var applied int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, name).Scan(&applied); err != nil {
		return false, fmt.Errorf("check migration %q: %w", name, err)
	}
	return applied > 0, nil
}

func commitMigration(tx *sql.Tx, name string) error {
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %q: %w", name, err)
	}
	return nil
}

func rollbackTransaction(tx *sql.Tx) {
	_ = tx.Rollback()
}

func stockPoolFilter(search string) (string, []any) {
	if search == "" {
		return "", nil
	}
	return " WHERE name LIKE ? ESCAPE '!'", []any{"%" + escapeLike(search) + "%"}
}

func escapeLike(search string) string {
	replacer := strings.NewReplacer("!", "!!", "%", "!%", "_", "!_")
	return replacer.Replace(search)
}

func countStockPools(ctx context.Context, db *sql.DB, filter string, arguments []any) (int64, error) {
	var total int64
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM t_stock_pool`+filter, arguments...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count stock pools: %w", err)
	}
	return total, nil
}

func listStockPools(ctx context.Context, db *sql.DB, filter string, arguments []any, page, pageSize int, total int64) ([]domain.StockPool, int64, error) {
	offset := (page - 1) * pageSize
	arguments = append(arguments, pageSize, offset)
	rows, err := db.QueryContext(ctx, stockPoolSelect+filter+` ORDER BY updated_at DESC, id DESC LIMIT ? OFFSET ?`, arguments...)
	if err != nil {
		return nil, 0, fmt.Errorf("list stock pools: %w", err)
	}
	defer rows.Close()
	items := make([]domain.StockPool, 0, pageSize)
	for rows.Next() {
		pool, err := scanStockPool(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, pool)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate stock pools: %w", err)
	}
	return items, total, nil
}

const stockPoolSelect = `SELECT id, name, description, source, created_at, updated_at, 0 AS member_count FROM t_stock_pool`

func readStockPool(ctx context.Context, db *sql.DB, id int64) (domain.StockPool, error) {
	return scanStockPool(db.QueryRowContext(ctx, stockPoolSelect+` WHERE id = ?`, id))
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanStockPool(scanner rowScanner) (domain.StockPool, error) {
	var pool domain.StockPool
	var description sql.NullString
	if err := scanner.Scan(&pool.ID, &pool.Name, &description, &pool.Source, &pool.CreatedAt, &pool.UpdatedAt, &pool.MemberCount); err != nil {
		return domain.StockPool{}, err
	}
	if description.Valid {
		pool.Description = &description.String
	}
	return pool, nil
}

func nullableDescription(description *string) any {
	if description == nil {
		return nil
	}
	return *description
}

var _ interface {
	Migrate(context.Context) error
} = (*MySQLStockPoolStore)(nil)
