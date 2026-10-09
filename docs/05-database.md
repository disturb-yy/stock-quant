# 05 — 数据库与迁移设计

数据库：MySQL 8.4，字符集 utf8mb4，时区所有事件 `UTC` datetime(6)，**业务交易日 DATE** 独立存储。统一表前缀 `t_`。当前结构由 `db/migrations/0001.up.sql` 与 `db/migrations/0002.up.sql` 共同定义；已提交迁移版本不可修改，后续变更新增版本。

## 表与聚合
- `t_schema_migrations`：迁移执行器维护的版本账本，记录版本号、不可变文件 checksum、dirty 状态和执行方向；down 不删除此账本。
- `t_stock`：按 ts_code 保存证券基础信息、上市退市日；`list_status` 只作当前辅助，历史池由上市/退市有效日期决定。
- `t_trade_calendar`：exchange/cal_date 唯一；开市标识。
- `t_daily_price`：`(ts_code,trade_date)` 唯一；未复权 OHLC、`amount_yuan`（元）、`volume_lot`（手）、`source_hash`、`revision`、`fetched_at`。仓储接收已转换为元的金额，不再乘 1000。
- `t_adj_factor`：`(ts_code,trade_date)` 唯一；正值、`source_hash`、`revision`、`fetched_at`。
- `t_stock_status_daily`：按日 ST、停牌状态与可信度、来源；`UNKNOWN` 必须有状态，不用 boolean 默认 false。
- `t_sync_job`：task_key 唯一（source/api/date），status/attempts/requested_at/received_rows/expected_rows/quality_report_json/last_error。同键提交返回已有任务；仅 PENDING 可进入 RUNNING，开始执行时 attempts 加一；RUNNING 可到 SUCCESS/FAILED/BLOCKED，终态不覆盖，失败重试使用新 task_key。
- `t_data_snapshot`：snapshot_id, as_of, revision/hash, complete, st_quality, created_at。冻结该次运行的关键数据与状态摘要；完整复现需保留对应数据版本或不可变导出引用。
- `t_strategy`：strategy id+version，参数 JSON、config_hash；已用于 run 的版本不原地覆盖。
- `t_screening_run`：唯一 run_key 代表策略版本+日期+config_hash+snapshot_hash。同键提交返回已有运行；状态为 PENDING/RUNNING/SUCCESS/FAILED/BLOCKED，成功、失败和阻塞均为终态；运行结果、拒绝原因摘要和 SUCCESS 在同一事务提交。
- `t_screening_result`：run_id+ts_code 唯一，raw_factors_json, factor_scores_json, total_score, final_rank, reason_json；存**全量筛后候选**或分表记录拒绝列表，不能只保留 Top20 而丢审计。
- `t_backtest_run`：run_id、唯一 run_key、策略版本、日期区间、config_hash、snapshot_hash、模式、状态、metrics/risk 和错误摘要。run_key 为策略、版本、日期区间、配置摘要、数据快照摘要和模式的 SHA-256；同键提交返回已有运行。0002 为旧记录按 run_id 确定性补入 run_key；旧记录未知的 snapshot_hash 保持 NULL，新运行必须绑定有效快照摘要。
- `t_backtest_equity`：run_id/date -> equity, cash, exposure, benchmark_equity。
- `t_backtest_trade`：交易意向、实际成交、拒绝原因、费用、数量、成交价。

## 行情仓储约定
- daily/adj_factor 的 `source_hash` 是单行规范化业务字段（不含 `fetched_at`/`revision`）的 SHA-256 十六进制摘要；原始整批 HTTP 响应 hash 属于同步批次审计信息，不能充当每行版本 hash。
- daily/adj_factor 实质字段或行 hash 改变时 `revision` 加 1；仅重抓时间改变时更新 `fetched_at`，不增加 revision。stock/calendar 当前 schema 不含行 hash/revision，仓储只 upsert 当前记录。
- 仓储拒绝超出 MySQL DECIMAL 总位数/scale 的值，不做静默舍入。`amount_yuan` 以人民币元传入；Tushare 千元转换归 provider 数据映射层。
- 代码范围查询按 `trade_date ASC` 返回；全市场日期查询按 `ts_code ASC` 返回；交易日历按日期升序。stock 查询不按 `list_status` 过滤，退市证券仍可读取历史资料和日线。
- 行情仓储提供按证券/日期区间读取及按交易日全市场读取；后者直接使用 `(trade_date,ts_code)` 索引并按代码稳定排序。

## 一致性与索引
- 同步单日 bulk UPSERT in transaction；标记 synced 只有在记录落库且校验通过之后。
- `t_screening_run` status = `PENDING|RUNNING|SUCCESS|FAILED|BLOCKED`；先 INSERT，再由 PENDING 进入 RUNNING；结果 batch、运行总数、拒绝摘要和 SUCCESS 原子保存，中途失败不能留下半结果。结果分页按 final_rank 升序，未排名结果最后，再按 ts_code 升序。
- `t_backtest_run` 使用与筛选相同的受限状态流转；同键唯一性由数据库约束保障。迁移 0002 可安全重试，回滚只移除新增列和索引，不删除既有回测记录。
- 关键查询索引：date+ts_code、strategy+as_of、run_id+rank；JSON 字段不作第一版热点过滤。
- 并发单运行 `run_key UNIQUE` 防重，冲突由唯一键决定，禁止先查后插竞争；多实例扩展可加租约，但首版单实例。

## 迁移执行与验收
- 迁移版本保存在 `t_schema_migrations`，记录版本、文件 checksum、执行方向和 dirty 状态；同版本 checksum 不一致时必须失败。
- 首次应用版本前，执行器会确认该迁移将创建的表均不存在；检测到同名既有对象时拒绝接管，不写入 applied 版本记录。MySQL DDL 会隐式提交。迁移语句须具备幂等性；中途失败保留 dirty 记录，修复引发失败的外部条件后，可用相同文件/checksum 显式重试。
- 使用 MySQL `GET_LOCK` 串行化迁移，迁移期间固定同一物理连接；重复 up 不改动已应用版本。down 每次只撤销最新版本，down 文件只删除该迁移创建的对象并保留版本表。
- CLI 只执行显式 `migrate up|down`，应用启动不自动迁移；down 仅允许 `APP_ENV=development|test`。
- 本阶段 P02-01 验收：空库 up 建表、重复 up 无变化、down 只删除本迁移对象、down 后再次 up、部分失败可重试、并发锁和数据库连接失败路径。
- 后续仓储/业务验收：同一个 ts_code/date 同步两次仅一条且修订时 revision++/source_hash 更新；坏记录触发 rollback 且同步任务不被标 SUCCESS；策略运行中断后 status FAILED；EXPLAIN 常用日期条件命中索引。这些不属于迁移执行器票据。

## 不可忽略
若后续迁移增加 CHECK 约束，须在目标 MySQL 版本核验其行为；应用层仍必须再次验证金额、日期和枚举，不能只依赖数据库约束。
