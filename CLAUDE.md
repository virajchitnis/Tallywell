# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

Go and `gh` may not be on `PATH`. On the maintainer's machine both live in the directory above the repo root (`../go/bin` and `../gh/bin`). Find them with `which go` / `which gh` or add to your shell profile.

**Test (all):**
```bash
go test ./...
```

**Test (single package):**
```bash
go test ./internal/reconcile/
```

**Test (single test):**
```bash
go test ./internal/reconcile/ -run TestMatch
```

**Race detector + coverage:**
```bash
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

**Coverage gate (overall ≥80%, core packages ≥90%):**
```bash
bash scripts/coverage-gate.sh
```

**Lint:**
```bash
golangci-lint run
```

**Build (macOS .app bundle, universal arm64+amd64):**
```bash
bash scripts/build-mac-app.sh
```

**Run locally (headless, no tray):**
```bash
TALLYWELL_NO_TRAY=1 go run .
```

**E2E smoke tests (Playwright):**
```bash
go build -o dist/tallywell-local .
TALLYWELL_BIN=dist/tallywell-local npx playwright test --project=chromium
```
Run from the `e2e/` directory after `npm install`.

**Install pre-commit hook (denylist guard):**
```bash
bash scripts/install-hooks.sh
```

## Architecture

Tallywell is a local-first, loopback-only web app: a Go HTTP server opens in the user's default browser. There is no external network traffic and no accounts. All sensitive data is encrypted at rest on disk.

### Lifecycle layers (`internal/app` → `internal/secret` → `internal/store`)

The three layers form a strict stack:

1. **`internal/secret`** — pure crypto primitives. `EncryptBlob`/`DecryptBlob` (AES-256-GCM). `Envelope` is a multi-wrap key envelope: a random DEK is generated once; each unlock method (currently only passphrase via Argon2id) stores a separate `WrappedKey` that encrypts the DEK. Adding a new unlock method (OS keychain, passkey) just adds a new `WrappedKey`.

2. **`internal/store`** — persistence. The on-disk format is an encrypted JSON blob (`tracker.db.enc`). On `Open`, the blob is decrypted and loaded into an **in-memory SQLite** database (`:memory:`, `modernc.org/sqlite`). Every mutation calls `persist()` to write an updated encrypted snapshot atomically (temp file + rename). `Store.DB()` exposes the `*sql.DB` for read-only use by other packages.

3. **`internal/app`** — lifecycle state machine. Three phases: `PhaseNeedsSetup` (no envelope yet) → `PhaseLocked` (envelope exists, DEK not in memory) → `PhaseUnlocked` (store open). `Setup` creates the envelope + empty store. `Unlock` derives the DEK from the passphrase, decrypts the snapshot, and opens the store. `Lock` calls `store.Close()` and zeroes the DEK bytes.

### HTTP server (`internal/server`)

`server.New` takes an `*app.App` and returns a `*Server`. All app-data routes are wrapped in `guard()`, which checks the app phase, validates a session cookie (`tw_session`), and enforces a 15-minute idle auto-lock. No authentication middleware — the guard is on every data handler directly.

Templates are `go:embed`ded from `internal/server/web/templates/*.html` and parsed at startup. Static assets are served from `internal/server/web/static/`. `layout.html` defines `{{template "layout" .}}` used by all app pages; `setup.html` and `unlock.html` use `bare_start`/`bare_end` blocks (no nav).

`SetQuitFunc` injects the shutdown callback after the server is built (to avoid a circular dependency between the server constructor and the `http.Server`).

### `main.go`

Thin wiring only. Binds `127.0.0.1:0` (random port). On macOS, `tray.Run` must block the **main goroutine** (Cocoa requirement), so the HTTP server runs in a goroutine. `TALLYWELL_NO_TRAY=1` skips the tray and blocks on a `select` over SIGINT/SIGTERM and a `done` channel that the web UI Quit button closes.

### Data model (`internal/model`)

`Practice` and `Payer` are user-defined (nothing is hardcoded). A `Record` is the normalized unit: one row covers both a session and its eventual payout. Key fields: `ClientID` (initials/opaque ID — no names), `PracticeID`, `PayerID`, `Service`, `Status`, `Expected`, `Paid`, `DatePaid`, `Source`.

`Money` is stored as integer cents. `Date` is a `YYYY-MM-DD` string type with custom JSON marshaling.

### Reconciliation (`internal/reconcile`)

`Match(sessions, payouts, windowDays)` merges imported payout rows into existing sessions. Match criteria: same `ClientID` + `Service`, within ±45 days (default), session not already paid. Unmatched payouts are returned separately (never silently dropped).

### Report (`internal/report`)

`Write(db, path)` generates a `.xlsx` workbook using `excelize/v2`. Four sheets: Sessions, Dashboard, Tax Summary, Unmatched/Review.

### System tray (`internal/tray`)

`fyne.io/systray` — CGO on macOS (must build on Mac), pure Win32 syscalls on Windows, appindicator CGO on Linux. The tray icon is a 22×22 "T" glyph rendered programmatically as a template icon (auto-adapts to macOS dark/light mode).

### macOS .app bundle

`scripts/build-mac-app.sh` builds a universal binary (arm64 + amd64 via `lipo`), generates `AppIcon.icns` from `scripts/gen-icon.go` via `sips`/`iconutil`, and writes `Info.plist` with `LSUIElement=true` (no Dock icon) and `CFBundleIconFile`. Both architectures must be built with `MACOSX_DEPLOYMENT_TARGET=11.0` so the binary runs on macOS 11+ (not just the build machine's OS version). The Intel slice requires `CGO_ENABLED=1 CC="cc -arch x86_64"`.

## Commit hygiene

**Denylist:** personal names (therapist, practice, employer) and the build server hostname must never appear in any committed file. The pre-commit hook runs `scripts/check-denylist.sh` against the staged diff. In CI the terms are supplied via `TALLYWELL_DENYLIST` env var. Locally, copy `.denylist.local.example` to `.denylist.local` (gitignored) and add the real terms.

Use Conventional Commits style (`feat:`, `fix:`, `test:`, `refactor:`, `docs:`, `ci:`). Commit frequently — one logical unit per commit.

## Importers (not yet built)

The `internal/importers/` package is the next major feature. Each importer maps a platform's CSV export to `[]model.Record`. The `ImporterKey` field on `Payer` (e.g. `"alma"`, `"headway"`, `"simplepractice"`) identifies which importer to run. Importers drop any column containing PHI (names, addresses, notes) on ingestion. Test fixtures go in `testdata/` and must use synthetic data.
