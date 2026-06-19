package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tallywell/tallywell/internal/secret"
)

// badPath builds a path whose parent directory cannot be created because a
// regular file sits where a directory component must be. MkdirAll then fails,
// exercising the write error paths.
func badPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	fileInTheWay := filepath.Join(dir, "afile")
	if err := os.WriteFile(fileInTheWay, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(fileInTheWay, "sub", "tracker.enc")
}

func TestCreateFailsOnUnwritablePath(t *testing.T) {
	if _, err := Create(badPath(t), nil); err == nil {
		t.Fatal("expected Create to fail when the snapshot path is unwritable")
	}
}

func TestPersistFailsOnUnwritablePathAfterMutation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.json")
	s, err := Create(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	// Repoint at an unwritable location so the next persist fails.
	s.path = badPath(t)
	if err := s.PutPractice(samplePractice()); err == nil {
		t.Fatal("expected persist failure to surface from PutPractice")
	}
}

func TestBackupFailsOnUnwritablePath(t *testing.T) {
	dir := t.TempDir()
	dek, _ := secret.GenerateKey()
	s, _ := Create(filepath.Join(dir, "t.enc"), dek)
	defer s.Close()
	if err := s.BackupTo(badPath(t), dek); err == nil {
		t.Fatal("expected BackupTo to fail on unwritable path")
	}
}
