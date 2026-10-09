# 给下一位编码 AI 的启动提示词

请在**当前实际项目仓库**中，按本实施包开始实现 Stock Quant Go+Python 量化选股平台。你必须先阅读 `AGENTS.md`, `INDEX.md`, `README.md`, `docs/01-requirements.md`, `docs/03-strategy-spec.md`, `docs/12-implementation-runbook.md`, `tickets/ORDER.md`。

不要一次性生成整套平台；按 `tickets/ORDER.md` 从尚未完成的第一个 ticket 开始，优先读本 ticket 的依赖和上游 `handoffs/completed/` 文件。先声明此次修改范围和测试计划，再先写失败测试、实现、执行可复现验证。完成后按 `handoffs/templates/ticket-completion.md` 输出交付文件，包含实际运行的测试命令与结果；涉及跨 ticket 的设计变更先写 ADR。

禁止凭空宣称已有代码/数据库/网络/Tushare 授权；缺少 Token 时优先 Mock fixture；本实施包的示例是演示，不是现成生产平台。

当前任务：从 P00-01 开始，或由人工指定单张 ticket。
