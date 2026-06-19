// Package store persists Tallywell's data as an encrypted JSON snapshot and
// serves queries from an in-memory SQLite database rebuilt from that snapshot.
// The on-disk snapshot is the source of truth; SQLite is the query engine.
//
// A non-nil data-encryption key (DEK) encrypts the snapshot at rest (the
// Protected and Convenience tiers). A nil DEK writes a plaintext snapshot (the
// Minimal tier).
package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/tallywell/tallywell/internal/secret"
)

// ErrNotFound indicates no snapshot exists at the given path (first run).
var ErrNotFound = errors.New("store: no snapshot found")

// Store holds the in-memory database and the persistence settings.
type Store struct {
	db   *sql.DB
	path string
	dek  []byte // nil => plaintext snapshot (Minimal tier)
}

// openMemDB opens a fresh in-memory SQLite database. MaxOpenConns(1) keeps the
// single in-memory database alive across queries (each connection would
// otherwise get its own empty database).
func openMemDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// Create initializes a new, empty snapshot at path and returns an open Store.
// dek may be nil for a plaintext (Minimal-tier) snapshot.
func Create(path string, dek []byte) (*Store, error) {
	db, err := openMemDB()
	if err != nil {
		return nil, err
	}
	s := &Store{db: db, path: path, dek: dek}
	if err := emptySnapshot().load(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.persist(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Open loads an existing snapshot from path into a new Store. It returns
// ErrNotFound if the file does not exist, or a decryption error if the DEK is
// wrong (or the data is corrupt).
func Open(path string, dek []byte) (*Store, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	plain := raw
	if dek != nil {
		plain, err = secret.DecryptBlob(dek, raw)
		if err != nil {
			return nil, err
		}
	}

	var snap Snapshot
	if err := json.Unmarshal(plain, &snap); err != nil {
		return nil, fmt.Errorf("store: parse snapshot: %w", err)
	}
	if snap.SchemaVersion > schemaVersion {
		return nil, fmt.Errorf("store: snapshot schema v%d newer than supported v%d", snap.SchemaVersion, schemaVersion)
	}

	db, err := openMemDB()
	if err != nil {
		return nil, err
	}
	if err := snap.load(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db, path: path, dek: dek}, nil
}

// persist writes the current in-memory state to the snapshot file, encrypting
// it when a DEK is set. The write is atomic (temp file + rename).
func (s *Store) persist() error {
	snap, err := snapshotFromDB(s.db)
	if err != nil {
		return err
	}
	plain, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	out := plain
	if s.dek != nil {
		out, err = secret.EncryptBlob(s.dek, plain)
		if err != nil {
			return err
		}
	}
	return atomicWrite(s.path, out)
}

// BackupTo writes an encrypted snapshot of the current state to path. A DEK is
// required — backups are always client-side encrypted (see SECURITY-AND-HIPAA).
func (s *Store) BackupTo(path string, dek []byte) error {
	if dek == nil {
		return errors.New("store: backup requires an encryption key")
	}
	snap, err := snapshotFromDB(s.db)
	if err != nil {
		return err
	}
	plain, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	enc, err := secret.EncryptBlob(dek, plain)
	if err != nil {
		return err
	}
	return atomicWrite(path, enc)
}

// Close releases the in-memory database. Data is already persisted on each
// change; Close does not write.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB exposes the underlying database for read-only queries by other packages
// (e.g. reconcile, report). Callers must not mutate via this handle.
func (s *Store) DB() *sql.DB { return s.db }

// atomicWrite writes data to path via a temp file in the same directory.
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tallywell-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
