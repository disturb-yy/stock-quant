# 04 — Tushare 2000 积分取数协议与数据治理

## 权限表（以实际 Token 调用为最终准绳）
- `stock_basic`：证券基础资料，2000 积分起，**每分钟 50 次**（单接口独立限流）；建议 `list_status=L,D,P` 各自按需拉取，不只取 L，否则历史幸存者偏差。官方：https://tushare.pro/document/1?doc_id=25
- `trade_cal`：沪深交易日；https://tushare.pro/document/2?doc_id=26
- `daily`：未复权日线，`amount` 为**千元**、`vol` 为手，停牌日无记录，按 `trade_date` 拉全市场；https://tushare.pro/document/2?doc_id=27
- `adj_factor`：个股复权因子，支持按日拉取全市场；https://tushare.pro/document/2?doc_id=28
- `daily_basic`：估值扩展预留，第一版策略不强依赖；https://tushare.pro/document/2?doc_id=32
- `suspend_d`：停复牌，2000 积分可用，单次约5000记录；https://tushare.pro/document/2?doc_id=214
- `stk_limit`：涨跌停价，2000 积分可用，后续回测成交约束；https://tushare.pro/document/2?doc_id=183
- `namechange`：曾用名，可作为 ST 研究辅助，但**不可保证完整准确还原每个历史 ST 状态**；应先验权再使用。
- `stock_st`：3000 积分起，不可作为当前 2000 方案前提；https://tushare.pro/document/2?doc_id=397
- `pro_bar`：Python SDK 的组合接口，无法直接使用 HTTP；Go 应使用 `daily` + `adj_factor`；https://tushare.pro/document/1?doc_id=109

## HTTP 契约
POST `https://api.tushare.pro`（如证书或地址文档更新，使用配置覆盖）；`{"api_name":"daily","token":"<SECRET>","params":{"trade_date":"20261008"},"fields":"ts_code,trade_date,open,high,low,close,vol,amount"}`。响应含 `code,msg,data.fields,data.items`。以返回 `fields` 索引解析 `items`，不依赖固定列序；未知字段兼容、必需字段缺失报错。官方：https://tushare.pro/document/1?doc_id=40

客户端用 `context.Context` 发送请求，未提供更短调用期限时默认 30 秒；仅允许 HTTPS provider endpoint，HTTP 只可用于 loopback 测试。HTTP 429 映射 `RATE_LIMITED`，provider code 2002 映射 `PERMISSION_DENIED`，超时/取消分别映射 `TIMEOUT`/`CANCELLED`，不可用上游映射 `UPSTREAM_UNAVAILABLE`，不完整响应映射 `DATA_INCOMPLETE`。错误信息不得包含 Token 或原始请求体。

进程内限流同时应用 global 100 次/分钟和 per-api 默认 100 次/分钟，`stock_basic` 默认 40 次/分钟；配置可覆盖这些速率。令牌桶突发容量固定为 1，以平滑启动。每次实际 HTTP 尝试（包括重试）都重新取得 global 和对应 API 配额。重试默认最多 3 次尝试（含首次请求），对 HTTP 429、5xx 和明确瞬时网络错误使用指数退避与 full jitter；默认退避基数 100ms、上限 2s。权限码 2002、其他 provider 业务拒绝、参数错误、响应解析错误、取消和期限到达均不重试。限流等待、退避和网络调用共享同一个调用 context 与默认 30 秒总期限。

客户端默认使用 `slog.Default()` 写入结构化查询结束日志，也可注入 `*slog.Logger`。日志只包含随机本地 `request_id`、脱敏后的 `api_name`、包含等待/退避在内的总时延、行数、业务日期、错误分类和尝试次数；业务日期只从 `trade_date`、`start_date`、`end_date` 白名单参数中提取。日志不包含错误字符串、Token、请求体、任意参数或 provider 原始消息。

客户端还可注入并发安全的 `tushare.Observer` 接收同一组脱敏查询摘要，供指标适配器使用。Observer 不接收 Token、请求体、任意参数或 provider 原始消息。指标适配器不得将 request_id 或业务日期作为标签。

实现核验时，Tushare 官方 HTTP 页面仍以 `http://api.tushare.pro` 为请求示例；本项目保持文档指定的 HTTPS 默认并禁止自动降级，以免明文发送 Token。真实 TLS 连通性与 provider 权限尚未验证，后续真实调用若不能使用 TLS，需先明确安全决策再调整传输限制。

## 单位与字段转换
- `trade_date`: `YYYYMMDD` → `DATE` (Asia/Shanghai 交易日期)；`ts_code` 保留 `.SH/.SZ` 后缀。
- DTO 按官方字段名从返回的动态列映射；`stock_basic` 必需身份字段为 `ts_code,symbol,name,area,industry,cnspell,market,list_date`，可空字段为 `fullname,enname,exchange,curr_type,list_status,delist_date,is_hs,act_name,act_ent_type`。领域映射另外要求 `exchange` 和 `list_status`；状态只接受当前合同 `L/D/P`，遇到官方列出的 `G/UN` 显式报 `DATA_INCOMPLETE`，不能丢行或伪装为受支持状态。
- `trade_cal` 必需 `exchange,cal_date,is_open`；`pretrade_date` 可空。`is_open` 接收官方整数 `0/1`（也兼容字符串形式），其他值拒绝。
- `daily` 领域映射必需 `ts_code,trade_date,open,high,low,close,vol,amount`；未使用的 `pre_close/change/pct_chg/ah_vol/ah_amount` 保留为可空 DTO 字段。停牌日无日线行。
- `daily.amount`: 保留原 JSON 十进制文本，精确执行千元 × 1000 得到 `amount_yuan`。`daily.vol` 单位手，不要当股；如转股数需 ×100，但不强依赖。
- `adj_factor` 的 `ts_code,trade_date,adj_factor` 必需，因子严格 >0。
- `suspend_d` 必需 `ts_code,trade_date,suspend_type`，`suspend_timing` 可空；`S/R` 保持停复牌事件语义，不推导每日状态。`stk_limit` 必需 `trade_date,ts_code,up_limit,down_limit`，`pre_close,asset_type,exchange` 可空。两者当前仅映射和校验，不新增持久化表或市场领域模型。
- `adj_factor`: 严格 >0；价格所有小数来自原接口，无隐式舍入。
- 使用 `types.Decimal` 保留规范十进制值；单位转换、DECIMAL 范围检查和数据库文本读写不经过 `float64`。MySQL 约定金额 DECIMAL(24,4)、价格 DECIMAL(20,6)、因子 DECIMAL(24,10)，精度/scale 超界直接报错，不允许静默舍入。只有公式/统计等数值计算边界可显式调用 `Float64Checked`；精确值不从计算结果回写。

## 配额管理
- 第一版单进程速率控制：保守 global 100 req/min，`stock_basic` 专用 40 req/min（低于官方档位，均配置化），与 API 实际返回限制协调调整；避免误以为“所有接口 200/min”。
- 429/流控类响应指数退避+抖动，限尝试次数；权限 2002 或类似拒绝立刻停止；HTTP 5xx 和临时网络错误可重试；全部日志脱敏。
- 必须记录 `api_name`、request_id、接口时延、返回行数、业务日期、错误分类；**不能**记录 Token。

## 初始回填 / 增量同步
1. 首先拉交易日历、全部历史证券状态并维护历史身份。
2. 以交易日期批量拉取 `daily`、`adj_factor`，按日期逐日处理、存储，单日结果标记齐备之后可进入下游；`daily_basic` 可作为非关键扩展。
3. 每日按数据齐备检查触发，不仅依赖 `18:30` 固定时间；T 日复权因子存在盘前维护特性，需校验 T 日数据存在。
4. `sync_job` 管理游标 / 状态、计数、数据校验、重试；断点重跑 `UPSERT`；任何异常日期不得默默跳过。
5. `daily` 单日无数据不等于一定停牌；对上市股票根据日历+停牌公告区分，并保存缺失原因。需要设置“预期股票范围”，按上市/退市日期，不与当前上市集合比较。

## 实测矩阵与凭据
- 无 Token：使用 HTTP fixture Mock 通过本地测试，不应阻塞纯策略任务。
- 有 Token：只在人工授权的真实集成中按 `stock_basic,trade_cal,daily,adj_factor,suspend_d,stk_limit,namechange` 验权并把结果记录 `handoffs/`。禁止输出 Token/完整请求体。
- 权限不足时跳过可选接口；若影响 ST 严格模式则拒绝发布正式回测结果。

## 数据版本与可见性
保存拉取批次、`fetched_at`、校验时间、原始响应 hash 和修订编号。若源数据随后修改，旧的选股结果用原先 snapshot hash 复现，而非直接说“不可能复现”。
