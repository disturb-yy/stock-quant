# Stock DDD

一个使用 Go 构建的股票业务服务示例项目，采用轻量 DDD（DDD Lite）组织方式。

## 目标

项目重点不是机械套用 DDD 四层目录，而是建立稳定的业务边界和依赖方向：

- 先按业务领域拆分 package。
- 每个业务领域内部使用独立 `domain` 子包，作为强业务边界。
- Application 和 Interface 默认保留在领域根 package，通过文件职责和命名区分。
- Infrastructure 放在领域的 `infrastructure` 子包。

## 技术选型

| 类别 | 选型 | 用途 |
|---|---|---|
| 开发语言 | Go | 服务端开发 |
| HTTP 框架 | Gin | HTTP 路由、中间件和请求响应适配 |
| 关系数据库 | MySQL | 业务数据持久化 |

具体依赖及版本以 `go.mod` 为准。Gin 仅用于 Interface 层，MySQL Driver 及数据库访问实现仅用于 Infrastructure 层；Domain 不依赖这些技术实现。

## 架构

```text
stock-ddd/
├── AGENTS.md
├── INDEX.md
├── README.md
│
├── cmd/
│   └── server/
│       ├── AGENTS.md
│       ├── INDEX.md
│       └── main.go
│
├── internal/
│   ├── stock/
│   │   ├── AGENTS.md
│   │   ├── INDEX.md
│   │   ├── domain/
│   │   │   ├── AGENTS.md
│   │   │   ├── INDEX.md
│   │   │   ├── stock.go
│   │   │   └── repository.go
│   │   ├── infrastructure/
│   │   │   └── repository_mysql.go
│   │   ├── service.go
│   │   └── handler.go
│   │
│   ├── market/
│   ├── analysis/
│   └── auth/
│
└── pkg/
    ├── config/
    └── logger/
```

## 分层含义

| 层级 | 职责 |
|---|---|
| Domain | 实体、值对象、聚合、领域规则、领域接口 |
| Application | 用例编排、事务边界、调用领域能力 |
| Interface | HTTP / RPC / CLI 等入口适配 |
| Infrastructure | MySQL、Redis、Tushare、Kafka 等技术实现 |

核心依赖规则：

```text
Interface ───────┐
                 ▼
Application ──> Domain
                 ▲
Infrastructure ──┘
```

`domain` 不允许依赖 Gin、GORM、MySQL Driver、Redis Client、Tushare SDK 等外部技术实现。

## 文档使用方式

### 人类

优先阅读：

1. `README.md`
2. 根 `INDEX.md`
3. 目标 package 的 `INDEX.md`
4. 源码

### AI Agent

读取与任务直接相关的最小上下文：

```text
任务
 ↓
运行环境未提供根规则时读取根 AGENTS.md
 ↓
使用根 INDEX.md 定位目标
 ↓
读取目标最近一级 AGENTS.md / INDEX.md（存在且相关时）
 ↓
必要源码
 ↓
实现与验证
```

## 开发原则

- 业务优先于技术实现。
- package 表示业务或技术边界。
- `internal/<bounded-context>/domain` 是核心业务规则区。
- `internal/<bounded-context>/infrastructure` 是数据库和外部服务适配区。
- Domain 不依赖 Infrastructure。
- Handler 不承载核心业务逻辑。
- Repository 接口定义在业务需要它的一侧。
- MySQL、Redis、外部 API 仅作为接口实现。
- 不因为“看起来像 DDD”而创建无意义目录。

## 构建

```bash
go mod tidy
go build ./...
```

## 运行

```bash
go run ./cmd/server
```

## 测试

```bash
go test ./...
```

## 本地运行与健康检查

```bash
HTTP_ADDRESS=:8357 go run ./cmd/server
curl -i http://127.0.0.1:8357/api/v1/health
```

健康接口成功返回 HTTP `200` 和 `{"status":"ok"}`。`HTTP_ADDRESS` 缺失时使用 `:8357`；地址格式错误或端口被占用时，启动日志会输出具体地址和底层错误。

## FND-003 本地 Demo

从后端工作区执行下面一条命令会启动 Docker MySQL、幂等执行 migration + seed、启动后端和前端：

```bash
cd /home/jadon/projects/go/stock-quant && ./scripts/dev/start.sh
```

默认地址为：MySQL `127.0.0.1:3307`、后端 `http://127.0.0.1:8357`、前端 `http://127.0.0.1:4173`。脚本显式设置 `VITE_API_PROXY_TARGET=http://127.0.0.1:8357`；按 `Ctrl-C` 会停止后端、前端并停止本地 MySQL 容器，数据卷保留以便下次复用。

单独执行 seed：

```bash
DB_HOST=127.0.0.1 DB_PORT=3307 DB_NAME=stock_quant_dev DB_USER=stock_quant DB_PASSWORD=stock_quant_dev go run ./cmd/seed
```

开发状态接口为 `GET http://127.0.0.1:8357/api/v1/dev/demo-status`，市场概览接口为 `GET http://127.0.0.1:8357/api/v1/markets/overview`，行业表现接口为 `GET http://127.0.0.1:8357/api/v1/markets/sectors`，OpenAPI 为 `GET http://127.0.0.1:8357/api/v1/openapi.json`。`mode=demo` 表示数据库中 fixture 版本和四类计数完全匹配；`mode=fallback` 表示尚未 seed 或请求了当前未实现的真实 Provider；`mode=real` 为未来真实 Provider 实现保留，当前不会被伪装返回。市场概览和行业表现响应中的 `source` 会明确标记 Seed 版本。

## 静态检查

```bash
go vet ./...
```

如果项目引入 golangci-lint：

```bash
golangci-lint run
```
# stock-quant
