// Package infrastructure 提供 Research 项目的 MySQL 持久化实现。
package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/disturb-yy/stock-quant/internal/research/domain"
)

const researchMigration = "0011_research_workspace"

const createResearchWorkspaceSQL = `
CREATE TABLE IF NOT EXISTS t_research_workspace (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500) NULL,
    seed_key VARCHAR(100) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_t_research_workspace_seed_key (seed_key),
    INDEX idx_t_research_workspace_updated_at (updated_at, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

type researchSeedProject struct {
	seedKey     string
	name        string
	description string
	createdAt   string
	updatedAt   string
}

var demoResearchProjects = []researchSeedProject{
	{
		seedKey:     "fnd-003-demo-v8-research-valuation",
		name:        "演示研究项目-估值",
		description: "用于验证 Research 项目创建、详情和最近更新时间排序。",
		createdAt:   "2024-06-28 09:00:00",
		updatedAt:   "2024-06-28 09:00:00",
	},
	{
		seedKey:     "fnd-003-demo-v8-research-industry",
		name:        "演示研究项目-行业",
		description: "用于验证最近研究列表中的第二个可复核项目。",
		createdAt:   "2024-06-28 09:00:00",
		updatedAt:   "2024-06-28 10:00:00",
	},
}

// MySQLResearchStore 实现 Research 项目的持久化、迁移和 Demo Seed。
type MySQLResearchStore struct {
	db *sql.DB
}

// NewMySQLResearchStore 创建 Research MySQL 存储。
func NewMySQLResearchStore(db *sql.DB) (*MySQLResearchStore, error) {
	if db == nil {
		return nil, errors.New("research database connection is required")
	}
	return &MySQLResearchStore{db: db}, nil
}

// Migrate 幂等创建 Research 项目表。
func (store *MySQLResearchStore) Migrate(ctx context.Context) error {
	if _, err := store.db.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS schema_migrations (
            name VARCHAR(191) NOT NULL PRIMARY KEY,
            applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
		return fmt.Errorf("create schema migrations table for research: %w", err)
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin research migration: %w", err)
	}
	rollback := true
	defer func() {
		if rollback {
			_ = tx.Rollback()
		}
	}()
	var applied int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, researchMigration).Scan(&applied); err != nil {
		return fmt.Errorf("check migration %q: %w", researchMigration, err)
	}
	if applied > 0 {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit existing migration %q: %w", researchMigration, err)
		}
		rollback = false
		return nil
	}
	if _, err := tx.ExecContext(ctx, createResearchWorkspaceSQL); err != nil {
		return fmt.Errorf("create t_research_workspace: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES (?)`, researchMigration); err != nil {
		return fmt.Errorf("record migration %q: %w", researchMigration, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %q: %w", researchMigration, err)
	}
	rollback = false
	return nil
}

// Create 持久化一个由应用层校验过的 Research 项目。
func (store *MySQLResearchStore) Create(ctx context.Context, input domain.ResearchProjectInput) (domain.ResearchProject, error) {
	result, err := store.db.ExecContext(ctx, `
        INSERT INTO t_research_workspace (name, description)
        VALUES (?, ?)`, input.Name, nullableDescription(input.Description))
	if err != nil {
		return domain.ResearchProject{}, fmt.Errorf("insert research project: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.ResearchProject{}, fmt.Errorf("read research project id: %w", err)
	}
	return store.Get(ctx, id)
}

// List 按真实持久化更新时间、ID 倒序稳定分页读取最近项目。
func (store *MySQLResearchStore) List(ctx context.Context, page, pageSize int) ([]domain.ResearchProject, int64, error) {
	var total int64
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM t_research_workspace`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count research projects: %w", err)
	}
	rows, err := store.db.QueryContext(ctx, `
        SELECT id, name, description, created_at, updated_at
        FROM t_research_workspace
        ORDER BY updated_at DESC, id DESC
        LIMIT ? OFFSET ?`, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list research projects: %w", err)
	}
	defer rows.Close()
	items := make([]domain.ResearchProject, 0, pageSize)
	for rows.Next() {
		item, err := scanResearchProject(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan research project: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate research projects: %w", err)
	}
	return items, total, nil
}

// Get 读取一个 Research 项目的真实基础元数据。
func (store *MySQLResearchStore) Get(ctx context.Context, id int64) (domain.ResearchProject, error) {
	project, err := scanResearchProject(store.db.QueryRowContext(ctx, `
        SELECT id, name, description, created_at, updated_at
        FROM t_research_workspace
        WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ResearchProject{}, domain.ErrResearchProjectNotFound
	}
	if err != nil {
		return domain.ResearchProject{}, fmt.Errorf("read research project: %w", err)
	}
	return project, nil
}

// SeedDemo 幂等写入两个具有不同更新时间的可复核 Research 项目。
func (store *MySQLResearchStore) SeedDemo(ctx context.Context) error {
	for _, project := range demoResearchProjects {
		if _, err := store.db.ExecContext(ctx, `
            INSERT INTO t_research_workspace (name, description, seed_key, created_at, updated_at)
            VALUES (?, ?, ?, ?, ?)
            ON DUPLICATE KEY UPDATE
                name = VALUES(name), description = VALUES(description),
                created_at = VALUES(created_at), updated_at = VALUES(updated_at)`,
			project.name, project.description, project.seedKey, project.createdAt, project.updatedAt); err != nil {
			return fmt.Errorf("seed research project %q: %w", project.seedKey, err)
		}
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanResearchProject(scanner rowScanner) (domain.ResearchProject, error) {
	var project domain.ResearchProject
	var description sql.NullString
	if err := scanner.Scan(&project.ID, &project.Name, &description, &project.CreatedAt, &project.UpdatedAt); err != nil {
		return domain.ResearchProject{}, err
	}
	if description.Valid {
		project.Description = &description.String
	}
	return project, nil
}

func nullableDescription(description *string) any {
	if description == nil {
		return nil
	}
	return *description
}

var _ interface {
	Migrate(context.Context) error
	Create(context.Context, domain.ResearchProjectInput) (domain.ResearchProject, error)
	List(context.Context, int, int) ([]domain.ResearchProject, int64, error)
	Get(context.Context, int64) (domain.ResearchProject, error)
} = (*MySQLResearchStore)(nil)
