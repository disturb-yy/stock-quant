# market/ports 导航

| 文件 | 职责 |
|---|---|
| `ports.go` | 快照读取、初始股票身份/交易日历 provider、上游日历查询及行情仓储接口 |
| `sync_job_store.go` | 同步任务幂等创建、读取和状态变更接口 |
| `ports_test.go` | MarketDataProvider fake 的日期范围请求/响应测试 |
