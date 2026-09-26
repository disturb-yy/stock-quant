# internal/data/adapter/source 导航

## 目录职责

提供 A 股数据源 Adapter。

## 当前文件

- factory.go：根据配置选择 Provider Adapter。
- mock.go：提供测试和本地运行使用的 Mock 数据源。
- tushare.go：调用 Tushare API 并转换数据。
- source_test.go：验证数据源选择和数据转换行为。
