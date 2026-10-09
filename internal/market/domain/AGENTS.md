# Market Domain Rules

- Own stock identity, trading calendar, raw bars, and historical status concepts.
- Keep provider DTOs, SQL, HTTP, and infrastructure imports out of this package.
- Preserve trading-date semantics and explicit amount units from the project rules.
- Represent snapshot identity separately from its local file reference; do not embed provider or SQL DTOs.
- Daily prices store unadjusted prices, amount in yuan, volume in lots, and explicit source hash/revision values.
- Stock and calendar records follow the existing schema and do not invent row revisions; historical stock lookup includes delisted securities.
