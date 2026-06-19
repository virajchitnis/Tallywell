# Security & HIPAA considerations

*This document explains Tallywell's security model and your responsibilities. It
is not legal advice. Software is never "HIPAA compliant" on its own — compliance
is a property of your practices.*

## Data Tallywell stores

To minimize risk, Tallywell stores only what it needs to track money and
caseload: a client **initial/ID**, session **date**, **payer**, **service
code**, and **dollar amounts**. It does not store client names, notes,
diagnoses, or addresses. Treat all of it as PHI regardless — initials plus exact
dates are not de-identified under HIPAA Safe Harbor.

## Local-first: no business associate

Tallywell runs entirely on your computer. It has no server and sends nothing to
the maintainers, so the developers never receive your PHI and are **not your
business associate** — the same way a locally installed spreadsheet program
isn't. There is no account to create and nothing to "log in" to over the network.

## At-rest encryption and protection tiers

By default your database is **encrypted at rest** with a key derived from a
passphrase you choose; it is only decrypted into memory while the app is
unlocked. You pick a protection tier at setup (changeable later):

- **Protected (default):** passphrase to unlock, encrypted at rest, auto-lock
  after inactivity. Works even without full-disk encryption.
- **Convenience:** still encrypted at rest, but the key is held in your operating
  system keychain so the app unlocks transparently; access is gated by your OS
  login.
- **Minimal:** no app encryption; relies entirely on your OS account and
  full-disk encryption. Lowest friction, weakest protection.

**If you lose your passphrase, your data cannot be recovered** — it is the
encryption key, and there is no server to reset it.

## Backups

Backups are **encrypted on your device before they leave it**, so they can be
placed in a cloud-synced folder (e.g. iCloud Drive) without the cloud provider
being able to read them — and therefore without needing a business associate
agreement with that provider. Plain (unencrypted) spreadsheet exports are written
locally; only copy them to the cloud if you accept that they contain readable PHI.

## Your responsibilities (a short checklist)

- Enable full-disk encryption (FileVault on macOS, BitLocker on Windows).
- Use a strong OS login password and screen auto-lock.
- Choose a strong Tallywell passphrase and store it safely.
- Perform your own HIPAA risk analysis for your practice.
- Keep your own backups; verify you can restore them.
