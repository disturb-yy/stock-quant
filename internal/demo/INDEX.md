# INDEX.md

## Package

`internal/demo`

## Role

FND-003 开发环境 fixture、状态查询、seed 用例与 dev-only HTTP 适配。

## Current Contents

- `fixture.go`：`fnd-003-demo-v5` 的确定股票、行业分类、行业成分、121 个交易日日线、daily basic、带口径财务指标、指数 Seed 数据及版本校验。
- `service.go`：状态查询和 seed 应用边界。
- `handler.go`：`GET /api/v1/dev/demo-status`，依赖存储接口，不直连数据库。
- `infrastructure/mysql.go`：MySQL Driver、`0001_demo_seed` 至 `0004_stock_overview` migration、行业分类与成分/daily basic/财务指标幂等 upsert 和状态查询。
- `infrastructure/mysql_integration_test.go`：显式测试 DSN 下的 migration/seed 重复执行验证。
