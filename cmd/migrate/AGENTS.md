# cmd/migrate 局部规约

本目录是数据库迁移命令行入口。遵循父级 cmd/AGENTS.md 与根 AGENTS.md。

## 职责

- 读取数据库配置。
- 使用共享 MySQL GORM 连接和 migration runner 执行 up 或 down。
- 管理命令行进程级退出码和连接关闭。

## 边界与验证

- 不在此目录编写迁移 SQL、领域 Repository 或业务规则。
- 迁移资源由根目录 migrations 提供；通用执行逻辑由 internal/infrastructure/migration 提供。
- 最小验证：执行 go test -mod=readonly ./cmd/migrate ./internal/infrastructure/... -count=1。
