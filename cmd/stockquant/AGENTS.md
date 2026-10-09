# Stock Quant Command Rules

- Keep this package as the Composition Root: parse the command, construct application services, and connect standard input/output.
- Select and inject concrete domain ports here when adapters exist; keep mutable registries out of domain and app packages.
- Trusted Python strategy registration is owned by P08-04, not by the P01-03 scaffolding.
- Do not add SQL, provider requests, HTTP handlers, or business rules here.
- Keep process health distinct from database/provider readiness checks.
- Database migrations are explicit `migrate up|down` commands; do not run them during health checks or application startup.
- `migrate down` must reject environments other than `development` and `test`.
