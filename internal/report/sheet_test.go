package report

import (
	"errors"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestSheetErrorShortCircuits(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	s := &sheet{f: f}

	// An invalid sheet name (contains ':') makes NewSheet fail and sets s.err.
	s.use("Invalid:Name")
	if s.err == nil {
		t.Fatal("expected error for invalid sheet name")
	}

	// Once errored, further calls must short-circuit without panicking.
	s.use("Another")
	s.row("value")
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func TestWriteXLSXWriterError(t *testing.T) {
	if err := WriteXLSX(failWriter{}, sampleInput()); err == nil {
		t.Fatal("expected error from failing writer")
	}
}
