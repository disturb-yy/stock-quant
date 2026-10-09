# market/ports 规约

- 接口描述 market domain 提供的能力，不包含 SQL、Tushare DTO、HTTP 或重试细节。
- `SnapshotRepository` 只读取已形成的规范快照；`MarketDataProvider` 只提供上游交易日历数据，两者职责不可混淆。
- 所有可能阻塞的操作接收 `context.Context`；日期使用 `shared/types.TradingDate`。
- SyncJobStore 只保存任务身份、计数、质量报告和状态；任务调度、provider 调用与重试策略不放在仓储端口。
- 同一 `task_key` 返回已存在任务；状态转换由明确的方法表达，不提供任意状态覆盖接口。
