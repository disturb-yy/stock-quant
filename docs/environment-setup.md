# Development Environment Setup

## Supported baseline

- Go 1.27.x
- Python 3.12.x
- MySQL 8.4 LTS (database work begins in P02)
- Node.js 24.x LTS (dashboard work begins in P11)
- Asia/Shanghai timezone semantics

Install Go and Python before running the current checks. MySQL and Node.js are included in the pinned project baseline but are not required by the P00-01 fixture-only tests. Docker is optional; later tickets may use it for database integration checks.

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

The default `DATA_PROVIDER=mock` mode needs no credentials. To select Tushare, set `DATA_PROVIDER=tushare` and provide `TUSHARE_TOKEN` through a local secret manager or the current shell environment. The doctor check validates that the variable is present, but does not call Tushare or prove account permissions. Do not put real credentials in `.env.example`, Git, test fixtures, logs, or handoff evidence.

Database connection variables are listed in `.env.example`; database configuration becomes active in later tickets. Do not use the example values as production credentials.
