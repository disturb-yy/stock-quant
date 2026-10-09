# Tushare 权限矩阵

**核对日期：**2026-10-09　**账户实测状态：**`BLOCKED`（本机没有配置 Token，且未发起真实调用授权）

官方积分门槛是公开资料，不代表当前账户的实际权限。只有受人工授权的真实 Token 调用可以更新“实测状态”；本地 fixtures、文档和客户端权限错误测试均不计为账户权限证据。

| API | 项目分级 | 官方公开最低门槛 | 当前 Token 实测 | 真实调用记录 |
|---|---|---:|---|---|
| `stock_basic` | 核心 | 2000 积分起 | `NOT_TESTED` | 无；Token 未配置 |
| `trade_cal` | 核心 | 2000 积分 | `NOT_TESTED` | 无；Token 未配置 |
| `daily` | 核心 | 120 积分起 | `NOT_TESTED` | 无；Token 未配置 |
| `adj_factor` | 核心 | 2000 积分起 | `NOT_TESTED` | 无；Token 未配置 |
| `suspend_d` | 可选 | 2000 积分起 | `NOT_TESTED` | 无；Token 未配置 |
| `stk_limit` | 可选 | 2000 积分起 | `NOT_TESTED` | 无；Token 未配置 |
| `namechange` | 可选 | 官方接口页未列最低积分 | `NOT_TESTED` | 无；Token 未配置 |
| `stock_st` | 高门槛，非核心 | 3000 积分起 | `NOT_TESTED` | 未调用；无 3000 积分权限证据 |

官方资料来源： [API 权限总表](https://tushare.pro/document/1?doc_id=108)、[stock_basic](https://tushare.pro/document/1?doc_id=25)、[trade_cal](https://tushare.pro/document/2?doc_id=26)、[daily](https://tushare.pro/document/2?doc_id=27)、[adj_factor](https://tushare.pro/document/2?doc_id=28)、[suspend_d](https://tushare.pro/document/2?doc_id=214)、[stk_limit](https://tushare.pro/document/2?doc_id=183)、[stock_st](https://tushare.pro/document/2?doc_id=397)、[namechange](https://tushare.pro/wctapi/documents/100.md)。

## 本地 Mock 与凭据状态

- `DATA_PROVIDER=mock` 的 doctor 输出 `provider_status=offline/mock`；该状态不表示真实同步成功。
- `DATA_PROVIDER=tushare` 缺少 `TUSHARE_TOKEN` 时诊断失败，不会静默回退到 Mock。
- 配置 Token 只会输出 `provider_status=configured_unverified`；环境诊断不调用 provider，也不证明权限。
- P03-04 本地单测和 HTTP fixture 仅覆盖映射、权限错误分类与安全边界；未触发任何 Tushare 网络请求。
