#!/usr/bin/env bash
set -euo pipefail

mapfile -t packages < <(go list ./... | awk '$0 !~ /\/node_modules\// { print }')

if [ "${#packages[@]}" -eq 0 ]; then
  echo "no Go packages found" >&2
  exit 1
fi

go test "$@" "${packages[@]}"
