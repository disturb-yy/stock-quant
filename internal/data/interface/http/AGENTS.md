# internal/data/interface/http 局部规约

本目录提供 A 股数据 HTTP Handler。遵循父级 internal/data/interface/AGENTS.md 与根 AGENTS.md。

## 职责

- 解析请求参数和 JSON DTO。
- 调用 data Application Service。
- 将结果、领域错误和分页转换为 HTTP 响应。
- 注册数据同步和股票查询路由。

## 边界与验证

- 可以依赖 Gin、data/application 和 data/domain。
- 不直接访问 GORM、Repository 具体实现、SQL 或 Tushare。
- 最小验证：执行 go test -mod=readonly ./internal/data/interface/http -count=1。
