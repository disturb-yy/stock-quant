# P00 阶段门禁与证据

- 本阶段 tickets 与 handoff 文件：`P00-01` — `handoffs/completed/P00-01.md`; `P00-02` — `handoffs/completed/P00-02.md`; `P00-03` — `handoffs/completed/P00-03.md`。三个本地实现均经独立审核；P00-03 的 hosted CI/分支保护验收仍为 BLOCKED。
- 验证环境：Linux amd64；Go 1.27.0；Python 3.12.3；未安装/启动 MySQL；未配置 Tushare token；目标目录无 Git 元数据。
- 关键端到端命令与真实结果：`make doctor` 通过 mock 配置检查；`make run` 输出 `{"status":"ok","scope":"process"}`，只证明进程级命令可运行；`make check` 通过 gofmt、secret scan、Go/Python/tool/config/domain tests、vet 与 build。P00-03 的临时 Go fixture 证明格式错误和失败单测会阻断质量门。
- 覆盖矩阵：数据 — 无数据库/外部市场数据依赖；算法 — 本阶段无业务算法；异常 — Go 失败单测、格式错误和包发现失败路径均验证阻断；安全 — secret 值脱敏与不可读文件失败关闭通过；性能 — 不适用，本阶段无业务负载。
- 上一阶段契约兼容：本项目无前序实现阶段；源指南的 schema 与业务契约未被修改。P00 建立 Go 应用骨架与跨阶段质量门。
- 新增 ADR 与已知限制：无 ADR。MySQL 未运行；真实 Tushare 权限未验证；没有 GitHub 仓库/remote，Hosted Actions 执行和服务端分支保护未验证。P00-03 要求失败检查阻断合并，故此未验证项阻止本阶段通过。
- Gate: **BLOCKED**。本地工程基线实现和审查通过，但 P00-03 阶段验收要求 hosted CI 能阻断合并；当前无法证明该条件，不能把本阶段标为 PASS。
- 审核人及日期：`p00_01_review` 独立审核 P00-03 本地实现（2026-10-09）；阶段门禁状态待取得 GitHub hosted CI 与 branch protection 证据后复核。
- 下一阶段可依赖的稳定接口、表与 fixture：`cmd/stockquant` 进程级 health CLI；`internal/app.HealthService`；Go/Python 版本和本地 Make 目标；`.github/workflows/ci.yml` + `tools/check.sh`。当前没有生产数据库表或市场数据 fixture。P01 暂不启动，遵循 P00 阶段门禁。
