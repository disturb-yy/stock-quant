# AGENTS.md

## Scope

本目录是行情领域边界，负责日线行情和市场概览查询。

## Rule

`domain` 只放日线、指数快照和市场宽度等业务概念；Provider 选择只返回明确的 demo/real/fallback 语义，不得把本地 fixture 冒充真实外部 Provider。

市场概览的 SQL 和数据库字段只能留在 `infrastructure`；Application Service 负责完整性校验和响应聚合，Handler 只负责 HTTP 适配。
