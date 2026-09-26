# 后端工程规约

## 目的与阅读顺序

本文件定义本项目的 AI 编程规约。修改前必须读取：

1. 与目标变更直接相关的最近一级 `AGENTS.md` 与 `INDEX.md`（如存在）；
2. 目标源码与相邻测试；
3. 根 `AGENTS.md`、`README.md` 与 `go.mod`。

不得仅凭文件名或猜测修改代码。新增业务能力前，必须已有确认的产品设计、接口契约与验收标准。

## 文档与交付范围

- 项目内的 README、AGENTS、INDEX、package 文档和源码注释只描述当前已实现、可验证的能力、职责与使用方式。
- 不在项目中记录待实现的功能、推测性架构、占位接口、`TODO` / `FIXME` 计划或“以后支持”的说明；未满足的设计或依赖留在需求管理系统，不以空代码或文档占位替代。
- 需求、Story、Bug 或问题单标识用于外部跟踪与 Git 提交；项目内说明只陈述最终行为，例如“提供 `GET /api/v1/health` 用于进程存活检查”，而不是“由某需求引入”或“修复某问题单”。
- 若变更不产生可运行、可验证的能力，不为它添加项目内功能说明；只在外部工作项和提交记录中保留交付背景。

## 技术选型

- 运行时使用 Go 1.27。
- HTTP 服务使用 Gin；MySQL 数据访问使用 GORM 的 MySQL Dialector（`gorm.io/driver/mysql`）。
- 当前领域 adapter 使用 GORM 的 `*gorm.DB` 完成查询、事务和持久化；领域 adapter 仍负责本领域的模型映射、查询语义和 Repository 实现。
- 数据库 schema 迁移继续使用 `golang-migrate` 执行版本化 SQL，由共享 migration runner 接收 GORM 数据库实例；GORM 不替代版本化 migration。
- 数据库连接、Ping 和连接池生命周期由 `internal/infrastructure/database` 负责；调用方在 Composition Root 获取底层连接并负责关闭。

## 架构

项目采用 Go + DDD Lite。业务领域以 `internal/<bounded-context>/` 为边界，共享技术基础设施放在 `internal/infrastructure/`：

```text
internal/
├── infrastructure/              # 跨 bounded context 共享的技术能力
│   ├── database/                 # 数据库连接、连接池等通用能力
│   └── migration/                # 接收调用方 schema FS 的通用迁移执行器
└── <bounded-context>/
    ├── domain/                   # 强业务边界
    ├── application/              # Application：用例编排
    ├── interface/                # Interface：HTTP 适配
    └── adapter/                  # 当前 bounded context 特有的技术适配器
```

- `domain/`：实体、值对象、聚合、领域服务和业务所需的 Repository 接口。
- 领域根 package：Application Service 与 HTTP Handler；职责通过文件命名区分。
- `internal/infrastructure/`：多个 bounded context 可复用的数据库连接、迁移执行器等技术能力；不得依赖任何具体 domain 的模型、表或业务接口。
- `<bounded-context>/adapter/`：当前 bounded context 特有的 Repository、Provider、Cache、消息队列与外部 API Adapter 具体实现。
- 仅当出现明确的职责边界、依赖问题或测试隔离需求时继续拆包；不得为“看起来像 DDD”预建空包。

## 依赖方向

允许：

```text
Interface -> Application
Application -> Domain
共享 Infrastructure -> 标准库或第三方库，不得依赖 bounded context
Domain Adapter -> Domain 与共享 Infrastructure
cmd/server -> 所有具体实现
pkg -> 标准库或第三方库
```

禁止：

```text
Domain -> Application / Infrastructure / Interface
Domain -> Gin、数据库驱动、Redis、消息队列或外部 Provider SDK
领域 A -> 领域 B 的具体 Adapter 实现
```

跨领域协作通过公开接口或 Application Service 完成，不得直接依赖对方的具体存储或 Provider。

## 各层职责

### Domain

`internal/<bounded-context>/domain` 只表达业务概念与规则：

- 可以包含 Entity、Value Object、Aggregate、Domain Service、Repository Interface；
- 不处理 HTTP、不写 SQL、不读配置、不调用外部 API；
- 不依赖具体日志、数据库或消息组件；
- 应能在不启动 HTTP Server、数据库或外部服务的情况下完成单元测试。

### Application

Application Service 负责用例编排、事务边界、权限或流程级协调，并调用 Domain 与抽象接口。不得解析 HTTP 参数、编写 SQL、泄露表结构，或用大量 `if/else` 取代领域规则。

### Interface

HTTP Handler / Adapter 负责参数解析、输入校验、DTO 转换、调用 Application Service 和 HTTP 响应映射。不得直接写 SQL、调用 Repository 具体实现或第三方 API，也不得承载核心业务规则。

Gin 只允许出现在 Interface 层以及 `cmd/server` 的全局路由装配。

### 共享 Technical Infrastructure

- `internal/infrastructure/database` 提供跨 bounded context 复用的数据库连接、Ping 和连接池生命周期能力；不得包含任何 `t_*` 表语义或 domain 类型。
- `internal/infrastructure/migration` 提供通用 migration runner，必须接收调用方提供的 `fs.FS`；不得嵌入或解析具体 domain 的 schema。
- 其他共享技术能力只有在存在真实复用方时才加入；不得预建空包或把单一 domain 的适配器上提。

### Domain Adapter

`internal/<bounded-context>/adapter` 承载当前 bounded context 特有的 Repository、Provider、Cache、MQ 和外部 API Adapter。SQL、row mapping、外部 Provider 响应转换可以依赖该 domain 的模型和接口，但不能反向污染共享 Infrastructure。

领域 schema SQL 是 domain 资源；例如 A 股数据迁移保留在根目录 `migrations/`，由该包暴露 `embed.FS` 后交给共享 migration runner 执行。

## Composition Root 与路由

`cmd/server` 是服务的 Composition Root，仅负责：

- 加载配置与初始化日志；
- 创建数据库、外部客户端、Repository / Provider 具体实现（在相应能力被确认后）；
- 注入 Application Service 与注册 Handler；
- 创建 HTTP Server，并完成启动和关闭。

`cmd/server/router.go` 是全局路由装配位置。当前将它放在此处是合理的，因为它只：

1. 创建 Gin Engine；
2. 安装跨领域中间件（例如日志和 Recovery）；
3. 创建 API 版本路由组；
4. 调用各领域的 `RegisterRoutes` 注册 Handler。

`router.go` 不得新增匿名业务 Handler、DTO 解析、领域服务调用、SQL、Repository / Provider 具体实现或业务规则。新领域接口应由 `internal/<bounded-context>/handler.go`（或职责明确的 HTTP Adapter）拥有，再由 `router.go` 调用其注册函数。

`main.go` 保持薄，只组合依赖、创建 Server 并处理生命周期；修改业务行为时不应优先改动 `cmd/server`。

## 包与文件

- 仅在 package 存在独立边界、导航需求或局部规则时创建子级 `AGENTS.md` / `INDEX.md`；不得复制根规约全文。
- 新增需要独立说明的 package 时，同步创建对应的局部文档。
- 删除、重命名或迁移重要入口时，同步更新受影响的 `INDEX.md`。
- 推荐文件名：`service.go`、`handler.go`、`repository.go`、`repository_mysql.go`、`provider.go`、`model.go`、`mapper.go`。
- 避免无职责的 `utils.go`、`common.go`、`helper.go`、`manager.go`、`misc.go`。

## 配置与敏感信息

- 配置只能从 Composition Root 进入系统；Domain、Application 和 Infrastructure 不得分散调用 `os.Getenv` 或读取配置文件。
- 当前基础框架仅有 `HTTP_ADDRESS` 时可在 `cmd/server` 读取；新增第二类运行配置前，必须收敛到职责明确的配置边界，并在启动时一次性校验后注入依赖。
- 必填配置缺失或格式错误必须阻止服务启动，并返回可诊断但不泄露敏感值的错误。
- 密钥、Token、密码、DSN 与个人数据不得硬编码、写入示例配置、日志、错误响应或测试快照；本地 `.env` 文件不提交。
- 配置结构、环境变量名称和默认值属于运行契约。新增或修改它们时，更新 README 或对应运行文档。

## HTTP 契约与错误语义

- 对外 HTTP 接口统一从版本化路径（当前为 `/api/v1`）暴露；新接口必须先有已确认的 Method、Path、请求字段、响应字段、状态码、错误、分页、空值与空结果语义。
- 不在未确认契约前引入全局响应 envelope、错误码体系或 OpenAPI 生成链路；一旦选定，应由唯一的契约来源统一维护，不允许 Handler 各自定义格式。
- Handler 在边界完成参数解析、格式校验和错误映射；Application / Domain 仍必须防御性校验，不依赖 HTTP 层绕过非法调用。
- 客户端错误与服务端错误必须可区分；响应不得暴露 SQL、堆栈、Token、DSN 或内部实现细节。
- 新增公开接口时，补充对应 Handler 测试；契约工具链确认后，同步更新其生成工件，不手写或猜测生成结果。

## 健康检查、日志与 panic 边界

- 所有 HTTP Router 必须注册 Recovery。当前 `gin.Recovery()` 覆盖同一请求链中的 Handler 和中间件 panic，并向客户端返回 500；它不覆盖启动期或独立 goroutine 的 panic。
- Recovery 与日志不得将 panic 堆栈、密钥或内部错误细节写入 HTTP 响应。需要保留诊断信息时，只写入受控日志目标。
- 当前 `GET /api/v1/health` 仅表示 HTTP 进程存活，不代表数据库、消息队列或外部 Provider 已就绪。依赖被确认后，另行定义并验收就绪检查的依赖范围与失败语义。
- 日志应记录可操作的事件上下文，避免记录敏感信息；引入结构化日志、请求 ID、追踪或指标前，先确认其运行与存储方案。
- 后台任务必须由 Composition Root 明确创建、持有取消 Context 并在关闭时等待退出。是否恢复后台 goroutine 的 panic 应由任务的失败语义决定；不得静默吞掉 panic 或让无主 goroutine 长期运行。

## 错误、Context 与测试

- 所有错误必须向上返回或明确处理；返回底层错误时使用 `%w` 保留错误链。
- Domain 错误应表达业务语义；Infrastructure 错误可补充技术上下文，但不得泄露敏感数据。
- 所有可能阻塞或 I/O 的 Application / Infrastructure 方法优先以 `context.Context` 为第一个参数；不得把 Context 存入长期持有的 struct。
- 测试优先级：Domain 单元测试、Application 用例测试、Repository / Provider 集成测试、Handler 测试。
- 修改业务规则时必须先补充或更新 Domain 测试；跨 package、依赖或公开契约变更时运行 `go test ./...`，必要时运行 `go vet ./...`。

## 变更纪律

修改前：确认领域与层级、检查依赖方向、读取目标局部规约与测试。修改后：更新受影响测试和导航文档，运行最小相关验证。

禁止为了完成任务将逻辑全部塞入 `service.go`、从 Domain 直接调用技术实现、让 Handler 直接访问数据库、创建无业务意义的 package，或绕过局部规约跨 package 修改。

## 注释与函数复杂度

- 注释默认使用中文；Gin、Go、HTTP、API、DTO、MySQL、Redis、DDD 等英文专有名词保留英文写法。
- 注释应解释“为什么这样做”：业务约束、架构边界、兼容性原因、非显而易见的取舍或异常处理原因。不要逐行复述代码已经表达的“做了什么”。
- 仅在需要说明上述原因、导出的边界语义或复杂控制流时添加注释；命名清晰的简单代码不为凑注释而注释。
- 生产函数最多 50 个有效代码行（不含空行与注释）。超过上限时，按清晰且可独立测试的职责拆分；不得为满足行数而制造无意义的包装函数。
