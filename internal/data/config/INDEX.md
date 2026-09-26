# internal/data/config 导航

## 目录职责

读取和校验 A 股数据源及同步策略配置。

## 当前文件

- config.go：定义 Provider、Endpoint、Token、重试和 Worker 配置。
- config_test.go：验证数据源配置的默认值和错误语义。
