# screening/ports 规约

- Store interfaces persist screening-domain values and do not perform selection or ranking.
- Keep MySQL, SQL, Tushare, and HTTP types out of this package.
- Pass `context.Context` to storage operations.
- Duplicate run keys return the existing run; results and SUCCESS are committed atomically, and terminal states cannot be overwritten.
- Result pages sort by rank ascending, unranked rows last, then stock code ascending.
