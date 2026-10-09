# market/ports 规约

- 接口描述 market domain 提供的能力，不包含 SQL、Tushare DTO、HTTP 或重试细节。
- `SnapshotRepository` 只读取已形成的规范快照；`MarketDataProvider` 只提供上游交易日历数据，两者职责不可混淆。
- 所有可能阻塞的操作接收 `context.Context`；日期使用 `shared/types.TradingDate`。
