# internal/health 局部规约

本目录提供 HTTP 进程存活检查。遵循父级 internal/AGENTS.md 与根 AGENTS.md。

## 职责

- 提供健康检查 Handler 和路由注册。
- 仅表达 HTTP 进程是否能够响应，不扩展为数据库或外部 Provider readiness。

## 边界与验证

- 不访问数据库、不调用领域 Service、不读取环境变量。
- HTTP 响应映射保持稳定并覆盖 Handler 测试。
- 最小验证：执行 go test -mod=readonly ./internal/health -count=1。
