# Roadmap

Human-readable overview of planned work. Once the repo is public, each item is
mirrored to a GitHub Issue with labels (`enhancement`, `importer`, `security`,
`distribution`, `testing`) and a milestone.

## Now (v1)

- Core local app: manual session entry, rates, dashboard, outstanding tracking.
- Encrypted-at-rest storage with protection tiers (Protected / Convenience / Minimal).
- xlsx export (Sessions / Dashboard / Tax summary / Unmatched).
- CSV importers for the first payers (built from real sample exports).
- Cross-platform release (macOS arm64/amd64, Windows, Linux) + Pages download page.

## Planned

| Item | Theme | Priority |
|---|---|---|
| Community importers: Grow Therapy, Rula, Thriving, etc. | importer | high |
| Passkey / Touch ID unlock via WebAuthn PRF (added DEK wrap) | security | medium (fast-follow) |
| Apple Developer signing + notarization (remove Gatekeeper warning) | distribution | medium |
| Random client codes option (vs. initials) to lower identifiability | security | medium |
| Optional printable recovery key (fallback for lost passphrase) | security | medium |
| Change-passphrase flow (re-wrap data key) | security | medium |
| Employer comp model: salary vs per-session handling | feature | medium |
| Homebrew tap + documented `go install` | distribution | low |
| App-level DB encryption (SQLCipher) reconsideration | security | low |
| Mobile / phone access story | feature | low |
| Opt-in encrypted cloud copy of plain exports | feature | low |
| Expert-determination de-identification mode | security | low |
| macOS-runner CI smoke job (real binary + E2E) | testing | low |
| Native desktop app (graduate from local web UI) | feature | low / maybe-never |
