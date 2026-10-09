# Market Domain Rules

- Own stock identity, trading calendar, raw bars, and historical status concepts.
- Keep provider DTOs, SQL, HTTP, and infrastructure imports out of this package.
- Preserve trading-date semantics and explicit amount units from the project rules.
- Represent snapshot identity separately from its local file reference; do not embed provider or SQL DTOs.
- Daily OHLC, amount in yuan, volume in lots, and adjustment factors use exact `types.Decimal` values; calculations must cross an explicit `Float64Checked` boundary.
- Keep metadata such as `source_hash`, `revision`, `fetched_at`, and `updated_at` explicit; provider mapping must not invent it.
- Stock and calendar records follow the existing schema and do not invent row revisions; historical stock lookup includes delisted securities.
- Sync job identity uses a stable task key; statuses are typed domain values and transitions never depend on SQL details.
