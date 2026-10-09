#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "$0")" && pwd)
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT

cat > "$temp_dir/go" <<'MOCK_GO'
#!/usr/bin/env bash
if [[ "$*" == 'list -buildvcs=false -m' ]]; then
  printf '%s\n' 'stock-quant'
  exit 0
fi
if [[ "$*" == 'list -buildvcs=false -f {{.ImportPath}} {{join .Imports " "}} ./internal/...' ]]; then
  case "${MOCK_GO_MODE:-}" in
    app-violation)
      printf '%s\n' 'stock-quant/internal/app stock-quant/internal/infrastructure/mysql'
      ;;
    domain-violation)
      printf '%s\n' 'stock-quant/internal/market/domain stock-quant/internal/infrastructure/mysql'
      ;;
    clean)
      printf '%s\n' 'stock-quant/internal/app stock-quant/internal/market/domain'
      ;;
    discovery-failure)
      exit 42
      ;;
    *)
      exit 43
      ;;
  esac
  exit 0
fi
exit 44
MOCK_GO
chmod +x "$temp_dir/go"

expect_violation() {
  local mode=$1 expected=$2 output
  output=$(MOCK_GO_MODE="$mode" GO="$temp_dir/go" "$script_dir/check-domain-imports.sh" 2>&1) && {
    printf 'expected %s import violation to fail boundary check\n' "$mode" >&2
    exit 1
  }
  if [[ "$output" != *"$expected"* ]]; then
    printf 'boundary checker did not report %s violation: %s\n' "$mode" "$output" >&2
    exit 1
  fi
}

expect_violation app-violation 'stock-quant/internal/app imports infrastructure'
expect_violation domain-violation 'stock-quant/internal/market/domain imports infrastructure'

clean_output=$(MOCK_GO_MODE=clean GO="$temp_dir/go" "$script_dir/check-domain-imports.sh")
if [[ "$clean_output" != *'business package import boundaries passed'* ]]; then
  printf 'expected clean business packages to pass: %s\n' "$clean_output" >&2
  exit 1
fi

failure_output=$(MOCK_GO_MODE=discovery-failure GO="$temp_dir/go" "$script_dir/check-domain-imports.sh" 2>&1) && {
  printf '%s\n' 'expected package discovery failure to fail boundary check' >&2
  exit 1
}
if [[ "$failure_output" == *'business package import boundaries passed'* ]]; then
  printf '%s\n' 'boundary checker hid a package discovery failure' >&2
  exit 1
fi

printf '%s\n' 'business import checker regression paths passed'
