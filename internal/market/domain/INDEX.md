# Market Domain Index

| File | Responsibility |
|---|---|
| `doc.go` | Declares the market domain package; behavior is introduced by later tickets |
| `snapshot.go` | Immutable snapshot identity and local data reference values |
| `market_data.go` | Stock identity, exchange calendar, unadjusted daily bar and adjustment factor; market decimal values remain exact |
| `sync_job.go` | Sync task identity, persisted status, attempts, and data quality summary |
