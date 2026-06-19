package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tallywell/tallywell/internal/model"
)

func TestDBAccessor(t *testing.T) {
	dir := t.TempDir()
	s, _ := Create(filepath.Join(dir, "t.enc"), nil)
	defer s.Close()
	if s.DB() == nil {
		t.Fatal("DB() returned nil")
	}
	if err := s.DB().Ping(); err != nil {
		t.Fatalf("DB ping: %v", err)
	}
}

func TestOpenCorruptPlaintextFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("this is not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path, nil); err == nil {
		t.Fatal("expected parse error on corrupt snapshot")
	}
}

func TestOpenRejectsNewerSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "future.json")
	snap := Snapshot{SchemaVersion: schemaVersion + 5}
	b, _ := json.Marshal(snap)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path, nil); err == nil {
		t.Fatal("expected error for newer schema version")
	}
}

func TestRecordWithBlankDates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.json")
	s, _ := Create(path, nil)
	// Scheduled session: no paid date; zero DatePaid must round-trip as blank.
	rec := model.Record{ID: "s1", Date: model.MustParseDate("2026-07-01"), PracticeID: "p", PayerID: "y", Status: model.StatusScheduled, Expected: 9000}
	if err := s.PutRecord(rec); err != nil {
		t.Fatal(err)
	}
	s.Close()

	s2, _ := Open(path, nil)
	defer s2.Close()
	recs, _ := s2.Records()
	if len(recs) != 1 || !recs[0].DatePaid.IsZero() {
		t.Fatalf("blank date_paid not round-tripped: %+v", recs)
	}
}
