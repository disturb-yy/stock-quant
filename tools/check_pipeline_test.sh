#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "$0")" && pwd)
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT

make_fixture() {
  local root=$1
  mkdir -p "$root"
  cat > "$root/go.mod" <<'GO_MOD'
module example.test/check

go 1.27.0
GO_MOD
  cat > "$root/Makefile" <<'MAKEFILE'
test:
	GOFLAGS=-buildvcs=false go test ./...
MAKEFILE
}

unformatted_root="$temp_dir/unformatted"
make_fixture "$unformatted_root"
cat > "$unformatted_root/main.go" <<'GO_SOURCE'
package main
func main(){}
GO_SOURCE
output=$(GOFLAGS=-buildvcs=false "$script_dir/check.sh" "$unformatted_root" 2>&1) && {
  printf '%s\n' 'check.sh accepted an unformatted Go file' >&2
  exit 1
}
[[ "$output" == *'need gofmt'* ]] || {
  printf 'format failure was not reported: %s\n' "$output" >&2
  exit 1
}

failing_test_root="$temp_dir/failing-test"
make_fixture "$failing_test_root"
cat > "$failing_test_root/main.go" <<'GO_SOURCE'
package main

func main() {}
GO_SOURCE
cat > "$failing_test_root/main_test.go" <<'GO_TEST'
package main

import "testing"

func TestIntentionalFailure(t *testing.T) {
	t.Fatal("intentional quality gate failure")
}
GO_TEST
output=$(GOFLAGS=-buildvcs=false "$script_dir/check.sh" "$failing_test_root" 2>&1) && {
  printf '%s\n' 'check.sh accepted a failing Go test' >&2
  exit 1
}
[[ "$output" == *'intentional quality gate failure'* ]] || {
  printf 'test failure was not reported: %s\n' "$output" >&2
  exit 1
}

printf '%s\n' 'quality gate failure-path tests passed'
