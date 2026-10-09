# 11 — 测试策略与质量门禁（统一）

## 四级验证
1. 快速单测：公式/边界/过滤/排名/无未来函数；纯函数每次改动必跑。Go：`go test ./...`; Python：`python -m unittest discover -s python/tests`。
2. 契约/集成测试：httptest 模拟 Tushare；`DATA_INCOMPLETE`、缺字段、流控、权限；MySQL 临时容器的迁移、事务、幂等；subprocess 协议 JSON Schema。
3. 端到端（固定 fixture）：交易日历 -> 历史快照 -> Go因子 -> 过滤归一化 -> Top20 -> SQL 结果 -> API -> 页面。
4. 正式数据试运行：在有人工提供 Token 且权限已核验时执行；数据源真实调用仅为人工授权检查，CI 不依赖该 Token。

## 主要用例矩阵
| Case | 输入/刺激 | 预期 | 阶段 |
|---|---|---|---|
| DATA-01 | `daily.amount=12500.5` | 数据库 `amount_yuan=12500500` | P03-P04 |
| DATA-02 | `data.fields` 交换列位置 | 数据解析不受影响 | P03 |
| DATA-03 | 数据只有60/61日期 | 不产生合格 snapshot | P05 |
| DATA-04 | 当前未上市而历史上存在股票 | 历史 universe 有该证券 | P04-P05 |
| DATA-05 | `stock_st` 403/权限不足 | 不使用此接口作为必需依赖，严格回测被阻断或取得其它可信来源 | P03-P10 |
| FCT-01 | 61日平价/恒定成交额 | M60=M20=activity=vol=0 | P06 |
| FCT-02 | 日涨1%连续60天 | M60≈1.01^60-1 | P06 |
| FCT-03 | 近期20日成交额为之前两倍 | Activity=ln(2) | P06 |
| FCT-04 | 30日前拆股、close减半/adj翻倍 | 动量不被误判下跌50% | P06 |
| FCT-05 | 含 NaN/Inf/0 价格 | `INVALID_FACTOR_INPUT` | P06 |
| FLT-01 | 日均成交额=30m | 通过（含边界） | P07 |
| FLT-02 | `close==MA20` 或 `momentum60==0` | 排除 | P07 |
| FLT-03 | 不明历史 ST、strict | BLOCKED，不宣称正式策略结果 | P07/P10 |
| SCR-01 | 4项得分 [80,60,90,70] | total=75.0 | P07 |
| SCR-02 | 同因子 [10,20,20,40] | percentile [0,50,50,100] | P07 |
| SCR-03 | 低波原值 [.1,.2,.3] | percentile [100,50,0] | P07 |
| SCR-04 | 单个候选股票 | 每项 50，总分 50 | P07 |
| RUN-01 | Python stdout 混入日志 | 报协议错误 | P08 |
| RUN-02 | Python 超时/退出码非0 | 清理子进程，任务失败 | P08 |
| RUN-03 | Go/Python 固定 fixture | 原始因子一致，容差≤1e-10 | P08 |
| API-01 | 重复创建同一 run_key | 唯一结果，不重复任务 | P09 |
| API-02 | 任务 RUNNING 未完全保存 | 查询不能看到 SUCCESS/半结果 | P09 |
| BT-01 | T日筛选 T+1 开盘买入 | 不提前成交 | P10 |
| BT-02 | T+1 开盘涨停且无法买入 | 拒单并保留现金 | P10 |
| BT-03 | 买入同日卖出 | 拒绝，T+1 | P10 |
| BT-04 | 给定 T+1 之后的数据变化 | T日信号不变（同快照版本） | P10 |
| BT-05 | 企业行为数据未覆盖 | 标 research-only 或 BLOCKED | P10 |
| UI-01 | loading/empty/failed/blocked | 页面不显示虚假的成功或持仓 | P11 |

## 每个 ticket 的验收门禁
- Code: 格式、静态检查、单测通过。
- Contract: 对外接口/表/schema 变动已改契约和 fixtures。
- Evidence: 保存真实命令行摘要、输入样例、输出、风险，参 `handoffs/templates/ticket-completion.md`。
- No regression: 旧测试全部通过；新增 bug 先写失败回归测试。
- Data: 有 `as_of`, config hash, snapshot hash, version。

## 测试 fixture 策略
小规模可手算：1/2/4 股票、20/40/61交易日、停牌间断、拆股复权、重复 TS code；不允许把随机结果当黄金值。可重复的冻结 fixtures 应有 JSON 文件、算法/数据版本及计算说明。

## 严格阻断场景
缺交易日历、缺 61 日价、价格/因子异常、ST 不可核实（严格模式）、数据源返回字段不全、回测企业行为未支持、真实凭据缺失且 ticket 明确要求线上验权。BLOCKED 非 PASS。
