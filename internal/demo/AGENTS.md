# AGENTS.md

## Scope

本目录是开发环境演示数据的应用与 Interface 边界。

## Responsibilities

允许：

- 编排版本化 fixture seed。
- 编排真实数据库状态查询。
- 注册仅 development 环境使用的 demo-status HTTP 路由。

禁止：

- 在本目录写 SQL 或导入 MySQL Driver。
- 使用内存数据作为运行时 fallback。
- 将 local fixture 宣称为真实 Provider。

SQL、数据库连接和具体 migration 必须留在 `infrastructure`。
