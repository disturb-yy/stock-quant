# AGENTS.md

## Scope

本目录是开发 fixture seed 的 CLI Composition Root。

## Rule

只负责加载配置、创建 Infrastructure、运行 migration/seed 和输出结果；不在 CLI 中写 SQL 或业务规则。
