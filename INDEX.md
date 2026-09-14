# Project Index

## Purpose

本文件是项目的全局索引地图。

它回答：

- 项目有哪些主要模块？
- 一个需求应该从哪里开始找？
- 哪些 package 是业务领域？
- 哪些 package 是基础能力？

具体编码限制请阅读对应 `AGENTS.md`。

## Entry Points

| 路径 | 职责 |
|---|---|
| `cmd/server` | 服务启动、依赖注入、Composition Root |
| `build/chart` | Helm 部署包装与运行时配置 |
| `internal/stock` | 股票基础信息领域 |
| `internal/market` | 行情数据领域 |
| `internal/analysis` | 股票分析领域 |
| `internal/auth` | 身份认证与授权领域 |
| `internal/health` | 跨领域健康 HTTP 接口适配 |
| `internal/migration` | 数据库 migration 命名与执行边界 |
| `pkg/config` | 配置加载 |
| `pkg/logger` | 通用日志能力 |

## Domain Map

### stock

负责：

- 股票基础信息。
- 股票代码。
- 股票名称。
- 上市状态。
- 股票生命周期相关规则。

不负责：

- K 线。
- 实时行情。
- 技术指标。
- 登录认证。

入口：

```text
internal/stock/
```

### market

负责：

- 日线行情。
- 分钟行情。
- 行情同步。
- 行情数据查询。

不负责股票身份认证和用户账户。

入口：

```text
internal/market/
```

### analysis

负责：

- 技术指标。
- 分析结果。
- 策略分析。
- 分析任务编排。

入口：

```text
internal/analysis/
```

### auth

负责：

- 用户身份。
- 登录。
- Token。
- 权限检查。

入口：

```text
internal/auth/
```

## Common Change Navigation

| 需求 | 首选位置 |
|---|---|
| 修改股票上市状态规则 | `internal/stock/domain` |
| 修改股票查询流程 | `internal/stock` |
| 修改股票 HTTP API | `internal/stock/handler.go` |
| 修改股票 MySQL 存储 | `internal/stock/infrastructure/repository_mysql.go` |
| 修改行情模型 | `internal/market/domain` |
| 修改行情同步流程 | `internal/market` |
| 修改 Tushare 接入 | `internal/market/infrastructure/provider_tushare.go` |
| 修改指标计算规则 | `internal/analysis/domain` |
| 修改登录流程 | `internal/auth` |
| 修改启动和依赖注入 | `cmd/server` |
| 修改配置解析 | `pkg/config` |
| 修改日志 | `pkg/logger` |

## Reading Order

Agent 处理一个局部需求时，读取与变更直接相关的最小上下文：

```text
运行环境未提供根规则时读取根 AGENTS.md
→ 使用根 INDEX.md 定位目标
→ 读取目标最近一级 AGENTS.md / INDEX.md（存在且相关时）
→ 源码
```

不要从全仓库无差别扫描开始。
