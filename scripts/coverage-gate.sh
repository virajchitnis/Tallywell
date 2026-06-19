#!/usr/bin/env bash
# Enforce Tallywell's coverage gate: overall >= 80%, and >= 90% on the core
# logic packages (reconcile, report, secret; importers once added). Run from the
# repo root.
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"

overall_min=80
core_min=90
core_pkgs=("internal/reconcile" "internal/report" "internal/secret" "internal/importers")

fail=0

echo "==> Overall coverage"
go test ./... -coverpkg=./internal/... -coverprofile=coverage.out >/dev/null
overall="$(go tool cover -func=coverage.out | awk '/^total:/ {gsub("%","",$3); print $3}')"
echo "Overall: ${overall}% (min ${overall_min}%)"
awk -v v="$overall" -v m="$overall_min" 'BEGIN{exit !(v+0 >= m)}' || {
  echo "  FAIL: overall coverage below ${overall_min}%"; fail=1; }

echo "==> Core packages"
for pkg in "${core_pkgs[@]}"; do
  [[ -d "$pkg" ]] || continue
  line="$(go test "./$pkg/" -cover 2>/dev/null | tail -1)"
  pct="$(printf '%s\n' "$line" | grep -oE 'coverage: [0-9.]+%' | grep -oE '[0-9.]+' || true)"
  [[ -z "$pct" ]] && { echo "  WARN: no coverage reported for $pkg"; continue; }
  echo "${pkg}: ${pct}% (min ${core_min}%)"
  awk -v v="$pct" -v m="$core_min" 'BEGIN{exit !(v+0 >= m)}' || {
    echo "  FAIL: ${pkg} below ${core_min}%"; fail=1; }
done

[[ $fail -eq 0 ]] && echo "Coverage gate passed." || echo "Coverage gate FAILED."
exit $fail
