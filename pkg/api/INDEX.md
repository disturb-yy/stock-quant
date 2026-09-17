# INDEX.md

## Package

`pkg/api`

## Role

跨领域 API 契约、错误模型和分页约定。

## Current Contents

- `response.go`：统一成功 envelope 和分页成功响应 DTO。
- `error.go`：稳定错误码和错误 DTO。
- `pagination.go`：分页请求/响应 DTO、默认值和查询参数校验。
- `openapi.go`：API v1 实际输出的 OpenAPI JSON 文档构造，包含市场概览、行业表现、市场信号、股票排行榜、股票详情、股票财务、股票估值响应、分页及来源 Schema。
