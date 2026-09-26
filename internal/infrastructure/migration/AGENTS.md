# internal/infrastructure/migration 局部规约

本目录提供共享 migration runner。遵循父级 internal/infrastructure/AGENTS.md 与根 AGENTS.md。

## 职责

- 接收调用方提供的 fs.FS、迁移版本方向和 GORM 数据库实例。
- 使用 golang-migrate 执行版本化 SQL。

## 边界与验证

- 不嵌入或解析具体 bounded context 的 schema；schema 资源由调用方提供。
- 不使用 GORM AutoMigrate 替代版本化 migration。
- 最小验证：执行 go test -mod=readonly ./internal/infrastructure/migration -count=1。
