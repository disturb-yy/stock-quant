package infrastructure

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/disturb-yy/stock-quant/internal/screener/domain"
)

const savedScreenerMigration = "0008_screener_saved"

const createSavedScreenerSQL = `
CREATE TABLE IF NOT EXISTS t_screener (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500) NULL,
    current_version BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_t_screener_updated_at (updated_at, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

const createSavedScreenerVersionSQL = `
CREATE TABLE IF NOT EXISTS t_screener_version (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    screener_id BIGINT UNSIGNED NOT NULL,
    version BIGINT UNSIGNED NOT NULL,
    spec_json JSON NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_t_screener_version (screener_id, version),
    CONSTRAINT fk_t_screener_version_screener FOREIGN KEY (screener_id) REFERENCES t_screener (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

// MySQLSavedScreenerStore 实现方案身份和不可变版本的事务读写。
type MySQLSavedScreenerStore struct {
	db *sql.DB
}

func NewMySQLSavedScreenerStore(db *sql.DB) (*MySQLSavedScreenerStore, error) {
	if db == nil {
		return nil, errors.New("saved screener database connection is required")
	}
	return &MySQLSavedScreenerStore{db: db}, nil
}

// Migrate 幂等创建方案当前表和不可变版本表。
func (store *MySQLSavedScreenerStore) Migrate(ctx context.Context) error {
	if _, err := store.db.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS schema_migrations (
            name VARCHAR(128) NOT NULL PRIMARY KEY,
            applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
		return fmt.Errorf("create schema migrations table for saved screeners: %w", err)
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin saved screener migration: %w", err)
	}
	rollback := true
	defer func() {
		if rollback {
			_ = tx.Rollback()
		}
	}()
	var applied int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, savedScreenerMigration).Scan(&applied); err != nil {
		return fmt.Errorf("check migration %q: %w", savedScreenerMigration, err)
	}
	if applied > 0 {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit existing migration %q: %w", savedScreenerMigration, err)
		}
		rollback = false
		return nil
	}
	if _, err := tx.ExecContext(ctx, createSavedScreenerSQL); err != nil {
		return fmt.Errorf("create t_screener: %w", err)
	}
	if _, err := tx.ExecContext(ctx, createSavedScreenerVersionSQL); err != nil {
		return fmt.Errorf("create t_screener_version: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES (?)`, savedScreenerMigration); err != nil {
		return fmt.Errorf("record migration %q: %w", savedScreenerMigration, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %q: %w", savedScreenerMigration, err)
	}
	rollback = false
	return nil
}

func (store *MySQLSavedScreenerStore) Create(ctx context.Context, input domain.SavedScreenerInput) (domain.SavedScreener, error) {
	specJSON, err := marshalSavedSpec(input.Spec)
	if err != nil {
		return domain.SavedScreener{}, err
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.SavedScreener{}, fmt.Errorf("begin saved screener create: %w", err)
	}
	rollback := true
	defer func() {
		if rollback {
			_ = tx.Rollback()
		}
	}()
	result, err := tx.ExecContext(ctx, `
        INSERT INTO t_screener (name, description, current_version)
        VALUES (?, ?, 1)`, input.Name, nullableDescription(input.Description))
	if err != nil {
		return domain.SavedScreener{}, fmt.Errorf("insert saved screener: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.SavedScreener{}, fmt.Errorf("read saved screener id: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
        INSERT INTO t_screener_version (screener_id, version, spec_json)
        VALUES (?, 1, ?)`, id, specJSON); err != nil {
		return domain.SavedScreener{}, fmt.Errorf("insert saved screener version: %w", err)
	}
	saved, err := getSavedScreenerTx(ctx, tx, id)
	if err != nil {
		return domain.SavedScreener{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.SavedScreener{}, fmt.Errorf("commit saved screener create: %w", err)
	}
	rollback = false
	return saved, nil
}

func (store *MySQLSavedScreenerStore) List(ctx context.Context, page, pageSize int) ([]domain.SavedScreener, int64, error) {
	var total int64
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM t_screener`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count saved screeners: %w", err)
	}
	offset := (page - 1) * pageSize
	rows, err := store.db.QueryContext(ctx, `
        SELECT s.id, s.name, s.description, s.current_version, s.created_at, s.updated_at, v.spec_json
        FROM t_screener AS s
        INNER JOIN t_screener_version AS v ON v.screener_id = s.id AND v.version = s.current_version
        ORDER BY s.updated_at DESC, s.id DESC
        LIMIT ? OFFSET ?`, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list saved screeners: %w", err)
	}
	defer rows.Close()
	items := make([]domain.SavedScreener, 0, pageSize)
	for rows.Next() {
		item, err := scanSavedScreener(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate saved screeners: %w", err)
	}
	return items, total, nil
}

func (store *MySQLSavedScreenerStore) Get(ctx context.Context, id int64) (domain.SavedScreener, error) {
	saved, err := getSavedScreener(ctx, store.db, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.SavedScreener{}, domain.ErrSavedScreenerNotFound
	}
	if err != nil {
		return domain.SavedScreener{}, fmt.Errorf("read saved screener: %w", err)
	}
	return saved, nil
}

func (store *MySQLSavedScreenerStore) Update(ctx context.Context, id, version int64, input domain.SavedScreenerInput) (domain.SavedScreener, error) {
	specJSON, err := marshalSavedSpec(input.Spec)
	if err != nil {
		return domain.SavedScreener{}, err
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.SavedScreener{}, fmt.Errorf("begin saved screener update: %w", err)
	}
	rollback := true
	defer func() {
		if rollback {
			_ = tx.Rollback()
		}
	}()
	var currentVersion int64
	err = tx.QueryRowContext(ctx, `SELECT current_version FROM t_screener WHERE id = ? FOR UPDATE`, id).Scan(&currentVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.SavedScreener{}, domain.ErrSavedScreenerNotFound
	}
	if err != nil {
		return domain.SavedScreener{}, fmt.Errorf("lock saved screener: %w", err)
	}
	if currentVersion != version {
		return domain.SavedScreener{}, &domain.SavedScreenerVersionConflictError{CurrentVersion: currentVersion}
	}
	nextVersion := currentVersion + 1
	if _, err := tx.ExecContext(ctx, `
        INSERT INTO t_screener_version (screener_id, version, spec_json)
        VALUES (?, ?, ?)`, id, nextVersion, specJSON); err != nil {
		return domain.SavedScreener{}, fmt.Errorf("insert updated saved screener version: %w", err)
	}
	result, err := tx.ExecContext(ctx, `
        UPDATE t_screener
        SET name = ?, description = ?, current_version = ?, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?`, input.Name, nullableDescription(input.Description), nextVersion, id)
	if err != nil {
		return domain.SavedScreener{}, fmt.Errorf("update saved screener: %w", err)
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		if err != nil {
			return domain.SavedScreener{}, fmt.Errorf("read saved screener update count: %w", err)
		}
		return domain.SavedScreener{}, domain.ErrSavedScreenerNotFound
	}
	saved, err := getSavedScreenerTx(ctx, tx, id)
	if err != nil {
		return domain.SavedScreener{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.SavedScreener{}, fmt.Errorf("commit saved screener update: %w", err)
	}
	rollback = false
	return saved, nil
}

func marshalSavedSpec(spec domain.ScreenerSpec) ([]byte, error) {
	encoded, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("serialize saved screener spec: %w", err)
	}
	return encoded, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func getSavedScreener(ctx context.Context, query interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, id int64) (domain.SavedScreener, error) {
	return scanSavedScreener(query.QueryRowContext(ctx, savedScreenerQuery, id))
}

func getSavedScreenerTx(ctx context.Context, tx *sql.Tx, id int64) (domain.SavedScreener, error) {
	return scanSavedScreener(tx.QueryRowContext(ctx, savedScreenerQuery, id))
}

const savedScreenerQuery = `
    SELECT s.id, s.name, s.description, s.current_version, s.created_at, s.updated_at, v.spec_json
    FROM t_screener AS s
    INNER JOIN t_screener_version AS v ON v.screener_id = s.id AND v.version = s.current_version
    WHERE s.id = ?`

func scanSavedScreener(scanner rowScanner) (domain.SavedScreener, error) {
	var saved domain.SavedScreener
	var description sql.NullString
	var specJSON []byte
	if err := scanner.Scan(&saved.ID, &saved.Name, &description, &saved.Version, &saved.CreatedAt, &saved.UpdatedAt, &specJSON); err != nil {
		return domain.SavedScreener{}, err
	}
	if description.Valid {
		saved.Description = &description.String
	}
	spec, err := decodeSavedSpec(specJSON)
	if err != nil {
		return domain.SavedScreener{}, err
	}
	saved.Spec = spec
	return saved, nil
}

func decodeSavedSpec(raw []byte) (domain.ScreenerSpec, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	var spec domain.ScreenerSpec
	if err := decoder.Decode(&spec); err != nil {
		return domain.ScreenerSpec{}, fmt.Errorf("decode saved screener spec: %w", err)
	}
	normalized, err := domain.NormalizeAndValidateSpec(spec)
	if err != nil {
		return domain.ScreenerSpec{}, fmt.Errorf("validate saved screener spec: %w", err)
	}
	return normalized, nil
}

func nullableDescription(description *string) any {
	if description == nil {
		return nil
	}
	return *description
}

var _ interface {
	Migrate(context.Context) error
} = (*MySQLSavedScreenerStore)(nil)
