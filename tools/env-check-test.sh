#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "$0")" && pwd)
output=$(DATA_PROVIDER=tushare TUSHARE_TOKEN= "$script_dir/env-check.sh" 2>&1) && {
  printf '%s\n' 'expected missing TUSHARE_TOKEN to fail' >&2
  exit 1
}
[[ "$output" == *'requires TUSHARE_TOKEN'* ]] || {
  printf 'missing-token error was not readable: %s\n' "$output" >&2
  exit 1
}

missing_bin=$(mktemp -d)
output=$(PATH="$missing_bin" /bin/bash "$script_dir/env-check.sh" 2>&1) && {
  rmdir "$missing_bin"
  printf '%s\n' 'expected missing Go dependency to fail' >&2
  exit 1
}
rmdir "$missing_bin"
[[ "$output" == *"missing required command 'go'"* ]] || {
  printf 'missing-dependency error was not readable: %s\n' "$output" >&2
  exit 1
}
[[ "$output" != *'token value'* ]] || {
  printf '%s\n' 'environment error output exposed a token value' >&2
  exit 1
}

output=$(DATA_PROVIDER=unknown "$script_dir/env-check.sh" 2>&1) && {
  printf '%s\n' 'expected unsupported provider to fail' >&2
  exit 1
}
[[ "$output" == *'expected mock or tushare'* ]] || {
  printf 'unsupported-provider error was not readable: %s\n' "$output" >&2
  exit 1
}

if grep -En 'TUSHARE_TOKEN=[^[:space:]]+' "$(dirname -- "$script_dir")/.env.example"; then
  printf '%s\n' '.env.example contains a non-empty token value' >&2
  exit 1
fi

printf '%s\n' 'environment configuration checks passed'
