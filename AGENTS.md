# AGENTS.md

## Purpose

本文件定义整个项目的 AI 编程规约。

修改前，读取与目标变更直接相关的最近一级 `AGENTS.md`、`INDEX.md`、源码和测试；根规则未由运行环境提供时，再读取根 `AGENTS.md`。

不得仅凭文件名或猜测修改代码。

## Architecture

项目采用 Go + DDD Lite。

核心原则：

```text
业务领域 package
└── domain/       # 强边界
```

每个业务领域位于 `internal/<bounded-context>/`：

- Domain 放在 `internal/<bounded-context>/domain`，作为强边界。
- Application 默认留在领域根 package。
- Interface 默认留在领域根 package。
- Infrastructure 放在 `internal/<bounded-context>/infrastructure`。

仅当职责已形成独立边界，或现有 package 出现明确的依赖或测试隔离问题时进一步拆包；不改变公开契约的局部拆包无需额外确认。

## Technology Choices

- HTTP 框架使用 Gin，仅限 Interface 层和 `cmd/server` 的路由装配。
- 关系数据库使用 MySQL；Driver、SQL 和数据库访问实现仅限 Infrastructure 层。
- 具体依赖及版本以 `go.mod` 为准。

## Dependency Rules

允许：

```text
Interface -> Application
Application -> Domain
Infrastructure -> Domain
cmd/server -> all concrete implementations
pkg -> third-party / stdlib
```

禁止：

```text
Domain -> Application
Domain -> Infrastructure
Domain -> Interface
Domain -> Gin/GORM/Redis/Tushare/Kafka SDK
领域 A -> 领域 B 的具体基础设施实现
```

跨领域调用优先通过公开接口或 Application Service。

## Domain Rules

`internal/<bounded-context>/domain`：

- 只放业务概念。
- 可以包含 Entity、Value Object、Aggregate、Domain Service、Repository Interface。
- 不处理 HTTP。
- 不写 SQL。
- 不读配置。
- 不直接调用外部 API。
- 不依赖数据库实现。
- 不依赖具体日志框架；已有仓库级、领域无关的日志接口时可以依赖。引入新的跨领域日志抽象属于架构变更，实施前确认。

Domain 代码应该能在不启动数据库、HTTP Server、Redis 的情况下进行单元测试。

## Application Rules

Application 负责：

- 用例编排。
- 调用领域能力。
- 调用 Repository / Provider 抽象。
- 事务边界。
- 权限或流程级协调。

Application 不负责：

- SQL 细节。
- HTTP 参数解析。
- 数据库表结构细节。
- 把业务规则全部写成 if/else 而绕开 Domain。

## Interface Rules

Handler / HTTP Adapter 负责：

- 参数解析。
- 输入校验。
- DTO 转换。
- 调用 Application。
- HTTP 状态码与响应输出。

不得在 Handler 中直接：

- 写 SQL。
- 调 Repository 实现。
- 调第三方 API。
- 编写核心业务规则。

## Infrastructure Rules

`internal/<bounded-context>/infrastructure` 负责实现：

- Repository。
- Provider。
- Cache。
- MQ。
- External API Adapter。

Infrastructure 可以依赖 Domain 中定义的接口和模型。

## Package Rules

仅在 package 具有独立边界、导航需求或局部规则时创建 `AGENTS.md` / `INDEX.md`；否则继承最近一级规则。

新增需要独立说明的 package 时，同步创建相应文档。

删除、重命名、迁移重要文件时，必须同步更新对应 `INDEX.md`。

## File Naming

推荐：

```text
service.go
sync.go
handler.go
repository.go
repository_mysql.go
provider.go
provider_tushare.go
model.go
mapper.go
```

避免：

```text
utils.go
common.go
helper.go
manager.go
misc.go
```

除非职责非常明确。

## Error Handling

- 错误必须向上返回或明确处理。
- 不吞错误。
- 使用 `%w` 保留错误链。
- Domain 错误应表达业务语义。
- Infrastructure 错误可包装技术上下文，但不得泄露敏感数据。

## Context

所有可能阻塞或涉及 I/O 的 Application / Infrastructure 方法应优先接收：

```go
context.Context
```

不得将 Context 存储在 struct 中长期持有。

## Testing

优先级：

1. Domain 单元测试。
2. Application 用例测试。
3. Repository / Provider 集成测试。
4. Handler 测试。

修改业务规则时必须优先补充 Domain 测试。

## Change Discipline

修改前：

- 根据 `INDEX.md` 定位代码。
- 检查依赖方向。
- 确认变更属于哪个领域和层级。

修改后：

- 行为变化时更新相关测试。
- 文件职责、入口或重要路径变化时更新受影响的 `INDEX.md`。
- 架构规则改变时更新 `AGENTS.md`。
- 先运行受影响 package 的测试；跨 package、依赖或公共契约变更时执行 `go test ./...`。
- 必要时执行 `go vet ./...`。

## Prohibited

禁止为了完成任务：

- 将所有逻辑塞进 service.go。
- 从 Domain 直接调用 MySQL/Redis/API。
- Handler 直接访问数据库。
- 创建无业务意义的 package。
- 未阅读局部 AGENTS.md 就跨 package 修改。
- 复制父级 AGENTS.md 全文到子 package。

## Coding Style
- 使用中文注释
- 复杂函数添加注释
- 单函数避免过长，尽量不超过50行，及时抽象成多个函数
