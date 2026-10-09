# 12 — 从零到可使用版本的具体运行手册

## 主控制规则
阶段推进以 **ticket→代码→真实测试→证据文件→下一 ticket** 为单位。禁止依靠前一个 AI 的聊天上下文传递信息；每张票在 `handoffs/completed/<ticket-id>.md` 落盘。人类只需检查 DoD 与阻断项。

## 正式环境前置
Go（示例最小Go 1.23；正式版按 P00 锁版本）、MySQL 8、Python 3.12/3.13（视科学计算依赖选择）、Node LTS、Docker(可选)、Tushare Token；Python 本地 `.venv` 由 `uv`/`pip` 管理。不要把 Token 填 YAML 送 Git。

## 一次 ticket 的标准流程
1. `git status --short` 确保状态清楚。创建 feature 分支/可选 worktree。复制 `prompts/TICKET-EXECUTION.md`，将 `<TICKET_ID>` 替换为下一任务。
2. 读取其 `Prerequisites` 和相关 docs/contract/上游 handoff。
3. AI 先给出影响面、最小实现、测试列表；先写失败单测（包括错误路径）。
4. 实现 Go package/handler/SQL/Python。接口改动必须修改 schema/example/API 文档。严禁越界做另一 ticket 功能。
5. `gofmt -w ...`, `go test ./...`, `go vet ./...`；Python tests、db migrate 和 E2E 视涉及范围加跑。
6. 把真实测试证据写到 `handoffs/completed/<ID>.md`，列出 git diff / commit id、运行命令、PASS/FAIL/BLOCKED、核心返回、遗留风险。
7. 审核 DoD、`tickets/ORDER.md` 标记完成（若项目中跟踪进度），通过后再下发下一票。

## 阶段门禁
- P00 => mock-only 环境可启动，依赖版本锁定，测试脚本稳定。
- P01 => request/result JSON Schema、Go/Python 类型、非法协议测试。
- P02 => MySQL迁移 up/down、复合唯一索引与事务幂等。
- P03 => Mock Tushare 可解析动态 fields，限流/权限错误。
- P04 => 指定交易日可幂等落库；历史股票池无幸存者偏差。
- P05 => 交易日历与61条完整校验、ST 状态可信度、snapshot_hash 可复现。
- P06 => 因子全部数学测试、窗口及复权测试通过。
- P07 => 过滤、百分位、评分、稳定排序、Top N 与拒绝原因。
- P08 => Python worker 与 Go 因子一致；超时、坏输出和安全策略验证。
- P09 => 使用 mock 数据通过 HTTP 运行/查询，持久化一致且任务幂等。
- P10 => 手工账本对齐、T+1/涨停/成本/幸存者偏差、结果有可信度标签。
- P11 => 5页可展示任务、数据质量与结果；错误状态具备可见反馈。
- P12 => 部署、备份/恢复、日志脱敏、端到端可重复。

## 可重复的 smoke 路径（生产代码完成后）
1. 初始化 DB 和迁移。
2. 无 Token 运行 fixture 模式：导入两/四只模拟股票 × 61 日，完成单日 data snapshot。
3. `stockquant screen ...` -> JSON/SQL 验证四个因子与 score。
4. HTTP 提交重复 `screen-run` -> 同 run_key，不重复记录。
5. `PythonRunner` 跑同一快照 -> 返回值在容差内与 `GoRunner` 一致。
6. UI 读取同一 run_id，显示分数，查看失败或 blocked 状态。
7. 回测仅在 ST/status 公司行为数据可用时运行 strict，否则探索模式明显标记限制。

## 质量问题处理
测试失败：保留日志/fixture、建 bug ticket、修复后增加回归用例；源数据不足：标 BLOCKED，不伪造成功；方案冲突：新增 `docs/14-decisions-risks.md` ADR 条目，获批准前不改变冻结的因子权重/窗口。
