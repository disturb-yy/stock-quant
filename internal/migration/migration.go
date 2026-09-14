// Package migration 定义数据库 migration 的命名、注册和执行边界。
// 具体数据库连接和业务 schema 由 Infrastructure package 实现。
package migration

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Migration 表示一个具名且只向前执行的 schema 变更。
type Migration interface {
	Name() string
	Up(context.Context) error
}

type registeredMigration struct {
	name      string
	migration Migration
}

// Runner 按注册顺序执行已注册的 migration。
type Runner struct {
	migrations []registeredMigration
}

// NewRunner 校验并创建 migration runner。
func NewRunner(migrations ...Migration) (Runner, error) {
	seenNames := make(map[string]struct{}, len(migrations))
	registered := make([]registeredMigration, 0, len(migrations))
	for _, migration := range migrations {
		if migration == nil {
			return Runner{}, errors.New("migration is required")
		}

		name := strings.TrimSpace(migration.Name())
		if name == "" {
			return Runner{}, errors.New("migration name is required")
		}
		if _, exists := seenNames[name]; exists {
			return Runner{}, fmt.Errorf("duplicate migration name %q", name)
		}
		seenNames[name] = struct{}{}
		registered = append(registered, registeredMigration{name: name, migration: migration})
	}

	return Runner{migrations: registered}, nil
}

// Count 返回已注册的 migration 数量。
func (runner Runner) Count() int {
	return len(runner.migrations)
}

// Run 执行所有 migration，在取消或遇到首个错误时停止。
func (runner Runner) Run(ctx context.Context) error {
	if ctx == nil {
		return errors.New("migration context is required")
	}

	for _, migration := range runner.migrations {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("run migrations: %w", err)
		}
		if err := migration.migration.Up(ctx); err != nil {
			return fmt.Errorf("run migration %q: %w", migration.name, err)
		}
	}

	return nil
}
