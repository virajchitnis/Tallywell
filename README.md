# Tallywell

**Local-first income & session tracking for therapists.** No servers, no
accounts, no subscription — your client data never leaves your computer.

Tallywell helps a therapist who sees clients across several payers — insurance
platforms (Alma, Headway), private pay (SimplePractice), and/or a W-2 employer —
keep one clear picture of **sessions seen, money earned, and money still owed**,
without logging into four portals and reconciling by hand.

## Why it exists

A session you ran shows up in a payer's payout export weeks later, in a different
layout for every platform. Merging those exports and matching "session → payout"
by hand each month is tedious and error-prone. Tallywell does the reconciliation
for you and shows the result in one calm dashboard.

## How it works

- A single, self-contained binary. Double-click it; it opens in its own native
  application window. Nothing is exposed to the network.
- Add sessions manually, or import a payer's CSV export. Tallywell normalizes the
  different formats, matches sessions to payouts, and flags what's outstanding.
- Export a clean spreadsheet (`.xlsx`) any time — for your own records or your
  accountant.
- Adapts to your system's **light or dark mode** automatically.
- Optional **auto-unlock** via the OS keychain (macOS Keychain, Windows
  Credential Manager, libsecret on Linux) — no passphrase prompt at launch,
  data stays encrypted, passphrase always works as a fallback.
- Need to start over? A built-in reset option deletes all your data and returns
  to first-run state, with step-by-step uninstall instructions.
- **Version info and update check** in Settings — see your installed version and
  check GitHub for a newer release on demand (no background network calls).

## Privacy & security

Your data lives only on your machine, encrypted at rest by default. See
[SECURITY-AND-HIPAA.md](SECURITY-AND-HIPAA.md) for the security model, the
configurable protection tiers, and your responsibilities as a covered entity.

Tallywell is **not** an EHR or system of record, and is **not** tax or legal
advice. See [DISCLAIMER](DISCLAIMER).

## Status

Early development. See [ROADMAP.md](ROADMAP.md) for what's planned.

## Building

Requires Go 1.25+. The UI runs inside a native WebView window (WKWebView on
macOS, WebKitGTK on Linux), so CGO is required on non-Windows platforms.

```bash
go build ./...
```

To build the macOS `.app` bundle and DMG (must run on a Mac):

```bash
bash scripts/build-mac-app.sh
```

Linux builds require `libwebkit2gtk-4.1-dev` (Ubuntu/Debian) or the equivalent
for your distribution. Windows builds use `CGO_ENABLED=0` and open the default
browser as a fallback.

To run locally without the native window (headless mode, useful for testing):

```bash
TALLYWELL_NO_TRAY=1 go run .
```

## License

[MIT](LICENSE).
