# Test and CI Conventions

## Local checks

```bash
make test
make check
```

`make test` runs Go tests, available Python tests, tool tests, provider configuration edge cases, and the business import boundary checks. MySQL migration tests run against a real isolated database when `MYSQL_TEST_DSN` is set; otherwise they report a skip. `make check` adds Go formatting, `go vet`, a build, and the repository secret scan. GitHub Actions provides a disposable MySQL 8.4 service so CI always runs the database integration tests.

## Test placement and fixtures

- Go unit tests stay beside the package in `*_test.go`; use fixed, deterministic fixtures.
- Python worker tests belong in `python/tests/`. Repository tooling tests may live in `tools/test_*.py`.
- External APIs use local mocks/fixtures. CI must never depend on a real provider token or future market data.
- Temporary failure fixtures must be created outside the checkout and removed after the check.

## CI behavior

The workflow in `.github/workflows/ci.yml` runs `tools/check.sh` for pushes and pull requests. Any formatting, test, vet, build, or secret scan failure exits nonzero and blocks a green check. Repository settings must require this workflow's check before merging; that server-side rule is not configured by this file.

The secret scanner reports a file, line, and key name only. It never prints the matched value. False positives should be handled by making fixtures non-secret placeholders rather than adding real credentials or broad scan suppressions.

## Evidence

Each ticket records exact commands and PASS/FAIL/BLOCKED status in `handoffs/completed/<ID>.md`. A local pass does not prove that the hosted workflow ran or that branch protection requires it.
