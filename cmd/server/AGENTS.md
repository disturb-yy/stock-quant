# cmd/server 局部规约

本目录是 HTTP 服务的 Composition Root。遵循父级 cmd/AGENTS.md 与根 AGENTS.md。

## 职责

- main.go 负责加载配置、创建共享 GORM MySQL 连接、组装领域 Adapter 与 Application Service，并处理启动和关闭。
- router.go 负责 Gin Engine、全局中间件、版本路由组和各领域路由注册。

## 边界与验证

- 允许修改依赖注入、服务生命周期和全局路由装配。
- 业务规则、Repository SQL、外部 Provider 和 DTO 处理留在所属 internal 包。
- router.go 不直接访问数据库，不创建匿名业务 Handler。
- 最小验证：执行 go test -mod=readonly ./cmd/server ./internal/... -count=1。
