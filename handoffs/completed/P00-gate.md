# P00 阶段门禁与证据

- 本阶段任务与交接文件：`P00-01` — `handoffs/completed/P00-01.md`; `P00-02` — `handoffs/completed/P00-02.md`; `P00-03` — `handoffs/completed/P00-03.md`。本地实现均经独立审核；P00-03 的 GitHub CI 与分支保护已完成验证。
- 验证环境：Linux amd64；Go 1.27.0；Python 3.12.3；Git 2.43.0；MySQL 未安装/启动；未配置真实 Tushare Token。
- 关键端到端命令与真实结果：`make doctor` 通过模拟配置检查；`make run` 输出 `{"status":"ok","scope":"process"}`，只证明进程级命令可运行；`make check` 通过 gofmt、secret scan、Go/Python/工具/配置/领域边界测试、vet 与 build。P00-03 的临时 Go 样例证明格式错误和失败单测会阻断质量门。
- 托管 CI 与分支保护：首次提交 `7677452144557ad9c54d77e8a1ba56bb3f4ec33f` 的 [GitHub Actions 运行](https://github.com/disturb-yy/stock-quant/actions/runs/37888345242) 成功，`Go and Python quality gates` 检查通过。GitHub API 回读确认 `main` 要求 PR、必需检查为 `Go and Python quality gates`、要求分支保持最新、管理员受规则约束，并禁止强推和删除分支。PR 审批数为 0，即必须走 PR 且 CI 通过，但不强制人工审批。
- 覆盖矩阵：数据 — 无数据库或外部行情依赖；算法 — 本阶段无业务算法；异常 — Go 失败单测、格式错误和包发现失败路径均验证阻断；安全 — 凭据值脱敏与不可读文件失败关闭通过；性能 — 不适用，本阶段无业务负载。
- 上一阶段契约兼容：本项目无前序实现阶段；源指南中的 schema 与业务契约未被修改。P00 建立 Go 应用骨架与跨阶段质量门。
- 新增 ADR 与已知限制：无 ADR。MySQL 未运行，真实 Tushare 权限未验证；二者不属于 P00 的模拟模式基线验收。
- Gate: **PASS**。P00 三个任务的实现、本地质量门、GitHub Actions 工作流和 `main` 分支保护均有可复核证据。
- 审核人及日期：`p00_01_review` 于 2026-10-09 独立审核 P00-01、P00-02、P00-03 实现及本阶段门禁证据；文档复核 stable_id 为 `94a4a6a9b948d3b7205e00ba9682951173b15cce94027d50161fd7be29ccf470`，结论 CLEARED。
- 下一阶段可依赖的稳定接口、表与样例：`cmd/stockquant` 进程级健康命令；`internal/app.HealthService`；Go/Python 版本和本地 Make 目标；`.github/workflows/ci.yml` 与 `tools/check.sh`。当前没有生产数据库表或市场数据样例。下一任务为 P01-01。
