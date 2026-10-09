# Tushare 基础设施导航

| 文件 | 职责 |
|---|---|
| `doc.go` | Tushare 适配器包声明 |
| `client.go` | 有界 HTTP 客户端、provider envelope 与动态行解析 |
| `resilience.go` | 进程内 global/per-api 限流、有限重试及安全观测接口 |
| `dto.go` | 六个行情接口的 provider DTO；数值字段使用精确 `types.Decimal` |
| `mapping.go` | 按字段名解码 DTO、校验空值/日期/范围，并映射市场领域数据和单位 |
| `client_test.go` | 请求结构、错误映射、超时和 Token 脱敏测试 |
| `resilience_test.go` | fake clock 限流/取消恢复、重试分类/取消、slog 脱敏与瞬时网络错误测试 |
| `mapping_test.go` | 官方字段 fixture、动态列序、nullable、数值/日期校验及精确单位换算 |
| `testdata/` | 字段乱序、空结果和权限错误本地 fixture |
