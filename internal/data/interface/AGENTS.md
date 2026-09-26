# internal/data/interface 局部规约

本目录保存 A 股数据 bounded context 的外部接口适配。遵循父级 internal/data/AGENTS.md 与根 AGENTS.md。

## 职责

- 将外部协议请求转换为 Application 输入。
- 将 Application 结果和领域错误转换为已确认的外部响应。

## 边界与验证

- 不直接访问 Repository、GORM、SQL 或外部数据源。
- HTTP 相关实现放在 http/；新增其他协议时保持独立适配边界。
- 最小验证：执行 go test -mod=readonly ./internal/data/interface/... -count=1。
