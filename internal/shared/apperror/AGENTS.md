# shared/apperror 规约

- 错误携带稳定分类 code，并通过 `Unwrap` 保留原 cause，支持 `errors.Is`/`errors.As`。
- 新 code 需同步项目错误分类文档和适配层映射，不得在包内依赖 HTTP 状态码。
- cause 仅用于内部诊断；API/CLI 对外输出必须由边界层生成安全消息，不能直接泄露凭据、DSN 或绝对路径。
