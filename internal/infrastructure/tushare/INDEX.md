# Tushare Infrastructure Index

| File | Responsibility |
|---|---|
| `doc.go` | Declares the Tushare adapter package; the client is implemented in P03 |
| `client.go` | Configurable, bounded HTTP client; provider envelope and dynamic row parser |
| `client_test.go` | `httptest` coverage for request shape, error mapping, timeouts, and token redaction |
| `testdata/` | Local fixtures for reordered fields, empty results, and permission errors |
