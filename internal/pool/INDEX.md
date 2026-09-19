# INDEX.md

## Package

`internal/pool`

## Role

POL-001 股票池创建、搜索、列表与概览读取，以及 POL-002 成员管理的 Application、Interface 边界。

## Current Contents

- `domain/`：股票池身份、固定手工来源和名称/描述校验。
- `service.go`：创建、搜索分页、详情、来源与基础画像摘要和成员读写用例编排。
- `handler.go`：股票池、`GET /api/v1/stock-pools/{id}/summary`、成员读写 HTTP 适配及统一错误映射。
- `infrastructure/`：`t_stock_pool`、`t_stock_pool_member` 的 MySQL migration、持久化读写和幂等 Demo Seed。
