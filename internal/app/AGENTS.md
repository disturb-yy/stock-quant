# Application Package Rules

- Coordinate application use cases and cross-domain calls through explicit ports.
- Keep SQL, Tushare HTTP, Python subprocess, and transport parsing out of this package.
- Pass `context.Context` to operations and return wrapped errors.
- Compose injected market snapshot and factor runner ports; preserve typed error codes and causes.
- The P00 health service only reports process liveness; do not describe it as database/provider readiness.
