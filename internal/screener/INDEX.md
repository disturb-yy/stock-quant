# INDEX.md

## Package

`internal/screener`

## Role

SCR-001 量化选股执行 API 的 Application、Interface 和字段契约边界。

## Current Contents

- `domain/`：canonical field registry、ScreenerSpec 校验、AND 条件、排序和 Top N 规则。
- `service.go`：执行快照读取与 Domain 编排。
- `handler.go`：`POST /api/v1/screeners/run` HTTP 适配及错误映射。
- `infrastructure/`：复用既有 MySQL 读模型的参数化快照查询。
