package store

import (
	"path/filepath"
	"testing"

	"github.com/tallywell/tallywell/internal/model"
)

// TestOperationsOnClosedDB exercises the error branches of the query and
// mutation helpers by running them against a closed database.
func TestOperationsOnClosedDB(t *testing.T) {
	dir := t.TempDir()
	s, err := Create(filepath.Join(dir, "t.json"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := s.Practices(); err == nil {
		t.Error("Practices on closed db should error")
	}
	if _, err := s.Payers(); err == nil {
		t.Error("Payers on closed db should error")
	}
	if _, err := s.Rates(); err == nil {
		t.Error("Rates on closed db should error")
	}
	if _, err := s.Records(); err == nil {
		t.Error("Records on closed db should error")
	}
	if err := s.PutPractice(model.Practice{ID: "x"}); err == nil {
		t.Error("PutPractice on closed db should error")
	}
	if err := s.PutPayer(model.Payer{ID: "x"}); err == nil {
		t.Error("PutPayer on closed db should error")
	}
	if err := s.PutRate(model.Rate{ID: "x"}); err == nil {
		t.Error("PutRate on closed db should error")
	}
	if err := s.PutRecord(model.Record{ID: "x", Status: model.StatusScheduled}); err == nil {
		t.Error("PutRecord on closed db should error")
	}
	if err := s.DeleteRecord("x"); err == nil {
		t.Error("DeleteRecord on closed db should error")
	}
}
