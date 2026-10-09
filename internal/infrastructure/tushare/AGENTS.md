# Tushare Infrastructure Rules

- Keep provider request/response handling and network policy in this adapter.
- Redact tokens, use timeouts and rate limits, and never log credentials.
- Parse provider values from the returned field names; do not assume column order.
