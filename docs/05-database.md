# 05 — 数据库与迁移设计

数据库：MySQL 8，字符集 utf8mb4，时区所有事件 `UTC` datetime(6)，**业务交易日 DATE** 独立存储。统一表前缀 `t_`。参见 `db/migrations/0001_init.sql`。

## 表与聚合
- `t_stock`：按 ts_code 保存证券基础信息、上市退市日；`list_status` 只作当前辅助，历史池由上市/退市有效日期决定。
- `t_trade_calendar`：exchange/cal_date 唯一；开市标识。
- `t_daily_price`： `(ts_code,trade_date)` 唯一；原始 OHLC，amount_yuan，volume_lot，source_hash，revision。
- `t_adj_factor`：`(ts_code,trade_date)` 唯一；正值、revision。
- `t_stock_status_daily`：按日 ST、停牌状态与可信度、来源；`UNKNOWN` 必须有状态，不用 boolean 默认 false。
- `t_sync_job`：task_key 唯一（source/api/date），status/attempts/requested/received/expected/quality_report/last_error。重跑复用键。
- `t_data_snapshot`：snapshot_id, as_of, revision/hash, complete, st_quality, created_at。冻结该次运行的关键数据与状态摘要；完整复现需保留对应数据版本或不可变导出引用。
- `t_strategy`：strategy id+version，参数 JSON、config_hash；已用于 run 的版本不原地覆盖。
- `t_screening_run`：唯一 run_key 代表策略版本+日期+config_hash+snapshot_hash。job 状态及拒绝原因摘要。
- `t_screening_result`：run_id+ts_code 唯一，raw factor JSON, factor score JSON, total_score, rank, reason JSON；存**全量筛后候选**或分表记录拒绝列表，不能只保留 Top20 而丢审计。
- `t_backtest_run`：run_id, 日期区间、执行/手续费配置、基准、metrics、数据版本。
- `t_backtest_equity`：run_id/date -> equity, cash, exposure, benchmark_equity。
- `t_backtest_trade`：交易意向、实际成交、拒绝原因、费用、数量、成交价。

## 一致性与索引
- 同步单日 bulk UPSERT in transaction；标记 synced 只有在记录落库且校验通过之后。
- `t_screening_run` status = `PENDING|RUNNING|SUCCESS|FAILED|BLOCKED`；先 INSERT/RUNNING，再结果 batch 保存，最后原子设置 SUCCESS；中途失败不能让 UI 看到半结果。
- 关键查询索引：date+ts_code、strategy+as_of、run_id+rank；JSON 字段不作第一版热点过滤。
- 并发单运行 `run_key UNIQUE` 防重，冲突由唯一键决定，禁止先查后插竞争；多实例扩展可加租约，但首版单实例。

## 迁移验收
1. 全新 DB migrate up 能完成；重复 migrate up 不破坏。
2. 对同一个 ts_code/date 同步两次仅一条，若修订则 revision++/source_hash 更新。
3. 注入一条坏记录触发 rollback，同步任务不被标 SUCCESS。
4. 策略运行中断后 status FAILED，历史 SUCCESS 结果不可被覆盖。
5. EXPLAIN 常用日期条件命中索引。

## 不可忽略
MySQL 8 对 CHECK/JSON 的行为视具体版本核验；尽管 DB 定义了检查约束，应用层仍必须再次验证金额/日期/枚举，不能只依赖 CHECK。
