package store

// schemaVersion is the current logical schema version of the snapshot. It is
// stored in the snapshot and the meta table so future versions can migrate.
const schemaVersion = 1

// ddl creates the in-memory schema. The in-memory SQLite database is a query
// engine rebuilt from the (decrypted) snapshot on every load; the encrypted
// JSON snapshot on disk is the source of truth.
const ddl = `
CREATE TABLE meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
CREATE TABLE practices (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    kind TEXT NOT NULL
);
CREATE TABLE payers (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    practice_id  TEXT NOT NULL,
    kind         TEXT NOT NULL,
    importer_key TEXT NOT NULL DEFAULT ''
);
CREATE TABLE rates (
    id       TEXT PRIMARY KEY,
    payer_id TEXT NOT NULL,
    service  TEXT NOT NULL DEFAULT '',
    amount   INTEGER NOT NULL
);
CREATE TABLE records (
    id          TEXT PRIMARY KEY,
    date        TEXT NOT NULL,
    client_id   TEXT NOT NULL DEFAULT '',
    practice_id TEXT NOT NULL,
    payer_id    TEXT NOT NULL,
    service     TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL,
    expected    INTEGER NOT NULL DEFAULT 0,
    paid        INTEGER NOT NULL DEFAULT 0,
    date_paid   TEXT NOT NULL DEFAULT '',
    source      TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_records_payer ON records(payer_id);
CREATE INDEX idx_records_date  ON records(date);
`
