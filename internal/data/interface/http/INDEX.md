# internal/data/interface/http 导航

## 目录职责

提供 A 股数据同步和股票查询的 HTTP 边界。

## 当前文件

- handler.go：注册数据同步任务路由并处理请求、响应和错误映射。
- stock_query_handler.go：处理股票详情和日线查询请求。
- handler_test.go：验证数据同步 Handler 的请求校验和响应。
- stock_query_handler_test.go：验证股票查询 Handler。
