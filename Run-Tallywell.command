#!/usr/bin/env bash
# Double-click launcher for macOS. Place this next to the downloaded Tallywell
# binary. It picks the binary matching your Mac and starts the app, which opens
# your browser automatically.
#
# First launch: macOS may warn the app is from an unidentified developer (the
# build is not yet notarized). To allow it once: right-click the binary in
# Finder, choose Open, then confirm. After that it runs normally.
cd "$(dirname "$0")" || exit 1

arch="$(uname -m)"
case "$arch" in
  arm64) bin="tallywell-darwin-arm64" ;;
  x86_64) bin="tallywell-darwin-amd64" ;;
  *) bin="tallywell" ;;
esac
[ -x "./$bin" ] || bin="tallywell"

if [ ! -x "./$bin" ]; then
  echo "Could not find the Tallywell binary next to this launcher."
  echo "Make sure the downloaded binary is in the same folder."
  read -r -p "Press return to close."
  exit 1
fi

chmod +x "./$bin" 2>/dev/null || true
exec "./$bin"
