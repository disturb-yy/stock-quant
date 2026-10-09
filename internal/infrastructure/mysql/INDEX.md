# MySQL Infrastructure Index

| File | Responsibility |
|---|---|
| `doc.go` | Declares the MySQL adapter package |
| `migrate.go` | Loads versioned migrations, serializes execution, tracks checksums/dirty state, and applies up/down SQL |
| `migrate_test.go` | MySQL 8.4 integration tests for repeatability, rollback scope, recovery, locks, and connection failure |
