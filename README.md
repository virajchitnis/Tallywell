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

- A single, self-contained binary. Double-click it; it opens a private page in
  your browser at `localhost`. Nothing is exposed to the network.
- Add sessions manually, or import a payer's CSV export. Tallywell normalizes the
  different formats, matches sessions to payouts, and flags what's outstanding.
- Export a clean spreadsheet (`.xlsx`) any time — for your own records or your
  accountant.

## Privacy & security

Your data lives only on your machine, encrypted at rest by default. See
[SECURITY-AND-HIPAA.md](SECURITY-AND-HIPAA.md) for the security model, the
configurable protection tiers, and your responsibilities as a covered entity.

Tallywell is **not** an EHR or system of record, and is **not** tax or legal
advice. See [DISCLAIMER](DISCLAIMER).

## Status

Early development. See [ROADMAP.md](ROADMAP.md) for what's planned.

## Building

Requires Go 1.25+. `go build ./...`. A server-agnostic remote-Docker build helper
is in `scripts/remote-build.sh` (see that file for the environment variables it
reads).

## License

[MIT](LICENSE).
