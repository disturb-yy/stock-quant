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
exit 42
MOCK_GO
chmod +x "$temp_dir/go"

output=$(GO="$temp_dir/go" "$script_dir/check-domain-imports.sh" 2>&1) && {
  printf '%s\n' 'expected package discovery failure to fail boundary check' >&2
  exit 1
}
if [[ "$output" == *'domain package import boundaries passed'* ]]; then
  printf '%s\n' 'boundary checker hid a package discovery failure' >&2
  exit 1
fi

printf '%s\n' 'domain import checker failure path passed'
