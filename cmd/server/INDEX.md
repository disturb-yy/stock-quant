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

- `main.go`：从配置构造日志、MySQL store、可选 Tushare 同步、市场概览、行业表现、市场信号、股票排行榜、股票详情、股票研究型日线、股票财务、股票估值、保存方案、股票池和 Research 项目服务，执行多 Infrastructure migration；Demo 模式装配 demo-status，创建 HTTP Server 并监听 `:8357`。
- `migration.go`：在 Composition Root 按顺序调用多个 Infrastructure store 的 migration 入口。
- `router.go`：创建 Gin Router，按“请求日志 → Recovery”顺序安装中间件，创建 `/api/v1` 路由分组并装配 OpenAPI、健康路由、市场概览、行业表现、市场信号、股票排行榜、股票详情、股票研究型日线、股票财务、股票估值、选股执行、保存方案、股票池及来源/画像摘要、Research 项目、开发 demo-status、分页校验和统一错误处理。
