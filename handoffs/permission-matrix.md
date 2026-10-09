# Tushare 权限矩阵

**核对日期：**2026-10-09　**核心接口实测状态：**`PASS`（核心接口全部返回成功）；可选接口仍为 `NOT_TESTED`

官方积分门槛是公开资料，不代表当前账户的实际权限。只有受人工授权的真实 Token 调用可以更新“实测状态”；本地 fixtures、文档和客户端权限错误测试均不计为账户权限证据。

| API | 项目分级 | 官方公开最低门槛 | 当前 Token 实测 | 真实调用记录 |
|---|---|---:|---|---|
| `stock_basic` | 核心 | 2000 积分起 | `PASS` | 2026-10-09：`ts_code=600000.SH`；HTTP 200、接口码 0、1 行 |
| `trade_cal` | 核心 | 2000 积分 | `PASS` | 2026-10-09：SSE `20261008` 单日；HTTP 200、接口码 0、1 行 |
| `daily` | 核心 | 120 积分起 | `PASS` | 2026-10-09：`600000.SH` / `20261008`；HTTP 200、接口码 0、1 行 |
| `adj_factor` | 核心 | 2000 积分起 | `PASS` | 2026-10-09：`600000.SH` / `20261008`；HTTP 200、接口码 0、1 行 |
| `suspend_d` | 可选 | 2000 积分起 | `NOT_TESTED` | 未调用；不属于本票核心接口验收范围 |
| `stk_limit` | 可选 | 2000 积分起 | `NOT_TESTED` | 未调用；不属于本票核心接口验收范围 |
| `namechange` | 可选 | 官方接口页未列最低积分 | `NOT_TESTED` | 未调用；不属于本票核心接口验收范围 |
| `stock_st` | 高门槛，非核心 | 3000 积分起 | `NOT_TESTED` | 未调用；本票明确跳过高门槛接口 |

官方资料来源： [API 权限总表](https://tushare.pro/document/1?doc_id=108)、[stock_basic](https://tushare.pro/document/1?doc_id=25)、[trade_cal](https://tushare.pro/document/2?doc_id=26)、[daily](https://tushare.pro/document/2?doc_id=27)、[adj_factor](https://tushare.pro/document/2?doc_id=28)、[suspend_d](https://tushare.pro/document/2?doc_id=214)、[stk_limit](https://tushare.pro/document/2?doc_id=183)、[stock_st](https://tushare.pro/document/2?doc_id=397)、[namechange](https://tushare.pro/wctapi/documents/100.md)。

## 本地 Mock 与凭据状态

- `DATA_PROVIDER=mock` 的 doctor 输出 `provider_status=offline/mock`；该状态不表示真实同步成功。
- `DATA_PROVIDER=tushare` 缺少 `TUSHARE_TOKEN` 时诊断失败，不会静默回退到 Mock。
- 配置 Token 后，环境诊断仅输出 `configured_unverified`；本表中的 `PASS` 来自 2026-10-09 经授权发出的真实最小查询，不来自 doctor、fixture 或官方积分门槛。
- P03-04 本地单测和 HTTP fixture 仅覆盖映射、权限错误分类与安全边界；真实账户权限以本日 live smoke 记录为准。
