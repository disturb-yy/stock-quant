# INDEX.md

## Package

`internal/demo`

## Role

FND-003 开发环境 fixture、状态查询、seed 用例与 dev-only HTTP 适配。

## Current Contents

- `fixture.go`：`fnd-003-demo-v2` 的确定股票、日线和指数 Seed 数据及版本校验。
- `service.go`：状态查询和 seed 应用边界。
- `handler.go`：`GET /api/v1/dev/demo-status`，依赖存储接口，不直连数据库。
- `infrastructure/mysql.go`：MySQL Driver、`0001_demo_seed`/`0002_market_overview` migration、幂等 upsert 和状态查询。
- `infrastructure/mysql_integration_test.go`：显式测试 DSN 下的 migration/seed 重复执行验证。
