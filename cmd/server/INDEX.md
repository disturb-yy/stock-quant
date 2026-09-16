# INDEX.md

## Package

`cmd/server`

## Role

程序启动入口与依赖组装。

## Typical Flow

```text
config
→ infrastructure clients
→ repositories/providers
→ application services
→ handlers
→ router
→ server
```

修改业务行为时不要优先改本 package。

## Current Contents

- `main.go`：从配置构造日志、MySQL demo store、市场概览、行业表现、市场信号、股票排行榜、股票详情、股票研究型日线和股票财务查询服务，执行 migration；development 环境装配 demo-status，创建 HTTP Server 并监听 `:8357`。
- `migration.go`：在 Composition Root 调用 Infrastructure store 的五项 migration 入口。
- `router.go`：创建 Gin Router，按“请求日志 → Recovery”顺序安装中间件，创建 `/api/v1` 路由分组并装配 OpenAPI、健康路由、市场概览、行业表现、市场信号、股票排行榜、股票详情、股票研究型日线、股票财务、开发 demo-status、分页校验和统一错误处理。
