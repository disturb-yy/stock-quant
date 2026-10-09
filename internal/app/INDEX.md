# Application Package Index

| File | Responsibility |
|---|---|
| `health.go` | Minimal process-level health use case for the P00 CLI command |
| `health_test.go` | Process status and canceled-context behavior |
| `factor_service.go` | Snapshot and protocol runner composition for raw-factor requests |
| `factor_service_test.go` | Fake-port request/response, cancellation, and dependency failure coverage |
| `initial_market_sync.go` | P04-01 初始股票身份与沪深交易日历同步用例，先取齐数据再写入仓储 |
| `initial_market_sync_test.go` | L/D/P 合并、历史身份可见性、完整日历覆盖及 provider 失败测试 |
