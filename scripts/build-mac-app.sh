#!/usr/bin/env bash
# Build Tallywell.app — a self-contained macOS application bundle with a
# universal binary (Apple Silicon + Intel). Run on a Mac with Go installed.
#
# Output:
#   dist/Tallywell.app   — drag to /Applications on any Mac, or AirDrop direct
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"

APP="Tallywell"
BUNDLE_ID="io.tallywell.tallywell"
# Read version from git tag if available, fall back to "dev".
VERSION="$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "dev")"

APP_DIR="dist/${APP}.app"
CONTENTS="${APP_DIR}/Contents"
MACOS="${CONTENTS}/MacOS"
RESOURCES="${CONTENTS}/Resources"

echo "==> Cleaning previous build"
rm -rf "$APP_DIR"
mkdir -p "$MACOS" "$RESOURCES"

echo "==> Building arm64 (Apple Silicon)"
go build -o "${MACOS}/${APP}-arm64" .

echo "==> Building amd64 (Intel)"
CGO_ENABLED=1 CC="cc -arch x86_64" GOARCH=amd64 go build -o "${MACOS}/${APP}-amd64" .

echo "==> Creating universal binary with lipo"
lipo -create \
  "${MACOS}/${APP}-arm64" \
  "${MACOS}/${APP}-amd64" \
  -output "${MACOS}/${APP}"
rm "${MACOS}/${APP}-arm64" "${MACOS}/${APP}-amd64"
chmod +x "${MACOS}/${APP}"

echo "==> Writing Info.plist"
cat > "${CONTENTS}/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key>             <string>${APP}</string>
  <key>CFBundleDisplayName</key>      <string>${APP}</string>
  <key>CFBundleIdentifier</key>       <string>${BUNDLE_ID}</string>
  <key>CFBundleVersion</key>          <string>${VERSION}</string>
  <key>CFBundleShortVersionString</key><string>${VERSION}</string>
  <key>CFBundleExecutable</key>       <string>${APP}</string>
  <key>CFBundlePackageType</key>      <string>APPL</string>
  <key>NSHighResolutionCapable</key>  <true/>
  <key>LSUIElement</key>              <true/>
</dict>
</plist>
PLIST

# Strip quarantine so macOS Gatekeeper does not block on the recipient's Mac.
# (AirDrop between personal Macs does not re-add it.)
xattr -cr "$APP_DIR" 2>/dev/null || true

echo ""
echo "==> Done: dist/Tallywell.app  ($(du -sh "$APP_DIR" | cut -f1))"
echo "    AirDrop it to her Mac — she can double-click it from anywhere."
echo "    To install permanently: drag it to /Applications."
