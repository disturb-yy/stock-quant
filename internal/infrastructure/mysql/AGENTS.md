# MySQL Infrastructure Rules

- Implement application/domain-owned ports; keep SQL and driver details here.
- Use parameterized queries and explicit transactions for data writes.
- Keep each repository batch in one transaction; reject malformed values before committing, and preserve `source_hash`/`revision` semantics from `docs/05-database.md`.
- Date range reads are ascending; all-market date scans use the date-leading index and ascending `ts_code`.
- MySQL DDL is not transaction-atomic: migrations must record version/checksum/dirty direction, hold a connection-bound advisory lock, and use retry-safe SQL.
- Rollback only the latest applied migration; down SQL may drop only objects created by that migration.
- Do not import another domain's infrastructure implementation.
