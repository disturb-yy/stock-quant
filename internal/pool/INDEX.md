# INDEX.md

## Package

`internal/pool`

## Role

POL-001 股票池创建、搜索、列表与概览读取的 Application、Interface 边界。

## Current Contents

- `domain/`：股票池身份、固定手工来源和名称/描述校验。
- `service.go`：创建、搜索分页和详情用例编排。
- `handler.go`：`POST/GET /api/v1/stock-pools`、`GET /api/v1/stock-pools/{id}` HTTP 适配及统一错误映射。
- `infrastructure/`：`t_stock_pool` 的 MySQL migration、持久化读取和幂等 Demo Seed。
