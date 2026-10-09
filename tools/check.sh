#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "$0")" && pwd)
project_root=$(cd -- "${1:-$script_dir/..}" && pwd)
cd "$project_root"

printf '%s\n' '[1/5] gofmt'
go_files=()
while IFS= read -r -d '' path; do
  go_files+=("$path")
done < <(find . -type d \( -name .git -o -name .venv -o -name node_modules -o -name vendor \) -prune -o -type f -name '*.go' -print0)
if ((${#go_files[@]})); then
  unformatted=$(gofmt -l "${go_files[@]}")
  if [[ -n "$unformatted" ]]; then
    printf 'Go files need gofmt:\n%s\n' "$unformatted" >&2
    exit 1
  fi
fi

printf '%s\n' '[2/5] secret scan'
python3 "$script_dir/check_secrets.py" "$project_root"

printf '%s\n' '[3/5] tests'
make test

go_buildvcs_flags=()
if ! git -C "$project_root" rev-parse --show-toplevel >/dev/null 2>&1; then
  go_buildvcs_flags=(-buildvcs=false)
fi

printf '%s\n' '[4/5] go vet'
go vet "${go_buildvcs_flags[@]}" ./...

printf '%s\n' '[5/5] go build'
go build "${go_buildvcs_flags[@]}" ./...

printf '%s\n' 'quality checks passed'
