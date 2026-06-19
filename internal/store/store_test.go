package store

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/tallywell/tallywell/internal/model"
	"github.com/tallywell/tallywell/internal/secret"
)

func sampleData(t *testing.T, s *Store) {
	t.Helper()
	if err := s.PutPractice(model.Practice{ID: "pr1", Name: "Own Practice", Kind: model.PracticeOwn}); err != nil {
		t.Fatal(err)
	}
	if err := s.PutPayer(model.Payer{ID: "py1", Name: "Platform A", PracticeID: "pr1", Kind: model.PayerInsurancePlatform, ImporterKey: "platform_a"}); err != nil {
		t.Fatal(err)
	}
	if err := s.PutRate(model.Rate{ID: "rt1", PayerID: "py1", Amount: 12000}); err != nil {
		t.Fatal(err)
	}
	recs := []model.Record{
		{ID: "rc1", Date: model.MustParseDate("2026-06-01"), ClientID: "AB", PracticeID: "pr1", PayerID: "py1", Service: "90837", Status: model.StatusCompleted, Expected: 12000, Source: "manual"},
		{ID: "rc2", Date: model.MustParseDate("2026-06-08"), ClientID: "CD", PracticeID: "pr1", PayerID: "py1", Service: "90837", Status: model.StatusCompleted, Expected: 12000, Paid: 12000, DatePaid: model.MustParseDate("2026-06-20"), Source: "manual"},
	}
	if err := s.PutRecords(recs); err != nil {
		t.Fatal(err)
	}
}

func TestCreatePersistReopenEncrypted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tracker.db.enc")
	dek, _ := secret.GenerateKey()

	s, err := Create(path, dek)
	if err != nil {
		t.Fatal(err)
	}
	sampleData(t, s)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s2, err := Open(path, dek)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()

	prs, _ := s2.Practices()
	pys, _ := s2.Payers()
	rts, _ := s2.Rates()
	recs, _ := s2.Records()
	if len(prs) != 1 || len(pys) != 1 || len(rts) != 1 || len(recs) != 2 {
		t.Fatalf("unexpected counts: practices=%d payers=%d rates=%d records=%d", len(prs), len(pys), len(rts), len(recs))
	}
	if recs[0].ID != "rc1" || recs[1].ID != "rc2" {
		t.Errorf("records not ordered by date: %s,%s", recs[0].ID, recs[1].ID)
	}
	if !recs[1].IsPaid() || recs[1].DatePaid.String() != "2026-06-20" {
		t.Errorf("paid record not round-tripped: %+v", recs[1])
	}
	if pys[0].ImporterKey != "platform_a" || pys[0].Kind != model.PayerInsurancePlatform {
		t.Errorf("payer fields not round-tripped: %+v", pys[0])
	}
}

func TestEncryptedAtRest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tracker.db.enc")
	dek, _ := secret.GenerateKey()
	s, _ := Create(path, dek)
	if err := s.PutRecord(model.Record{ID: "rcX", Date: model.MustParseDate("2026-01-01"), ClientID: "SECRETINITIALS", PracticeID: "p", PayerID: "y", Status: model.StatusCompleted, Expected: 100}); err != nil {
		t.Fatal(err)
	}
	s.Close()

	raw := mustReadFile(t, path)
	if containsBytes(raw, []byte("SECRETINITIALS")) {
		t.Fatal("client identifier found in plaintext in encrypted snapshot")
	}
	if containsBytes(raw, []byte("schema_version")) {
		t.Fatal("snapshot JSON keys visible in encrypted file")
	}
}

func TestWrongKeyFailsToOpen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tracker.db.enc")
	dek, _ := secret.GenerateKey()
	s, _ := Create(path, dek)
	s.Close()

	wrong, _ := secret.GenerateKey()
	if _, err := Open(path, wrong); err == nil {
		t.Fatal("expected error opening with wrong key")
	}
}

func TestOpenMissingReturnsNotFound(t *testing.T) {
	_, err := Open(filepath.Join(t.TempDir(), "nope.enc"), nil)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPlaintextTier(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tracker.db.json")
	s, err := Create(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	sampleData(t, s)
	s.Close()

	// Plaintext snapshot should be readable JSON.
	raw := mustReadFile(t, path)
	if !containsBytes(raw, []byte("schema_version")) {
		t.Fatal("expected readable JSON in plaintext tier")
	}
	s2, err := Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	if recs, _ := s2.Records(); len(recs) != 2 {
		t.Fatalf("expected 2 records, got %d", len(recs))
	}
}

func TestBackupAndRestore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tracker.db.enc")
	backupPath := filepath.Join(dir, "sub", "backup.tallywell.enc")
	dek, _ := secret.GenerateKey()

	s, _ := Create(path, dek)
	sampleData(t, s)
	if err := s.BackupTo(backupPath, dek); err != nil {
		t.Fatal(err)
	}
	s.Close()

	restored, err := Open(backupPath, dek)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	if recs, _ := restored.Records(); len(recs) != 2 {
		t.Fatalf("restore: expected 2 records, got %d", len(recs))
	}

	// Backup without a key is refused.
	s3, _ := Open(path, dek)
	defer s3.Close()
	if err := s3.BackupTo(backupPath, nil); err == nil {
		t.Fatal("expected error backing up without a key")
	}
}

func TestDeleteAndUpdateRecord(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tracker.db.enc")
	dek, _ := secret.GenerateKey()
	s, _ := Create(path, dek)
	sampleData(t, s)

	// Update rc1 to paid.
	updated := model.Record{ID: "rc1", Date: model.MustParseDate("2026-06-01"), ClientID: "AB", PracticeID: "pr1", PayerID: "py1", Service: "90837", Status: model.StatusCompleted, Expected: 12000, Paid: 12000, DatePaid: model.MustParseDate("2026-06-25"), Source: "manual"}
	if err := s.PutRecord(updated); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteRecord("rc2"); err != nil {
		t.Fatal(err)
	}
	s.Close()

	s2, _ := Open(path, dek)
	defer s2.Close()
	recs, _ := s2.Records()
	if len(recs) != 1 {
		t.Fatalf("expected 1 record after delete, got %d", len(recs))
	}
	if !recs[0].IsPaid() {
		t.Error("update to paid did not persist")
	}
}
