# Tushare Infrastructure Rules

- Keep provider request/response handling and network policy in this adapter.
- Redact tokens, use timeouts and rate limits, and never log credentials.
- Parse provider values from the returned field names; do not assume column order.
- Preserve raw JSON cell values in generic responses; keep endpoint-specific DTO and unit conversion in later provider mapping code.
- Apply the caller context with a bounded default timeout; allow plain HTTP only for loopback tests.
- Classify provider errors with `shared/apperror`; never include tokens, request bodies, or unredacted provider messages in errors.
