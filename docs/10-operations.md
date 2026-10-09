# 10 — 任务编排、可观测和运维

## 任务状态机
`PENDING -> RUNNING -> SUCCESS|FAILED|BLOCKED|CANCELLED`。不可逆的 `SUCCESS` 不能回到 RUNNING；失败重试创建新的 attempt 或保留相同 job 并递增 attempt，最终记录 must distinguish each run。对于日频同步先校验 `trade_cal` 开市，再按 `stock_basic/status`, `daily`, `adj_factor`, optional `daily_basic` 顺序，最后 `quality` 和 `screen`。

## 触发
收盘后自动触发 (例 18:30 Asia/Shanghai)，但不能以到点作为数据完整证据。`Tushare daily` 和 `daily_basic` 可能不同步；同步/复权数据 ready 后选股。节假日和非交易日 SKIPPED（不算失败）。支持 CLI/HTTP 手动补指定日期。

## 故障处理
- 速率限制 -> 可退避重试（有限重试）；接口拒绝 -> BLOCKED/PERMISSION；网络异常 -> FAILED/retryable。
- 进程异常恢复：PENDING 或 RUNNING 超时任务可租约重领，幂等 key 防重复；不能双写结果。
- MySQL 短故障：事务提交失败不得标 SUCCESS；重复执行 `UPSERT` 后质量复查。
- Python 超时：进程树清理，临时数据删除，stdout 长度有限，run FAILED；失败原因不含秘密。

## 指标、日志、报警
metric：`tushare_calls_total{api,status}`、`sync_rows{date,api}`、`snapshot_coverage`、`screen_duration_seconds`、`python_worker_exit{reason}`。structured logs keys：request_id/run_id/strategy/version/date/timing/status；密钥及原始敏感内容一律掩码。

## 开发及运行命令（P00/P12 落地后）
```bash
make init
make test
make lint
make migrate-up
make run
stockquant sync --date 2026-10-08
stockquant screen --strategy momentum_v1 --date 2026-10-08
stockquant backtest --strategy momentum_v1 --from 2024-01-01 --to 2026-09-30
```
这里是约定**目标命令**，尚非本包已经存在的生产程序。
