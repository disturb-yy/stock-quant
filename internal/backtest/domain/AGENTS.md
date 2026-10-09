# Backtest Domain Rules

- Use point-in-time universes and data availability; never read future values.
- Model T-day signals and T+1 simulated execution with fees and unfilled-order constraints.
- Disclose survivorship, historical ST coverage, costs, and execution limitations.
- Bind each persisted run to a snapshot hash and immutable input key; keep run persistence separate from simulation calculations.
