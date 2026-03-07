package tunnel

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
)

func TestAccessPolicyAllowlistDenylist(t *testing.T) {
	allowKey, _ := btcec.NewPrivateKey()
	denyKey, _ := btcec.NewPrivateKey()
	otherKey, _ := btcec.NewPrivateKey()

	dir := t.TempDir()
	allowFile := filepath.Join(dir, "allow.txt")
	denyFile := filepath.Join(dir, "deny.txt")

	if err := os.WriteFile(allowFile, []byte(hexKey(allowKey.PubKey())+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(denyFile, []byte(hexKey(denyKey.PubKey())+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	p, err := loadAccessPolicy(allowFile, denyFile)
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}

	if err := p.check(allowKey.PubKey()); err != nil {
		t.Fatalf("allow key should pass: %v", err)
	}
	if err := p.check(denyKey.PubKey()); err == nil {
		t.Fatal("deny key should fail")
	}
	if err := p.check(otherKey.PubKey()); err == nil {
		t.Fatal("key not in allowlist should fail when allowlist is configured")
	}
}

func hexKey(k *btcec.PublicKey) string {
	return strings.ToLower(hex.EncodeToString(k.SerializeCompressed()))
}
