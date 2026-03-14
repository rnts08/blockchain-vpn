//go:build functional

package tunnel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
)

func TestFunctional_AccessControl_AllowlistOnly(t *testing.T) {
	t.Parallel()

	allowKey1, _ := btcec.NewPrivateKey()
	allowKey2, _ := btcec.NewPrivateKey()
	otherKey, _ := btcec.NewPrivateKey()

	dir := t.TempDir()
	allowFile := filepath.Join(dir, "allow.txt")

	if err := os.WriteFile(allowFile, []byte(hexKey(allowKey1.PubKey())+"\n"+hexKey(allowKey2.PubKey())+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	policy, err := loadAccessPolicy(allowFile, "")
	if err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	if err := policy.check(allowKey1.PubKey()); err != nil {
		t.Errorf("Allow key 1 should pass: %v", err)
	}

	if err := policy.check(allowKey2.PubKey()); err != nil {
		t.Errorf("Allow key 2 should pass: %v", err)
	}

	if err := policy.check(otherKey.PubKey()); err == nil {
		t.Error("Non-allowlisted key should fail")
	}

	t.Log("Access control allowlist works correctly")
}

func TestFunctional_AccessControl_DenylistOnly(t *testing.T) {
	t.Parallel()

	denyKey, _ := btcec.NewPrivateKey()
	otherKey, _ := btcec.NewPrivateKey()

	dir := t.TempDir()
	denyFile := filepath.Join(dir, "deny.txt")

	if err := os.WriteFile(denyFile, []byte(hexKey(denyKey.PubKey())+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	policy, err := loadAccessPolicy("", denyFile)
	if err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	if err := policy.check(denyKey.PubKey()); err == nil {
		t.Error("Denylisted key should fail")
	}

	if err := policy.check(otherKey.PubKey()); err != nil {
		t.Error("Non-denylisted key should pass")
	}

	t.Log("Access control denylist works correctly")
}

func TestFunctional_AccessControl_Empty(t *testing.T) {
	t.Parallel()

	key, _ := btcec.NewPrivateKey()

	policy, err := loadAccessPolicy("", "")
	if err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	if err := policy.check(key.PubKey()); err != nil {
		t.Error("Any key should pass when both lists are empty")
	}

	t.Log("Access control empty policy works correctly")
}

func TestFunctional_AccessControl_BothLists(t *testing.T) {
	t.Parallel()

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

	policy, err := loadAccessPolicy(allowFile, denyFile)
	if err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	if err := policy.check(allowKey.PubKey()); err != nil {
		t.Errorf("Allowlisted key should pass: %v", err)
	}

	if err := policy.check(denyKey.PubKey()); err == nil {
		t.Error("Denylisted key should fail even with allowlist present")
	}

	if err := policy.check(otherKey.PubKey()); err == nil {
		t.Error("Key not in either list should fail when allowlist is present")
	}

	t.Log("Access control combined allowlist/denylist works correctly")
}
