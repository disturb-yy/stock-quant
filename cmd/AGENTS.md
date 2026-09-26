# cmd 局部规约

本目录保存可执行程序的 Composition Root。遵循根 AGENTS.md；子目录负责各自程序的启动和依赖组合。

## 职责

- 组合配置、共享 Infrastructure、领域 Adapter、Application Service 与 Interface。
- 保持入口层薄，不承载领域规则、SQL 或 HTTP DTO 处理。

## 边界与验证

- 允许修改启动参数、依赖注入、生命周期和进程级错误处理。
- 领域行为应修改所属 internal 包；迁移 SQL 应修改 migrations。
- 最小验证：执行 go test -mod=readonly ./cmd/... -count=1。
