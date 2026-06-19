#!/usr/bin/env bash
# Point git at the repo's tracked hooks (.githooks/) for this clone.
set -euo pipefail
repo_root="$(git rev-parse --show-toplevel)"
git -C "$repo_root" config core.hooksPath .githooks
chmod +x "$repo_root/.githooks/"* "$repo_root/scripts/"*.sh 2>/dev/null || true
echo "Installed git hooks (core.hooksPath -> .githooks)."
echo "Tip: copy .denylist.local.example to .denylist.local to enable the commit-hygiene guard."
