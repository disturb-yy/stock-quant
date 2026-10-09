# Tushare 基础设施规约

- Provider 请求/响应处理和网络策略放在本适配器内。
- Token 必须脱敏；网络调用、限流等待和退避使用同一个有界调用 context。
- 每次 HTTP 尝试都必须取得 global 与 per-api 配额；限流突发容量保持为 1。
- 只重试明确可恢复的 429、5xx 和瞬时网络错误；不得仅按通用错误码推断可重试。
- 日志/指标只通过安全的 `Observation` 字段输出，禁止记录凭据、请求体、任意请求参数或 provider 原始消息。
- 根据 provider 返回的字段名解析数据，不假设列顺序。
- 通用响应保留原始 JSON 单元；具体 API DTO 和单位换算留给后续 provider 映射。
- 使用调用方 context 和有界默认超时；明文 HTTP 仅允许 loopback 测试 endpoint。
- 使用 `shared/apperror` 分类错误；错误信息不得包含 Token、请求体或未脱敏 provider 消息。
