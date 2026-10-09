# P03 阶段门禁与证据

## 本阶段交付

- [P03-01：Tushare HTTP 与动态字段解析](P03-01.md) — 已验收。
- [P03-02：限流、重试与日志脱敏](P03-02.md) — 已验收。
- [P03-03：DTO、单位映射与精确十进制存储](P03-03.md) — 已验收。
- [P03-04：真实 Token 核心 API 权限验收](P03-04.md) — 四个核心 API 的最小真实查询通过；可选 API 保持未测试。

## 验证环境与证据

- 环境：Linux amd64；Go 1.27.0；Python 3.12.3；Tushare 官方 HTTPS endpoint。
- Go、Python、质量门及 race 证据沿用各 ticket handoff 中记录的真实执行结果；P03-01/02/03 均有 `make check`、`go test`/`go vet` 和 GitHub Actions 通过记录。P03-03 另完成 MySQL 8.4 隔离 schema 集成验证。
- P03-04 live smoke：2026-10-09，经用户授权，对 `stock_basic`、`trade_cal`、`daily`、`adj_factor` 各发 1 次窄参数请求；共 4 次，HTTP 200、接口码 0，各返回 1 行。请求范围见 [`../P03-04-live-smoke.md`](../P03-04-live-smoke.md)，权限状态见 [`../permission-matrix.md`](../permission-matrix.md)。
- `suspend_d`、`stk_limit`、`namechange` 未调用；`stock_st` 按高门槛边界跳过。Mock、fixtures 和官方积分门槛均未被用作真实权限证明。

## 覆盖与限制

- Tushare HTTP 协议、动态字段解析、权限错误、限流/重试、脱敏、DTO 映射、单位转换、精确十进制和 MySQL 持久化：详见对应 P03 ticket handoff 中的定向及集成验证记录。
- 当前实测只证明本次 Token 在 2026-10-09 对上述四个核心接口可用；不代表额度、未来服务可用性或其他 API 权限。
- 本阶段未实现数据同步、全市场回填、定时增量或历史身份逻辑；这些属于 P04/P05。

## 阶段兼容与交接

- 上一阶段 P02 gate 已通过；P03 DTO、Tushare client、权限矩阵及环境变量契约保持兼容。
- P04-01 可依赖的稳定入口、迁移与仓储见 `internal/infrastructure/tushare/`、`internal/infrastructure/mysql/`、`internal/market/ports/` 和 P03-01/02/03 handoff。
- ADR 与已知限制见各 ticket 交付记录及 `docs/14-decisions-risks.md`。

## Gate

- Gate：`PASS`。
- 审核人/日期：独立文档审核 agent；2026-10-09；P03-04 handoff 及权限证据 `CLEAR`。
- 下一阶段：可开始 P04-01。
