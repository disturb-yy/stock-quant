# internal 导航

## 目录职责

保存服务内部的配置、领域实现、健康检查和共享技术能力。

## 当前子目录

- config/：运行时数据库配置读取与校验。
- data/：A 股数据 bounded context。
- health/：HTTP 进程存活检查。
- infrastructure/：跨 bounded context 共享的技术 Infrastructure。
