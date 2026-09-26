# internal/infrastructure/database 局部规约

本目录保存跨 bounded context 复用的数据库技术能力。遵循父级 internal/infrastructure/AGENTS.md 与根 AGENTS.md。

## 职责

- 提供数据库连接创建、Ping 和连接池生命周期管理。
- 对外暴露技术连接实例，不解释任何领域表或业务模型。

## 边界与验证

- 数据库驱动和 GORM Dialector 属于本层；领域 Repository 属于对应 bounded context 的 Adapter。
- 不读取领域配置、不嵌入 schema、不实现业务查询。
- 最小验证：执行 go test -mod=readonly ./internal/infrastructure/database/... -count=1。
