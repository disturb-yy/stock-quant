# INDEX.md — AI 阅读导航

| 主题 | 文件 | 使用时机 |
|---|---|---|
| 项目实施规约 | `AGENTS.md`, `README.md` | 每次开始前 |
| 本地环境/版本 | `docs/environment-setup.md`, `Makefile`, `.env.example` | 首次设置或环境问题 |
| 测试与 CI | `docs/testing.md`, `.github/workflows/ci.yml`, `tools/check.sh` | 本地/合并前质量验证 |
| 跨语言 JSON 协议与 DTO | `contracts/`, `python/stockquant/protocol.py` | P01/P08 请求与响应校验、编码和解码 |
| 通用交易日期与精确数值 | `internal/shared/types/` | 日期格式互转、精确十进制 JSON/SQL、DECIMAL 校验、显式 float64 计算边界 |
| 稳定错误分类 | `internal/shared/apperror/` | 跨层错误码与 cause 链 |
| 市场/因子/选股/回测 ports | `internal/market/ports/`, `internal/factor/ports/`, `internal/screening/ports/`, `internal/backtest/ports/` | 应用与行情、同步任务、因子执行及运行结果存储的接口边界 |
| Go CLI 与 Composition Root | `cmd/stockquant/`, `internal/app/` | 命令入口、用例组装 |
| 市场与策略领域 | `internal/market/domain/`, `internal/factor/domain/`, `internal/strategy/momentum/domain/` | P02 行情实体、P06/P07 因子与策略 |
| 选股与回测领域 | `internal/screening/domain/`, `internal/backtest/domain/` | P06/P07/P10 |
| 技术适配器 | `internal/infrastructure/` | MySQL 精确 DECIMAL 仓储/迁移、Tushare HTTP/DTO/映射/限流/重试、Python Runner、scheduler |
| 项目目标及边界 | `docs/01-requirements.md` | 每次开始前 |
| Go/Python 体系结构 | `docs/02-architecture.md` | package、依赖设计 |
| 四因子精确定义 | `docs/03-strategy-spec.md` | 因子、过滤、评分、回测 |
| Tushare 接口/积分/质量 | `docs/04-tushare.md` | Tushare/行情任务 |
| SQL 与仓储 | `docs/05-database.md`, `db/migrations/*`, `internal/infrastructure/mysql/` | P02/P04/P05/P09；任务及筛选/回测运行持久化见 `run_repositories.go` |
| REST API | `docs/06-api.md` | P09/P11 |
| 跨语言运行契约 | `docs/07-python-protocol.md`, `contracts/*` | P01/P08 |
| 回测交易语义 | `docs/08-backtest.md` | P10 |
| 低保真前端 | `docs/09-dashboard.md` | P11 |
| 任务/运维 | `docs/10-operations.md` | P09/P12 |
| 测试与质量门禁 | `docs/11-quality-gates.md` | 每个 ticket |
| 工程实施节奏 | `docs/12-implementation-runbook.md` | 全流程 |
| 安全与权限 | `docs/13-security.md` | P03/P08/P12 |
| 确认事项与风险 | `docs/14-decisions-risks.md` | 有分歧时 |
| 参考链接 | `docs/REFERENCES.md` | API 变更核验 |
| 阶段图 | `phases/README.md` | 阶段交接 |
| 原子任务索引 | `tickets/ORDER.md` | 每次分派一个任务 |
| AI 任务提示词 | `prompts/AI-START.md` | 拷贝给新 AI |
| 环境检查 | `tools/env-check.sh`, `tools/env-check-test.sh` | P00 本地环境 |

说明：源实施指南中的 `examples/contract-demo/` 是跨语言演示，不属于本项目生产实现。生产 Go 目录在 `docs/02-architecture.md` 中规划。
