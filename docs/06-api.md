# 06 — Go HTTP API v1

## 统一规则
- 前缀 `/api/v1`，JSON，业务日期 RFC3339 不适用：统一 `YYYY-MM-DD` 字符串。
- 统一错误 envelope：`{"error":{"code":"DATA_INCOMPLETE","message":"...","request_id":"...","details":{}}}`；可追踪不泄露秘密。
- GET 幂等；POST 执行任务返回 202 + `run_id`，相同 `Idempotency-Key` 和请求内容返回既有任务。底层 sync 使用 `task_key`，screen 使用 `run_key`，backtest 使用策略版本、日期区间、配置摘要、模式与 `snapshot_hash` 生成 SHA-256 run_key；唯一键冲突时返回已有运行。
- 列表支持 `limit` (1..100，默认20)、`offset` (>=0)。排序和筛选只允许白名单字段。

## API 列表及示例
| Method | Endpoint | 用途 | 返回/状态 |
|---|---|---|---|
| GET | `/healthz` | 进程存活 | 200 |
| GET | `/readyz` | DB/配置就绪 | 200/503 |
| GET | `/api/v1/strategies` | 列策略 | 200 |
| GET | `/api/v1/strategies/{id}` | 策略版本配置 | 200/404 |
| POST | `/api/v1/sync-jobs` | 提交历史/单日取数 | 202/409/422 |
| GET | `/api/v1/sync-jobs/{id}` | 查同步状态/质量 | 200/404 |
| GET | `/api/v1/data-quality?date=YYYY-MM-DD` | 快照准备度与缺失 | 200 |
| POST | `/api/v1/screen-runs` | 提交选股 | 202/409/422 |
| GET | `/api/v1/screen-runs/{id}` | 运行元数据/异常 | 200/404 |
| GET | `/api/v1/screen-runs/{id}/results` | Top N / 全量分数 | 200 |
| POST | `/api/v1/backtest-runs` | 提交回测 | 202 |
| GET | `/api/v1/backtest-runs/{id}` | 指标与状态 | 200/404 |
| GET | `/api/v1/backtest-runs/{id}/equity` | 净值时间序列 | 200 |

## POST /screen-runs 输入
```json
{"strategy_id":"momentum_v1","strategy_version":"1.0.0","trade_date":"2026-10-08","top_n":20,"mode":"strict"}
```
`202`:
```json
{"run_id":"<ULID>","status":"PENDING","links":{"self":"/api/v1/screen-runs/<ULID>"}}
```
`GET /screen-runs/{id}/results`:
```json
{"run_id":"<ULID>","as_of":"2026-10-08","strategy_version":"1.0.0","snapshot_hash":"sha256:...","total_eligible":300,"candidates":[{"ts_code":"000001.SZ","rank":1,"score":75.0,"factors":{"momentum_60":0.13,"momentum_20":0.07,"amount_activity_20":0.35,"volatility_20":0.2},"scores":{"momentum_60":80,"momentum_20":60,"amount_activity_20":90,"low_volatility":70},"reasons":["CLOSE_GT_MA20"]}]}
```
示例数据仅展示字段形状；实际横截面排名的值必须由真实同一批合格股票共同计算。

## 状态与错误矩阵
- 输入日期非交易日、窗口过短、无数据：422 `INVALID_ARGUMENT` / `DATA_INCOMPLETE`。
- 同一请求提交重复：相同 Idempotency-Key 返回旧任务，不重新计算；底层相同 task_key/run_key 返回既有任务或运行，不创建第二条记录。
- 策略版本不存在：404 `NOT_FOUND`；当前权限不足：403 `PERMISSION_DENIED`。
- 任务仅允许 `PENDING -> RUNNING -> SUCCESS|FAILED|BLOCKED`；SUCCESS/FAILED/BLOCKED 为终态，失败重试使用新任务键。筛选结果页按 rank 升序、未排名行最后、股票代码升序；结果、汇总和 SUCCESS 同一事务提交。计算中断时不能查询到半份成功结果。

## 验收 curl（生产代码实现后）
```bash
curl -fsS http://localhost:8080/healthz
curl -fsS http://localhost:8080/api/v1/strategies
curl -fsS 'http://localhost:8080/api/v1/data-quality?date=2026-10-08'
```
后续由 P09 建立 OpenAPI 文件或等效契约，并用 `httptest` 检查状态码、JSON、分页和权限。
