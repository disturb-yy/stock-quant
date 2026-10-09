# screening/ports 规约

- Store interfaces persist screening-domain values and do not perform selection or ranking.
- Keep MySQL, SQL, Tushare, and HTTP types out of this package.
- Pass `context.Context` to storage operations.
