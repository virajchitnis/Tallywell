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
VERSION="$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "dev")"

APP_DIR="dist/${APP}.app"
CONTENTS="${APP_DIR}/Contents"
MACOS="${CONTENTS}/MacOS"
RESOURCES="${CONTENTS}/Resources"
ICONSET="dist/${APP}.iconset"

echo "==> Cleaning previous build"
rm -rf "$APP_DIR" "$ICONSET" dist/tallywell-icon-1024.png "dist/${APP}.icns"
mkdir -p "$MACOS" "$RESOURCES" "$ICONSET"

# ── Icon ──────────────────────────────────────────────────────────────────────

echo "==> Generating icon (1024×1024)"
go run scripts/gen-icon.go

echo "==> Building iconset (all macOS sizes)"
# sips resizes the 1024px source; iconutil assembles the .icns.
make_size() { sips -z "$1" "$1" dist/tallywell-icon-1024.png --out "${ICONSET}/$2" >/dev/null; }
make_size 16   icon_16x16.png
make_size 32   "icon_16x16@2x.png"
make_size 32   icon_32x32.png
make_size 64   "icon_32x32@2x.png"
make_size 128  icon_128x128.png
make_size 256  "icon_128x128@2x.png"
make_size 256  icon_256x256.png
make_size 512  "icon_256x256@2x.png"
make_size 512  icon_512x512.png
make_size 1024 "icon_512x512@2x.png"
iconutil -c icns "$ICONSET" -o "dist/${APP}.icns"

# ── Binaries ──────────────────────────────────────────────────────────────────

LDFLAGS="-X main.version=v${VERSION}"

echo "==> Building arm64 (Apple Silicon)"
MACOSX_DEPLOYMENT_TARGET=11.0 go build -ldflags "${LDFLAGS}" -o "${MACOS}/${APP}-arm64" .

echo "==> Building amd64 (Intel)"
MACOSX_DEPLOYMENT_TARGET=11.0 CGO_ENABLED=1 CC="cc -arch x86_64" GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "${MACOS}/${APP}-amd64" .

echo "==> Creating universal binary with lipo"
lipo -create \
  "${MACOS}/${APP}-arm64" \
  "${MACOS}/${APP}-amd64" \
  -output "${MACOS}/${APP}"
rm "${MACOS}/${APP}-arm64" "${MACOS}/${APP}-amd64"
chmod +x "${MACOS}/${APP}"

# ── Bundle metadata ───────────────────────────────────────────────────────────

echo "==> Copying icon and writing Info.plist"
cp "dist/${APP}.icns" "${RESOURCES}/AppIcon.icns"

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
  <key>CFBundleIconFile</key>         <string>AppIcon</string>
  <key>NSHighResolutionCapable</key>  <true/>
  <key>LSMinimumSystemVersion</key>   <string>11.0</string>
</dict>
</plist>
PLIST

# Strip quarantine so macOS Gatekeeper does not block on the recipient's Mac
# when sent via AirDrop between personal Macs.
xattr -cr "$APP_DIR" 2>/dev/null || true

# ── DMG ───────────────────────────────────────────────────────────────────────

echo "==> Creating DMG"
DMG="dist/${APP}-${VERSION}.dmg"

# Stage: app bundle + Applications symlink so the user can drag-to-install.
staging=$(mktemp -d)
cp -R "$APP_DIR" "$staging/"
ln -s /Applications "$staging/Applications"

hdiutil create \
  -volname "$APP" \
  -srcfolder "$staging" \
  -ov \
  -format UDZO \
  "$DMG"

rm -rf "$staging"
xattr -cr "$DMG" 2>/dev/null || true

echo ""
echo "==> Done"
echo "    App bundle : dist/Tallywell.app  ($(du -sh "$APP_DIR" | cut -f1))"
echo "    Disk image : $DMG  ($(du -sh "$DMG" | cut -f1))"
echo ""
echo "    Share the .dmg — open it, drag Tallywell to Applications, done."
