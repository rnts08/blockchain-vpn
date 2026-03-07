package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	if err := WriteFileAtomic(path, []byte("one"), 0o600); err != nil {
		t.Fatalf("WriteFileAtomic first write failed: %v", err)
	}
	if err := WriteFileAtomic(path, []byte("two"), 0o600); err != nil {
		t.Fatalf("WriteFileAtomic second write failed: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(b) != "two" {
		t.Fatalf("expected final content to be 'two', got %q", string(b))
	}
}
