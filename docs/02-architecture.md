# 02 — 架构、依赖与代码规范

## 逻辑架构
```text
React UI -> Go HTTP handlers -> Application UseCases -> {Market, Factor, Strategy, Screening, Backtest}
                                  |                          |
                                  v                          v
                         ports/interfaces              Pure calculation
                                  |
                         Infrastructure adapters
                         | MySQL | Tushare | Python subprocess
```
部署初期 Go API+worker 可同进程或同一个二进制的子命令，不上消息队列。数据库 MySQL 8。Python subprocess 以 JSON Envelope 传入数据引用，按可配置并发度调用；本地演示 JSON 内联，但生产必须验证输入文件路径与 schema。

## 建议生产项目目录
```text
cmd/stockquant/main.go       # Composition Root and CLI (P00 currently exposes health)
internal/app/               # UseCase，注入接口，跨领域编排
internal/market/            # stock, trade_calendar, raw bar, status
internal/factor/            # 计算器/数值校验/因子版本
internal/strategy/momentum/ # v1 策略定义与权重
internal/screening/         # 过滤/截面排名/运行结果
internal/backtest/          # order, fill, position, equity, metrics
internal/infrastructure/tushare/
internal/infrastructure/mysql/
internal/infrastructure/runner/
internal/infrastructure/scheduler/
api/http/                  # DTO 与 handler
web/                       # React + TS + Vite
python/worker.py
python/strategies/
configs/strategies/
contracts/
migrations/
```
各 package 自身写 `AGENTS.md` + `INDEX.md`，新增模块 README 可选。Root `INDEX.md` 提供重要接口和跳转。

## 依赖方向
- `market` 不 import `screening`/`backtest`；`factor` 只接收纯 Bars；`strategy` 无 SQL/HTTP。
- `screening` 接受 data snapshot 和 factor result，不直接连接 Tushare；回测重用 `screening` 与因子模块。
- `app` 负责用例编排；跨领域通过清晰 DTO/ports，不允许 repository 跨域调用另一个 repository 的未公开实现。
- 可利用 `internal/shared/types` 放通用日期、金额、枚举，但严禁把领域模型堆进 shared。
- Composition Root 在 `cmd/stockquant/main.go` 完成 wiring，领域不读全局配置。

## 核心接口（Go 意图，不是最终可编译实现）
当前最小接口放在所属 domain 的 `ports` package：market 提供快照读取与交易日历查询，factor 提供协议 v1 `FactorRunner`，screening 提供 `ScreeningStore.SaveRun`。`FactorRunner` 位于 worker 协议边界，使用已冻结的 `contracts` DTO；因子计算 domain 仍只依赖纯 bars。`ScreeningStore` 目前只保存运行身份元数据，不包含尚未实现的过滤、候选或评分结果。

真实 adapter 可用后，由 `cmd/stockquant` 组合根选择并注入实现；领域包不维护全局可变注册表。受信任 Python 策略注册表属于 P08-04，不在 P01-03 提前实现。

所有接口都应保留输入日期、版本与批次身份；接口保持精简，避免为了“未来会扩展”引入空接口。

## 错误分类
稳定 code：`INVALID_ARGUMENT / DATA_INCOMPLETE / PERMISSION_DENIED / RATE_LIMITED / UPSTREAM_UNAVAILABLE / STRATEGY_FAILED / TIMEOUT / CANCELLED / INVALID_WORKER_RESPONSE / NOT_FOUND / CONFLICT / DATA_UNTRUSTED / INVALID_FACTOR_INPUT / INTERNAL`。跨层封装保留 cause 供 `errors.Is`/`errors.As` 诊断；API/CLI 对外消息由边界层安全映射，不得直接泄露 Token、DSN、绝对路径等敏感字段。可重试只对速率限制、网络中断与可恢复 5xx，禁止重试权限错误与公式错误。

## Go 规范
使用 context、显式错误、表格驱动测试、事务边界在 repository/usecase，限流集中；数据字段单位写在名称如 `AmountYuan`。迁移 schema version 化。运行时禁止因子计算通过 `time.Now()` 查询别的日期。
