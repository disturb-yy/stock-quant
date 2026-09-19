package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/disturb-yy/stock-quant/internal/research"
	"github.com/disturb-yy/stock-quant/internal/research/domain"
	_ "github.com/go-sql-driver/mysql"
)

// TestMySQLResearchStore 只在显式测试 DSN 下验证真实迁移、Seed、创建和最近排序。
func TestMySQLResearchStore(t *testing.T) {
	dsn := os.Getenv("STOCK_QUANT_TEST_DSN")
	if dsn == "" {
		t.Skip("set STOCK_QUANT_TEST_DSN to run the MySQL integration test")
	}
	database, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open test MySQL: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := database.PingContext(context.Background()); err != nil {
		t.Fatalf("ping test MySQL: %v", err)
	}
	store, err := NewMySQLResearchStore(database)
	if err != nil {
		t.Fatalf("NewMySQLResearchStore() error = %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("first migration: %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
	if err := store.SeedDemo(context.Background()); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if err := store.SeedDemo(context.Background()); err != nil {
		t.Fatalf("repeat seed: %v", err)
	}

	var seededCount int
	if err := database.QueryRowContext(context.Background(), `
        SELECT COUNT(*) FROM t_research_workspace
        WHERE seed_key IN (?, ?)`, demoResearchProjects[0].seedKey, demoResearchProjects[1].seedKey).Scan(&seededCount); err != nil {
		t.Fatalf("count seeded research projects: %v", err)
	}
	if seededCount != 2 {
		t.Fatalf("seeded research project count = %d, want 2", seededCount)
	}

	service, err := research.NewService(store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	projects, err := service.List(context.Background(), research.ResearchListRequest{Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("list seeded research projects: %v", err)
	}
	positions := make(map[string]int)
	for index, project := range projects.Data {
		if project.Name == demoResearchProjects[0].name || project.Name == demoResearchProjects[1].name {
			positions[project.Name] = index
		}
	}
	if len(positions) != 2 {
		t.Fatalf("seeded project positions = %#v, want both demo projects", positions)
	}
	if positions[demoResearchProjects[1].name] >= positions[demoResearchProjects[0].name] {
		t.Fatalf("seeded ordering = %#v, want newer project first", positions)
	}

	description := "集成测试描述"
	name := fmt.Sprintf("集成 Research 项目-%d", time.Now().UnixNano())
	created, err := service.Create(context.Background(), domain.ResearchProjectInput{Name: name, Description: &description})
	if err != nil {
		t.Fatalf("create research project: %v", err)
	}
	loaded, err := service.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get created research project: %v", err)
	}
	if loaded.ID != created.ID || loaded.Name != name || loaded.Description == nil || *loaded.Description != description {
		t.Fatalf("loaded research project = %#v, want created project", loaded)
	}
	if _, err := service.Get(context.Background(), int64(1<<63-1)); !errors.Is(err, domain.ErrResearchProjectNotFound) {
		t.Fatalf("missing research project error = %v, want not found", err)
	}
}
