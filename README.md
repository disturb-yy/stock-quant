# Stock Quant

Stock Quant 是面向 A 股、以日线为频率的量化研究平台。平台使用 Go 负责流程编排和 API，使用 MySQL 保存行情数据与运行记录，并通过受控的 Python 子进程计算原始因子。平台支持研究和模拟回测，不执行真实交易。

## 项目状态

项目按 [`tickets/ORDER.md`](tickets/ORDER.md) 中的顺序逐项实施。P00 工程基线、P01 契约/领域接口和 P02 MySQL/持久化均已通过阶段门；P03-01（Tushare HTTP 封装与动态字段解析）、P03-02（限流、重试、日志脱敏）和 P03-03（API DTO、单位映射与精确十进制存储）已验收。当前任务为 P03-04：本地权限分级和 offline/mock 诊断已实现，真实 Token 权限验收因缺少本地 Token 与单独的真实请求授权而处于 `BLOCKED`，项目未进入 P04。阶段证据见 [`handoffs/completed/P00-gate.md`](handoffs/completed/P00-gate.md)、[`handoffs/completed/P01-gate.md`](handoffs/completed/P01-gate.md)、[`handoffs/completed/P02-gate.md`](handoffs/completed/P02-gate.md)、[`handoffs/completed/P03-01.md`](handoffs/completed/P03-01.md)、[`handoffs/completed/P03-02.md`](handoffs/completed/P03-02.md) 和 [`handoffs/completed/P03-03.md`](handoffs/completed/P03-03.md)。已批准的规格见 [`docs/`](docs/)，已完成工作的证据记录在 `handoffs/completed/`。源指南中的示例用于演示 Go/Python JSON 交互，不是生产运行器或平台，也不作为本仓库的实现证据。

## 项目范围

- 覆盖中国大陆上交所和深交所的日线行情，输入窗口为 61 个交易日。
- 固定策略为 `momentum_v1`：60 日动量（40%）、20 日动量（30%）、成交活跃度（20%）和低波动率（10%）。
- Go 负责筛选、横截面评分和排序；Python 从只读快照计算原始因子。
- 在 T 日生成信号，在 T+1 日模拟执行，并明确展示回测限制。
- 当前提供 MySQL 8.4 版本化迁移命令和股票、交易日历、日线、复权因子仓储；Go HTTP API 和轻量级仪表盘将在后续票据实现。

V1 不包含真实券商委托、日内数据、机器学习预测、分布式服务或任意不可信 Python 插件。完整边界见 [`docs/01-requirements.md`](docs/01-requirements.md)，固定计算规则见 [`docs/03-strategy-spec.md`](docs/03-strategy-spec.md)。

## 技术基线

| 组件 | 版本线 | 用途 |
|---|---:|---|
| Go | 1.27.x | 后端与命令行工具 |
| Python | 3.12.x | 策略执行进程与测试 |
| MySQL | 8.4 LTS | 持久化存储 |
| Node.js | 24.x LTS | 仪表盘工具链 |
| 时区 | Asia/Shanghai | 交易日期与运行时 |

补丁版本可在对应版本线内升级。Go 模块声明最低语言/工具链版本。安装说明和 WSL 说明见 [`docs/environment-setup.md`](docs/environment-setup.md)。

## 本地开发

将 `.env.example` 复制为 `.env`，只填写当前运行模式所需的值。现有 Make 目标不会自动加载 `.env`；请在终端中导出变量，或使用本地密钥管理工具。不要提交 `.env`，也不要把真实 Tushare Token 写进命令历史。数据提供方默认使用本地固定样本/模拟模式；访问真实数据提供方需要显式配置 Token。

```bash
make setup-python
make doctor
make test
make check
```

普通测试不需要 Tushare Token；设置 `MYSQL_TEST_DSN` 后会在隔离数据库中执行真实 MySQL 迁移集成测试。GitHub Actions 使用临时 MySQL 8.4 服务。以下命令可用于构建、运行和显式迁移：

```bash
make build
make run
APP_ENV=development make migrate-up
APP_ENV=development make migrate-down
```

当前 `health` 命令返回 `{"status":"ok","scope":"process"}`，只表示进程级存活；它不检查 MySQL、Tushare 或 HTTP 是否就绪。迁移不会随应用启动自动运行；`migrate-down` 只允许在 `APP_ENV=development` 或 `APP_ENV=test` 时显式调用。

`make check` 是 GitHub Actions 质量门的本地对应检查。测试位置和证据要求见 [`docs/testing.md`](docs/testing.md)。

## 文档导航

- [`AGENTS.md`](AGENTS.md)：实现边界与仓库规约。
- [`INDEX.md`](INDEX.md)：设计、契约、阶段和任务导航。
- [`tickets/ORDER.md`](tickets/ORDER.md)：任务执行顺序。
- [`docs/12-implementation-runbook.md`](docs/12-implementation-runbook.md)：任务执行流程与阶段门禁。
- [`docs/11-quality-gates.md`](docs/11-quality-gates.md)：验证证据与测试用例。
- [`handoffs/templates/ticket-completion.md`](handoffs/templates/ticket-completion.md)：任务交付记录模板。
