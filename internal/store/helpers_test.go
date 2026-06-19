package store

import (
	"bytes"
	"os"
	"testing"

	"github.com/tallywell/tallywell/internal/model"
)

func samplePractice() model.Practice {
	return model.Practice{ID: "pr1", Name: "Own Practice", Kind: model.PracticeOwn}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func containsBytes(haystack, needle []byte) bool {
	return bytes.Contains(haystack, needle)
}
