# internal/data/adapter 导航

## 目录职责

承载 A 股数据 bounded context 的领域特有技术适配器。

## 当前子目录

- mysql/：使用 GORM 实现数据 Repository 和持久化 Record 映射。
- source/：实现 Tushare 和 Mock 数据源 Adapter。
