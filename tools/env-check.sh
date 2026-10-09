#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'environment check: %s\n' "$1" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command '$1'; see docs/environment-setup.md"
}

require_command go
require_command python3

go_version=$(go env GOVERSION)
[[ "$go_version" =~ ^go1\.27\.[0-9]+$ ]] || fail "Go 1.27.x required; found $go_version"

python_version=$(python3 -c 'import sys; print(".".join(map(str, sys.version_info[:3])))')
[[ "$python_version" =~ ^3\.12\.[0-9]+$ ]] || fail "Python 3.12.x required; found $python_version"

provider=${DATA_PROVIDER:-mock}
provider_status=offline/mock
case "$provider" in
  mock) ;;
  tushare)
    [[ -n "${TUSHARE_TOKEN:-}" ]] || fail "DATA_PROVIDER=tushare requires TUSHARE_TOKEN; mock fallback is disabled; configure it in your local environment"
    provider_status=configured_unverified
    ;;
  *) fail "unsupported DATA_PROVIDER '$provider'; expected mock or tushare" ;;
esac

printf 'environment check: Go %s, Python %s, DATA_PROVIDER=%s, provider_status=%s\n' "$go_version" "$python_version" "$provider" "$provider_status"
