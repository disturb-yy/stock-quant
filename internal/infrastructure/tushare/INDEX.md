# Tushare 基础设施导航

| 文件 | 职责 |
|---|---|
| `doc.go` | Tushare 适配器包声明 |
| `client.go` | 有界 HTTP 客户端、provider envelope 与动态行解析 |
| `resilience.go` | 进程内 global/per-api 限流、有限重试及安全观测接口 |
| `client_test.go` | 请求结构、错误映射、超时和 Token 脱敏测试 |
| `resilience_test.go` | fake clock 限流/取消恢复、重试分类/取消、slog 脱敏与瞬时网络错误测试 |
| `testdata/` | 字段乱序、空结果和权限错误本地 fixture |
