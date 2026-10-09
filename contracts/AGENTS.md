# 契约包规约

- `strategy-request.schema.json` 与 `strategy-result.schema.json` 是跨语言协议的唯一结构定义。
- Go DTO 和验证器可以与契约文件同目录，但不得复制一份可独立修改的 schema。
- 修改 wire format 时同步更新 Python DTO、两端测试、examples 和 `docs/07-python-protocol.md`。
- `schema_version` 必须显式提供；不允许静默填默认值。
