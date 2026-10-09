# Development Environment Setup

## Supported baseline

- Go 1.27.x
- Python 3.12.x
- MySQL 8.4 LTS (database work begins in P02)
- Node.js 24.x LTS (dashboard work begins in P11)
- Asia/Shanghai timezone semantics

Install Go and Python before running the current checks. MySQL 8.4 is required for database migration integration tests. Docker is optional for ordinary local checks; CI runs those tests against an isolated MySQL 8.4 service. Set `MYSQL_TEST_DSN` to opt into the real MySQL integration tests locally.

Official release references: [Go releases](https://go.dev/doc/devel/release), [Python version status](https://devguide.python.org/versions/), [MySQL LTS releases](https://dev.mysql.com/doc/refman/8.4/en/mysql-releases.html), and [Node.js releases](https://nodejs.org/en/about/previous-releases).

## Linux and WSL

Install the supported Go and Python release lines using their official downloads or your distribution's version manager. Confirm the selected versions with:

```bash
go version
python3 --version
```

For WSL, keep the checkout and `.venv` inside the Linux filesystem (for example, under `~/projects`) to avoid slower file operations and permission mismatches on `/mnt/c`. Use Linux Go, Python, and Node installations from the WSL shell; do not mix Windows executables into the Linux toolchain.

## Initialize the local environment

```bash
cp .env.example .env
make setup-python
make doctor
make test
```

The Python environment installs the pinned `jsonschema` validator required by the shared protocol. Run `make setup-python` to install the dependencies recorded in `python/requirements.txt`; later tickets must record and lock any additional dependencies. P00-01 does not automatically load `.env`; export values into the shell or provide them through a local secret manager before using `make doctor`.

The Makefile disables Go VCS stamping only when the checkout has no Git repository, so the initial scaffold can build without Git metadata. Once the project is inside Git, Go's normal VCS stamping remains enabled. For direct Go commands in a non-Git checkout, set `GOFLAGS=-buildvcs=false`.

## Provider credentials

The default `DATA_PROVIDER=mock` mode needs no credentials and doctor reports `provider_status=offline/mock`. To select Tushare, set `DATA_PROVIDER=tushare` and provide `TUSHARE_TOKEN` through a local secret manager or the current shell environment. Without a token the check fails and does not fall back to Mock. With a token it reports `provider_status=configured_unverified`; doctor does not call Tushare or prove account permissions. Do not put real credentials in `.env.example`, Git, test fixtures, logs, or handoff evidence.

Database connection variables are listed in `.env.example`. `make migrate-up` runs only when explicitly requested; it never runs at application startup. `make migrate-down` is restricted to `APP_ENV=development` or `APP_ENV=test`. Migration integration tests require `MYSQL_TEST_DSN`; the tests create and drop a uniquely named temporary database and must use a disposable MySQL account. Do not use the example values as production credentials.
