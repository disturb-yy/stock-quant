# Stock Quant Command Rules

- Keep this package as the Composition Root: parse the command, construct application services, and connect standard input/output.
- Do not add SQL, provider requests, HTTP handlers, or business rules here.
- Keep process health distinct from database/provider readiness checks.
