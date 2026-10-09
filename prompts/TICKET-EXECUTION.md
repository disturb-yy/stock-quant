# 单 ticket 执行提示词

实现 ticket：`<TICKET_ID>`。

1. 阅读全局 `AGENTS.md`、`INDEX.md` 和这张 ticket 的完整正文、`docs/` 引用及依赖 ticket 的 `handoffs/completed/`；如果代码目录的 `AGENTS.md`/`INDEX.md` 已存在，也要阅读。
2. 列出实现计划（文件、接口、可测试的行为与风险），不要同时实施下一张 ticket。
3. 先建立反例/边界测试，然后写最小实现；必要的迁移/契约/配置改动先于生产代码修改。
4. 提供真实验证：`go test ./...`、`go vet ./...`、相关 Python tests/SQL/HTTP fixture；报告 PASS/FAIL/BLOCKED。不允许以“预计通过”替代真实运行。
5. 完成后创建 `handoffs/completed/<TICKET_ID>.md`，只通过交付文件将事实交给下一个 AI，不依赖上下文记忆。未通过则说明阻断项，不标记 DONE。
