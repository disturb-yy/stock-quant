# stock-quant

股票分析量化平台后端基础框架。当前包含服务启动、优雅退出、健康检查，以及 A 股数据同步与查询能力。

## 技术选型

- 运行时：Go 1.27。
- HTTP：Gin。
- 数据访问：GORM 的 MySQL Dialector；A 股数据 adapter 使用 GORM 完成领域数据查询、事务与持久化。
- Schema 迁移：`golang-migrate` 执行版本化 SQL；GORM 不替代 migration。
- 共享技术基础设施位于 `internal/infrastructure`，A 股数据领域特有 adapter 位于 `internal/data/adapter`。

## 运行

```bash
DATABASE_DSN='user:password@tcp(127.0.0.1:3307)/stock_quant?parseTime=true' go run ./cmd/server
```

默认监听 `:8357`，可用 `HTTP_ADDRESS` 覆盖。

默认数据源为 `mock`；使用 Tushare 时设置 `DATA_SOURCE_PROVIDER=tushare`、`TUSHARE_TOKEN` 和可选的 `TUSHARE_ENDPOINT`。数据库 schema 使用 `cmd/migrate` 初始化：

```bash
DATABASE_DSN='user:password@tcp(127.0.0.1:3307)/stock_quant?parseTime=true' go run ./cmd/migrate -direction up
```

```bash
curl http://127.0.0.1:8357/api/v1/health
```

## 验证

```bash
go test ./...
go vet ./...
go build ./...
```
