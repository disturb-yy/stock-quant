# INDEX.md

## Package

`internal/migration`

## Role

数据库 migration 的命名、注册和执行边界。

## Naming and Execution

- 每个 migration 通过 `Migration.Name()` 提供名称；名称必须非空且唯一，建议使用 `0001_description` 形式。
- `Runner` 按注册顺序执行 migration，遇到取消或首个错误立即停止。
- Runner 本身不接入数据库 Driver；业务 migration 在对应 Infrastructure package 注册和执行。

## Current Contents

- `migration.go`：校验 migration 定义并按注册顺序执行。
- `migration_test.go`：验证空框架、命名校验、顺序、错误包装和取消行为。
- `internal/demo/infrastructure/mysql.go`：注册 `0001_demo_seed` 至 `0006_stock_financials` 并在 MySQL 中创建 FND-003 schema。
