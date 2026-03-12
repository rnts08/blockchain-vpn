package blockchain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReputationStore_New(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "reputation.json")

	store, err := NewReputationStore(dbPath)
	if err != nil {
		t.Fatalf("NewReputationStore failed: %v", err)
	}
	if store == nil {
		t.Fatal("store is nil")
	}
	if store.Records == nil {
		t.Error("Records is nil")
	}
}

func TestReputationStore_Score(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "reputation.json")

	store, err := NewReputationStore(dbPath)
	if err != nil {
		t.Fatalf("NewReputationStore failed: %v", err)
	}

	if score := store.Score("testkey"); score != 0 {
		t.Errorf("expected score 0 for unknown key, got %d", score)
	}

	if err := store.Record("testkey", 10, "test"); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	if score := store.Score("testkey"); score != 10 {
		t.Errorf("expected score 10, got %d", score)
	}
}

func TestReputationStore_Record(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "reputation.json")

	store, err := NewReputationStore(dbPath)
	if err != nil {
		t.Fatalf("NewReputationStore failed: %v", err)
	}

	if err := store.Record("key1", 5, "source1"); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	if score := store.Score("key1"); score != 5 {
		t.Errorf("expected score 5, got %d", score)
	}

	if err := store.Record("key1", 3, "source2"); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	if score := store.Score("key1"); score != 8 {
		t.Errorf("expected score 8 (5+3), got %d", score)
	}
}

func TestReputationStore_NegativeScore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "reputation.json")

	store, err := NewReputationStore(dbPath)
	if err != nil {
		t.Fatalf("NewReputationStore failed: %v", err)
	}

	if err := store.Record("badkey", -10, "source"); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	if score := store.Score("badkey"); score != -10 {
		t.Errorf("expected score -10, got %d", score)
	}
}

func TestReputationStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "reputation.json")

	store1, err := NewReputationStore(dbPath)
	if err != nil {
		t.Fatalf("NewReputationStore failed: %v", err)
	}

	if err := store1.Record("key1", 100, "source"); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	store2, err := NewReputationStore(dbPath)
	if err != nil {
		t.Fatalf("NewReputationStore failed: %v", err)
	}

	if score := store2.Score("key1"); score != 100 {
		t.Errorf("expected score 100 after reload, got %d", score)
	}
}

func TestReputationStore_LoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nonexistent.json")

	store, err := NewReputationStore(dbPath)
	if err != nil {
		t.Fatalf("NewReputationStore failed: %v", err)
	}

	if store.Score("anykey") != 0 {
		t.Error("expected 0 score for non-existent key")
	}
}

func TestReputationStore_ConcurrentRecord(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "reputation.json")

	store, err := NewReputationStore(dbPath)
	if err != nil {
		t.Fatalf("NewReputationStore failed: %v", err)
	}

	done := make(chan struct{})
	go func() {
		for i := 0; i < 50; i++ {
			store.Record("concurrent", 1, "test")
		}
		close(done)
	}()

	for i := 0; i < 50; i++ {
		store.Record("concurrent", 1, "test")
	}
	<-done

	if score := store.Score("concurrent"); score != 100 {
		t.Errorf("expected score 100, got %d", score)
	}
}

func TestDefaultReputationStorePath(t *testing.T) {
	path, err := DefaultReputationStorePath()
	if err != nil {
		t.Fatalf("DefaultReputationStorePath failed: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %s", path)
	}
}

func TestReputationStore_MultipleKeys(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "reputation.json")

	store, err := NewReputationStore(dbPath)
	if err != nil {
		t.Fatalf("NewReputationStore failed: %v", err)
	}

	keys := []string{"key1", "key2", "key3"}
	expected := []int{10, 20, 30}

	for i, key := range keys {
		if err := store.Record(key, expected[i], "test"); err != nil {
			t.Fatalf("Record failed: %v", err)
		}
	}

	for i, key := range keys {
		if score := store.Score(key); score != expected[i] {
			t.Errorf("expected score %d for %s, got %d", expected[i], key, score)
		}
	}
}

func TestReputationStore_EmptySource(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "reputation.json")

	store, err := NewReputationStore(dbPath)
	if err != nil {
		t.Fatalf("NewReputationStore failed: %v", err)
	}

	if err := store.Record("key1", 5, ""); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	if score := store.Score("key1"); score != 5 {
		t.Errorf("expected score 5, got %d", score)
	}
}

func TestReputationStore_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "reputation.json")

	if err := os.WriteFile(dbPath, []byte("invalid json"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, err := NewReputationStore(dbPath)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
