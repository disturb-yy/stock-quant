# internal/data/domain 导航

## 目录职责

表达 A 股数据同步和查询的业务模型、接口与规则。

## 当前文件

- data.go：定义 Instrument、DailyBar 及批量数据模型。
- errors.go：定义数据同步和查询领域错误。
- model.go：定义同步任务、状态、目标和来源模型。
- query.go：定义股票标识、查询输入输出和 Repository 接口。
- model_test.go：验证领域模型和状态转换。
- query_test.go：验证查询模型和分页规则。
