# internal/infrastructure/migration 导航

## 目录职责

提供接收 schema FS 的通用 migration runner。

## 当前文件

- runner.go：基于 GORM 底层数据库连接执行 up、down 和版本状态处理。
- runner_test.go：验证 migration runner 的方向和错误语义。
