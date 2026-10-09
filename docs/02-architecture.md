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
```go
type SnapshotRepository interface {
    Load(ctx context.Context, asOf TradingDate, lookback int) (Snapshot, error)
}
type FactorCalculator interface {
    Calculate(bars []Bar) (RawFactors, error)
}
type FactorRunner interface {
    Compute(ctx context.Context, req FactorRequest) (FactorResult, error)
}
type ScreeningEngine interface {
    Select(ctx context.Context, req ScreenRequest) (ScreenResult, error)
}
type ScreeningRepository interface {
    Save(ctx context.Context, result ScreenResult) error
}
```
所有接口必须具有对输入日期、版本与批次的不可变表示；P01 冻结 `contracts` 后再落地结构体。接口较小，避免为了“未来会扩展”引入十余个空接口。

## 错误分类
`INVALID_ARGUMENT / DATA_INCOMPLETE / PERMISSION_DENIED / RATE_LIMITED / UPSTREAM_UNAVAILABLE / STRATEGY_FAILED / TIMEOUT / INTERNAL`。跨层封装保留 `cause`，对外去除 Token/DSN/绝对路径等敏感字段；可重试只对速率限制、网络中断与可恢复 5xx，禁止重试权限错误与公式错误。

## Go 规范
使用 context、显式错误、表格驱动测试、事务边界在 repository/usecase，限流集中；数据字段单位写在名称如 `AmountYuan`。迁移 schema version 化。运行时禁止因子计算通过 `time.Now()` 查询别的日期。
