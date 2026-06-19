#!/usr/bin/env bash
# Build and test Tallywell inside Docker on a remote host over SSH, then fetch
# the cross-compiled binaries back. Server-agnostic: all host details come from
# the environment / a gitignored .env.build (see .env.build.example). No server
# identity is committed.
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

targets=(
  "darwin/arm64"
  "darwin/amd64"
  "windows/amd64"
  "linux/amd64"
)

echo "==> Syncing source to $TALLYWELL_BUILD_HOST:$TALLYWELL_BUILD_DIR/src"
ssh -o BatchMode=yes "$TALLYWELL_BUILD_HOST" "mkdir -p '$TALLYWELL_BUILD_DIR/src'"
rsync -az --delete \
  --exclude '.git' --exclude 'dist' --exclude '.env.build' --exclude '.denylist.local' \
  ./ "$TALLYWELL_BUILD_HOST:$TALLYWELL_BUILD_DIR/src/"

# Build the cross-compile loop executed inside the container.
build_cmds='set -e; go test ./...;'
for t in "${targets[@]}"; do
  os="${t%/*}"; arch="${t#*/}"
  out="dist/tallywell-${os}-${arch}"
  [[ "$os" == "windows" ]] && out="${out}.exe"
  build_cmds+=" echo building ${os}/${arch}; CGO_ENABLED=0 GOOS=${os} GOARCH=${arch} go build -o ${out} .;"
done

echo "==> Building + testing in $TALLYWELL_GO_IMAGE"
ssh -o BatchMode=yes "$TALLYWELL_BUILD_HOST" \
  "docker run --rm -v \$HOME/$TALLYWELL_BUILD_DIR/src:/src -v tallywell-gocache:/go -w /src $TALLYWELL_GO_IMAGE bash -c '$build_cmds'"

echo "==> Fetching binaries to ./dist"
mkdir -p dist
rsync -az "$TALLYWELL_BUILD_HOST:$TALLYWELL_BUILD_DIR/src/dist/" ./dist/
( cd dist && shasum -a 256 tallywell-* > SHA256SUMS 2>/dev/null || sha256sum tallywell-* > SHA256SUMS )
echo "==> Done. Artifacts:"
ls -1 dist/
