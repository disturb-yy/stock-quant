// Package infrastructure 提供股票池的 MySQL 持久化实现。
package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/disturb-yy/stock-quant/internal/pool/domain"
	"github.com/go-sql-driver/mysql"
)

const (
	stockPoolMigration       = "0009_stock_pool"
	stockPoolMemberMigration = "0010_stock_pool_member"
	demoSeedKey              = "fnd-003-demo-v8-stock-pool"
	demoPoolName             = "演示手工股票池"
	demoPoolDescription      = "用于验证股票池创建、搜索、详情和成员管理的确定性手工股票池。"
	demoMemberSymbol         = "000001.SZ"
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

const createStockPoolMemberSQL = `
CREATE TABLE IF NOT EXISTS t_stock_pool_member (
    pool_id BIGINT UNSIGNED NOT NULL,
    symbol VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (pool_id, symbol),
    INDEX idx_t_stock_pool_member_symbol (symbol),
    CONSTRAINT fk_t_stock_pool_member_pool FOREIGN KEY (pool_id) REFERENCES t_stock_pool (id) ON DELETE CASCADE,
    CONSTRAINT fk_t_stock_pool_member_instrument FOREIGN KEY (symbol) REFERENCES instruments (code)
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

// Migrate 幂等创建股票池及成员表并记录升级版本。
func (store *MySQLStockPoolStore) Migrate(ctx context.Context) error {
	if err := ensureMigrationTable(ctx, store.db); err != nil {
		return err
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin stock pool migration: %w", err)
	}
	defer rollbackTransaction(tx)
	if err := ensureMigration(ctx, tx, stockPoolMigration, createStockPoolSQL, "create t_stock_pool"); err != nil {
		return err
	}
	if err := ensureMigration(ctx, tx, stockPoolMemberMigration, createStockPoolMemberSQL, "create t_stock_pool_member"); err != nil {
		return err
	}
	return commitMigration(tx, stockPoolMemberMigration)
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

// ListMembers 按原始 symbol 升序稳定分页读取股票池成员。
func (store *MySQLStockPoolStore) ListMembers(ctx context.Context, poolID int64, page, pageSize int) ([]domain.StockPoolMember, int64, error) {
	if err := ensurePoolExists(ctx, store.db, poolID); err != nil {
		return nil, 0, err
	}
	var total int64
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM t_stock_pool_member WHERE pool_id = ?`, poolID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count stock pool members: %w", err)
	}
	rows, err := store.db.QueryContext(ctx, `
        SELECT member.symbol, instruments.name
        FROM t_stock_pool_member AS member
        INNER JOIN instruments ON instruments.code = member.symbol
        WHERE member.pool_id = ?
        ORDER BY member.symbol ASC
        LIMIT ? OFFSET ?`, poolID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list stock pool members: %w", err)
	}
	defer rows.Close()
	items := make([]domain.StockPoolMember, 0, pageSize)
	for rows.Next() {
		var member domain.StockPoolMember
		if err := rows.Scan(&member.Symbol, &member.Name); err != nil {
			return nil, 0, fmt.Errorf("scan stock pool member: %w", err)
		}
		items = append(items, member)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate stock pool members: %w", err)
	}
	return items, total, nil
}

// AddMember 在事务内添加成员并返回更新后的成员数。
func (store *MySQLStockPoolStore) AddMember(ctx context.Context, poolID int64, symbol string) (domain.StockPoolMember, int64, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.StockPoolMember{}, 0, fmt.Errorf("begin add stock pool member: %w", err)
	}
	defer rollbackTransaction(tx)
	if err := ensurePoolExists(ctx, tx, poolID); err != nil {
		return domain.StockPoolMember{}, 0, err
	}
	var existing string
	err = tx.QueryRowContext(ctx, `SELECT symbol FROM t_stock_pool_member WHERE pool_id = ? AND symbol = ? FOR UPDATE`, poolID, symbol).Scan(&existing)
	if err == nil {
		return domain.StockPoolMember{}, 0, domain.ErrStockPoolMemberConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.StockPoolMember{}, 0, fmt.Errorf("check stock pool member: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO t_stock_pool_member (pool_id, symbol) VALUES (?, ?)`, poolID, symbol); err != nil {
		if isDuplicateKeyError(err) {
			return domain.StockPoolMember{}, 0, domain.ErrStockPoolMemberConflict
		}
		return domain.StockPoolMember{}, 0, fmt.Errorf("insert stock pool member: %w", err)
	}
	member, err := readStockPoolMember(ctx, tx, poolID, symbol)
	if err != nil {
		return domain.StockPoolMember{}, 0, err
	}
	count, err := countStockPoolMembers(ctx, tx, poolID)
	if err != nil {
		return domain.StockPoolMember{}, 0, err
	}
	if err := tx.Commit(); err != nil {
		return domain.StockPoolMember{}, 0, fmt.Errorf("commit stock pool member: %w", err)
	}
	return member, count, nil
}

// DeleteMember 在事务内删除成员并返回更新后的成员数。
func (store *MySQLStockPoolStore) DeleteMember(ctx context.Context, poolID int64, symbol string) (int64, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin delete stock pool member: %w", err)
	}
	defer rollbackTransaction(tx)
	if err := ensurePoolExists(ctx, tx, poolID); err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM t_stock_pool_member WHERE pool_id = ? AND symbol = ?`, poolID, symbol)
	if err != nil {
		return 0, fmt.Errorf("delete stock pool member: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read deleted stock pool member count: %w", err)
	}
	if affected == 0 {
		return 0, domain.ErrStockPoolMemberNotFound
	}
	count, err := countStockPoolMembers(ctx, tx, poolID)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit deleted stock pool member: %w", err)
	}
	return count, nil
}

// SeedDemo 幂等写入一个包含真实股票成员的手工股票池。
func (store *MySQLStockPoolStore) SeedDemo(ctx context.Context) error {
	if _, err := store.db.ExecContext(ctx, `
        INSERT INTO t_stock_pool (name, description, source, seed_key)
        VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE id = id`, demoPoolName, demoPoolDescription, domain.SourceManual, demoSeedKey); err != nil {
		return fmt.Errorf("seed demo stock pool: %w", err)
	}
	var poolID int64
	if err := store.db.QueryRowContext(ctx, `SELECT id FROM t_stock_pool WHERE seed_key = ?`, demoSeedKey).Scan(&poolID); err != nil {
		return fmt.Errorf("read demo stock pool id: %w", err)
	}
	var instrumentExists bool
	if err := store.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM instruments WHERE code = ?)`, demoMemberSymbol).Scan(&instrumentExists); err != nil {
		return fmt.Errorf("check demo stock identity: %w", err)
	}
	if !instrumentExists {
		return fmt.Errorf("seed demo stock pool member %q: %w", demoMemberSymbol, domain.ErrStockPoolInstrumentNotFound)
	}
	if _, err := store.db.ExecContext(ctx, `
        INSERT INTO t_stock_pool_member (pool_id, symbol)
        VALUES (?, ?)
        ON DUPLICATE KEY UPDATE symbol = VALUES(symbol)`, poolID, demoMemberSymbol); err != nil {
		return fmt.Errorf("seed demo stock pool member: %w", err)
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

func ensureMigration(ctx context.Context, tx *sql.Tx, name, statement, description string) error {
	applied, err := migrationApplied(ctx, tx, name)
	if err != nil {
		return err
	}
	if applied {
		return nil
	}
	if _, err := tx.ExecContext(ctx, statement); err != nil {
		return fmt.Errorf("%s: %w", description, err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES (?)`, name); err != nil {
		return fmt.Errorf("record migration %q: %w", name, err)
	}
	return nil
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

const stockPoolSelect = `
SELECT id, name, description, source, created_at, updated_at,
       (SELECT COUNT(*) FROM t_stock_pool_member WHERE pool_id = t_stock_pool.id) AS member_count
FROM t_stock_pool`

func readStockPool(ctx context.Context, queryer queryRowContext, id int64) (domain.StockPool, error) {
	return scanStockPool(queryer.QueryRowContext(ctx, stockPoolSelect+` WHERE id = ?`, id))
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

type queryRowContext interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func ensurePoolExists(ctx context.Context, queryer queryRowContext, poolID int64) error {
	var existingID int64
	if err := queryer.QueryRowContext(ctx, `SELECT id FROM t_stock_pool WHERE id = ?`, poolID).Scan(&existingID); errors.Is(err, sql.ErrNoRows) {
		return domain.ErrStockPoolNotFound
	} else if err != nil {
		return fmt.Errorf("check stock pool: %w", err)
	}
	return nil
}

func countStockPoolMembers(ctx context.Context, queryer queryRowContext, poolID int64) (int64, error) {
	var count int64
	if err := queryer.QueryRowContext(ctx, `SELECT COUNT(*) FROM t_stock_pool_member WHERE pool_id = ?`, poolID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count stock pool members: %w", err)
	}
	return count, nil
}

func readStockPoolMember(ctx context.Context, queryer queryRowContext, poolID int64, symbol string) (domain.StockPoolMember, error) {
	var member domain.StockPoolMember
	if err := queryer.QueryRowContext(ctx, `
        SELECT member.symbol, instruments.name
        FROM t_stock_pool_member AS member
        INNER JOIN instruments ON instruments.code = member.symbol
        WHERE member.pool_id = ? AND member.symbol = ?`, poolID, symbol).Scan(&member.Symbol, &member.Name); err != nil {
		return domain.StockPoolMember{}, fmt.Errorf("read stock pool member: %w", err)
	}
	return member, nil
}

func isDuplicateKeyError(err error) bool {
	var mysqlError *mysql.MySQLError
	return errors.As(err, &mysqlError) && mysqlError.Number == 1062
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
