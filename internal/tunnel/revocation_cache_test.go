package tunnel

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
)

func TestRevocationCache_IsRevoked(t *testing.T) {
	key1, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("new key1: %v", err)
	}
	key2, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("new key2: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "revoked.txt")
	content := "# revoked client pubkeys\n" + hex.EncodeToString(key1.PubKey().SerializeCompressed()) + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write revocation file: %v", err)
	}

	cache := &revocationCache{revokedSet: map[string]struct{}{}}
	revoked, err := cache.IsRevoked(path, key1.PubKey())
	if err != nil {
		t.Fatalf("check key1 revoked: %v", err)
	}
	if !revoked {
		t.Fatal("expected key1 to be revoked")
	}

	revoked, err = cache.IsRevoked(path, key2.PubKey())
	if err != nil {
		t.Fatalf("check key2 revoked: %v", err)
	}
	if revoked {
		t.Fatal("expected key2 not to be revoked")
	}

	content = hex.EncodeToString(key2.PubKey().SerializeCompressed()) + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("rewrite revocation file: %v", err)
	}
	cache.mu.Lock()
	cache.lastLoad = time.Time{} // force refresh without waiting throttle interval
	cache.mu.Unlock()

	revoked, err = cache.IsRevoked(path, key2.PubKey())
	if err != nil {
		t.Fatalf("check key2 revoked after update: %v", err)
	}
	if !revoked {
		t.Fatal("expected key2 to be revoked after file update")
	}
}
