#!/usr/bin/env bash
# Double-click launcher for macOS. Place this next to the downloaded Tallywell
# binary. It starts Tallywell in the background — a menu-bar icon appears and
# your browser opens automatically. This Terminal window closes itself.
#
# First launch: macOS may warn the app is from an unidentified developer.
# To allow it once: right-click the binary in Finder → Open → confirm.
cd "$(dirname "$0")" || exit 1

arch="$(uname -m)"
case "$arch" in
  arm64)  bin="tallywell-darwin-arm64" ;;
  x86_64) bin="tallywell-darwin-amd64" ;;
  *)      bin="tallywell" ;;
esac
[ -x "./$bin" ] || bin="tallywell"

if [ ! -x "./$bin" ]; then
  osascript -e 'display alert "Tallywell" message "Could not find the Tallywell binary. Make sure the downloaded binary is in the same folder as this launcher." as critical' 2>/dev/null
  exit 1
fi

chmod +x "./$bin" 2>/dev/null || true

# Run in background; logs go to ~/Library/Logs/Tallywell.log.
mkdir -p "$HOME/Library/Logs"
"./$bin" >> "$HOME/Library/Logs/Tallywell.log" 2>&1 &

# Close this Terminal window — the menu-bar icon is the app's UI from here.
sleep 0.5
osascript -e 'tell application "Terminal" to close front window' 2>/dev/null
exit 0
