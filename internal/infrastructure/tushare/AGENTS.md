# Tushare 基础设施规约

- Provider 请求/响应处理和网络策略放在本适配器内。
- Token 必须脱敏；网络调用、限流等待和退避使用同一个有界调用 context。
- 每次 HTTP 尝试都必须取得 global 与 per-api 配额；限流突发容量保持为 1。
- 只重试明确可恢复的 429、5xx 和瞬时网络错误；不得仅按通用错误码推断可重试。
- 结构化日志使用注入的 `*slog.Logger`（默认 `slog.Default()`），只记录安全的 `Observation` 字段；禁止记录错误字符串、凭据、请求体、任意请求参数或 provider 原始消息。
- `Observer` 仅用于指标等适配；指标标签不得使用 request ID 或业务日期。
- 根据 provider 返回的字段名解析数据，不假设列顺序。
- 通用响应保留原始 JSON 单元；具体 API DTO 和单位换算留给后续 provider 映射。
- DTO 从 `json.RawMessage` 十进制文本解析 `types.Decimal`；单位转换先做精确十进制运算，不经 `float64`。
- provider 日期使用 `types.ParseTushareDate`；未知字段兼容，必需字段缺失、空值、类型错误和不支持状态必须报 `DATA_INCOMPLETE`。
- 领域映射要求调用者显式传入数据库元数据；不得伪造抓取时间、更新时间或来源摘要。
- `suspend_d` 保留 S/R 事件语义；`stk_limit` 和 `suspend_d` 仅做 DTO/验证，不扩建持久化表或每日状态推断。
- 使用调用方 context 和有界默认超时；明文 HTTP 仅允许 loopback 测试 endpoint。
- 使用 `shared/apperror` 分类错误；错误信息不得包含 Token、请求体或未脱敏 provider 消息。
