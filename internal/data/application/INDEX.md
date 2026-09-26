# internal/data/application 导航

## 目录职责

编排 A 股数据同步和股票数据查询用例。

## 当前文件

- service.go：创建、查询、重试和执行数据同步任务。
- stock_query.go：编排股票详情和日线查询。
- worker.go：轮询并执行到期的数据同步任务。
- service_test.go：验证同步 Service 的业务流程。
- stock_query_test.go：验证股票查询用例。
