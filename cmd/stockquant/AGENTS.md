# Stock Quant 命令包规约

- 本包是 Composition Root：解析命令、组装应用服务，并连接标准输入/输出。
- 适配器存在时，在此选择并注入 domain port 实现；可变注册表不得进入 domain 或 app 包。
- 命令层只负责组装，不承载 SQL、HTTP 处理器或业务规则。
- 进程存活与数据库/provider 就绪检查必须区分。
- 数据库迁移只能通过显式 `migrate up|down` 执行，不得放入健康检查或启动流程。
- `migrate down` 仅允许 `development` 和 `test` 环境。
- `sync initial` 仅显式调用；必须要求 Tushare 与 MySQL 配置，不得回退到 Mock、自动迁移或输出密钥。
