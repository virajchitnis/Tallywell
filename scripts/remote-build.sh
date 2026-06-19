#!/usr/bin/env bash
# Build and test Tallywell inside Docker on a remote host over SSH, then fetch
# the cross-compiled binaries back. Server-agnostic: all host details come from
# the environment / a gitignored .env.build (see .env.build.example). No server
# identity is committed.
#
# NOTE: macOS binaries are NOT built here — fyne.io/systray (the menu-bar icon
# library) requires CGO on macOS, which cannot cross-compile from Linux. macOS
# binaries are built on a macOS runner in CI (release.yml) or locally on a Mac
# with `go build .`. This script covers windows/amd64 and linux/amd64 only.
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"

# Load .env.build if present.
if [[ -f .env.build ]]; then
  set -a
  # shellcheck disable=SC1091
  . ./.env.build
  set +a
fi

: "${TALLYWELL_BUILD_HOST:?set TALLYWELL_BUILD_HOST (e.g. user@your-host) in .env.build}"
: "${TALLYWELL_BUILD_DIR:=tallywell-build}"
: "${TALLYWELL_GO_IMAGE:=golang:1.25}"

# macOS is intentionally absent — see note above.
targets=(
  "windows/amd64"
  "linux/amd64"
)

echo "==> Syncing source to $TALLYWELL_BUILD_HOST:$TALLYWELL_BUILD_DIR/src"
ssh -o BatchMode=yes "$TALLYWELL_BUILD_HOST" "mkdir -p '$TALLYWELL_BUILD_DIR/src'"
rsync -az --delete \
  --exclude '.git' --exclude 'dist' --exclude '.env.build' --exclude '.denylist.local' \
  ./ "$TALLYWELL_BUILD_HOST:$TALLYWELL_BUILD_DIR/src/"

# Build the cross-compile loop executed inside the container.
# go mod tidy runs first so go.sum stays current after dependency changes.
# Linux systray needs libayatana-appindicator3 headers for CGO.
build_cmds='set -e;'
build_cmds+=' apt-get update -qq && apt-get install -y -qq libayatana-appindicator3-dev 2>/dev/null || true;'
build_cmds+=' go mod tidy;'
build_cmds+=' go test ./...;'
for t in "${targets[@]}"; do
  os="${t%/*}"; arch="${t#*/}"
  out="dist/tallywell-${os}-${arch}"
  [[ "$os" == "windows" ]] && out="${out}.exe"
  # Windows: pure-Go systray (Win32 syscalls), CGO_ENABLED=0 fine.
  # Linux: CGO required for appindicator; build natively in the container.
  if [[ "$os" == "windows" ]]; then
    build_cmds+=" echo building ${os}/${arch}; CGO_ENABLED=0 GOOS=${os} GOARCH=${arch} go build -o ${out} .;"
  else
    build_cmds+=" echo building ${os}/${arch}; GOOS=${os} GOARCH=${arch} go build -o ${out} .;"
  fi
done

echo "==> Building + testing in $TALLYWELL_GO_IMAGE"
ssh -o BatchMode=yes "$TALLYWELL_BUILD_HOST" \
  "docker run --rm -v \$HOME/$TALLYWELL_BUILD_DIR/src:/src -v tallywell-gocache:/go -w /src $TALLYWELL_GO_IMAGE bash -c '$build_cmds'"

echo "==> Fetching binaries and updated go.sum to ./dist and ./"
mkdir -p dist
rsync -az "$TALLYWELL_BUILD_HOST:$TALLYWELL_BUILD_DIR/src/dist/" ./dist/
# Sync go.mod and go.sum back so new dependency checksums are committed.
rsync -az "$TALLYWELL_BUILD_HOST:$TALLYWELL_BUILD_DIR/src/go.mod" ./go.mod
rsync -az "$TALLYWELL_BUILD_HOST:$TALLYWELL_BUILD_DIR/src/go.sum" ./go.sum

( cd dist && shasum -a 256 tallywell-* > SHA256SUMS 2>/dev/null || sha256sum tallywell-* > SHA256SUMS )
echo "==> Done. Artifacts:"
ls -1 dist/
echo ""
echo "NOTE: go.mod and go.sum have been synced back. If they changed, commit them."
