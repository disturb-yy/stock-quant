# P03-04 真实 Token 权限最小验收报告

## 结果

**状态：`PASS`（本票核心接口范围）。** 用户已配置本地 `TUSHARE_TOKEN` 并授权最小真实查询。2026-10-09 对 Tushare 官方 HTTPS 接口发出 4 次请求，每个核心 API 各 1 次；均返回 HTTP 200、接口码 `0`、1 行数据。Token 未写入输出或证据。可选接口和高门槛接口未调用，仍为 `NOT_TESTED`。

## 已完成的本地检查

- 官方公开最低积分门槛已整理至 [`permission-matrix.md`](permission-matrix.md)，并与账户实测列分开维护。
- 本次以窄参数执行 HTTPS JSON POST；每个 API 一次，无重试：

| API | 请求范围 | HTTP | 接口码 | 行数 | 结果 |
|---|---|---:|---:|---:|---|
| `stock_basic` | `ts_code=600000.SH`；字段 `ts_code,name,list_status` | 200 | 0 | 1 | `PASS` |
| `trade_cal` | `exchange=SSE`，`20261008` 单日；字段 `exchange,cal_date,is_open,pretrade_date` | 200 | 0 | 1 | `PASS` |
| `daily` | `ts_code=600000.SH`，`trade_date=20261008`；字段 `ts_code,trade_date,close` | 200 | 0 | 1 | `PASS` |
| `adj_factor` | `ts_code=600000.SH`，`trade_date=20261008`；字段 `ts_code,trade_date,adj_factor` | 200 | 0 | 1 | `PASS` |

- 真实 provider 请求数：`4`；核心账户 API 权限：全部 `PASS`。
- `suspend_d`、`stk_limit`、`namechange` 未调用，继续保持 `NOT_TESTED`；`stock_st` 因 3000 积分门槛而跳过。
- `make doctor` 只检查本地配置，不执行真实请求；其 `configured_unverified` 状态不能替代本报告的账户实测。
- 脱敏后的逐请求标准输出：

```text
stock_basic: http=200 code=0 result=PASS rows=1
trade_cal: http=200 code=0 result=PASS rows=1
daily: http=200 code=0 result=PASS rows=1
adj_factor: http=200 code=0 result=PASS rows=1
```

- 执行日期：`2026-10-09`。一次性查询命令未记录各请求的精确时分；本报告明确保留此证据限制，不推测时间。

## 范围和后续边界

1. 本报告只证明该 Token 在 2026-10-09 对上述四个核心 API 的调用成功，不证明未来额度、服务可用性或其他 API 权限。
2. 可选接口如需验收，须另行确认范围后执行；`stock_st` 需要明确的 3000 积分权限证据和单独验收需求。

P03-04 核心验收、独立交接审核和 P03 阶段门禁均已通过；下一票为 P04-01。
