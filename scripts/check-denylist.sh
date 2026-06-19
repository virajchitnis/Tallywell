#!/usr/bin/env bash
# Commit-hygiene guard: fail if staged changes contain any denylisted term.
#
# Denylist terms live in .denylist.local (gitignored) so the forbidden strings
# themselves never enter the repo. A committed .denylist.local.example documents
# the format. Used both as a pre-commit hook (see scripts/install-hooks.sh) and
# in CI (where the denylist is supplied via the TALLYWELL_DENYLIST env var, one
# term per line, since .denylist.local is not committed).
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
denylist_file="$repo_root/.denylist.local"

terms=()
if [[ -f "$denylist_file" ]]; then
  while IFS= read -r line; do
    line="${line%%$'\r'}"
    [[ -z "$line" || "$line" == \#* ]] && continue
    terms+=("$line")
  done <"$denylist_file"
elif [[ -n "${TALLYWELL_DENYLIST:-}" ]]; then
  while IFS= read -r line; do
    [[ -z "$line" || "$line" == \#* ]] && continue
    terms+=("$line")
  done <<<"$TALLYWELL_DENYLIST"
else
  echo "check-denylist: no .denylist.local and no TALLYWELL_DENYLIST set; skipping." >&2
  echo "  (copy .denylist.local.example to .denylist.local to enable local protection)" >&2
  exit 0
fi

if [[ ${#terms[@]} -eq 0 ]]; then
  exit 0
fi

# What to scan: staged diff for a commit; full tree in CI.
if [[ "${1:-}" == "--all" ]]; then
  payload="$(git -C "$repo_root" grep -rIni --no-color -e . -- . || true)"
else
  payload="$(git -C "$repo_root" diff --cached --no-color || true)"
fi

found=0
for term in "${terms[@]}"; do
  if grep -iqF -- "$term" <<<"$payload"; then
    echo "check-denylist: BLOCKED — staged changes contain a denylisted term." >&2
    found=1
  fi
done

if [[ $found -ne 0 ]]; then
  echo "Remove the personal/infrastructure identifier(s) before committing." >&2
  exit 1
fi
