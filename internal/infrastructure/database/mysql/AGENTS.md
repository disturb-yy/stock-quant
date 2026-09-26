# internal/infrastructure/database/mysql 局部规约

本目录提供共享 MySQL 连接能力。遵循父级 internal/infrastructure/database/AGENTS.md 与根 AGENTS.md。

## 职责

- 使用 GORM MySQL Dialector 创建 *gorm.DB。
- 设置连接池、执行 Ping，并将关闭责任交给调用方。

## 边界与验证

- 只处理连接技术参数和生命周期，不包含 t_* 表语义、领域 Record 或 Repository。
- 不读取环境变量；DSN 由 Composition Root 读取并传入。
- 最小验证：执行 go test -mod=readonly ./internal/infrastructure/database/mysql -count=1。
