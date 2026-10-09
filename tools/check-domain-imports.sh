#!/usr/bin/env bash
set -euo pipefail

go_cmd=${GO:-go}
module_prefix=$("$go_cmd" list -buildvcs=false -m)
violations=0
package_lines=$("$go_cmd" list -buildvcs=false -f '{{.ImportPath}} {{join .Imports " "}}' ./internal/...)
while IFS= read -r package_line; do
  package=${package_line%% *}
  imports=${package_line#* }
  [[ "$package" == */domain ]] || continue
  for imported in $imports; do
    if [[ "$imported" == "$module_prefix/internal/infrastructure"* ]]; then
      printf 'domain package %s imports infrastructure package %s\n' "$package" "$imported" >&2
      violations=1
    fi
  done
done <<< "$package_lines"

if (( violations != 0 )); then
  exit 1
fi
printf '%s\n' 'domain package import boundaries passed'
