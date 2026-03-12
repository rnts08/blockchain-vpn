package auth

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
)

func TestAuthManager_NewAuthManager(t *testing.T) {
	am := NewAuthManager()
	if am == nil {
		t.Fatal("NewAuthManager returned nil")
	}
	if am.authorizedPeers == nil {
		t.Error("authorizedPeers map is nil")
	}
}

func TestAuthManager_AuthorizePeer(t *testing.T) {
	am := NewAuthManager()
	key, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey: %v", err)
	}
	pub := key.PubKey()

	// Authorize with unlimited data (dataQuota=0)
	am.AuthorizePeer(pub, 10*time.Minute, 0)
	keyStr := hexEncode(pub)

	info, ok := am.authorizedPeers[keyStr]
	if !ok {
		t.Fatal("peer not authorized after AuthorizePeer")
	}
	if info.dataQuota != 0 {
		t.Errorf("dataQuota = %d, want 0", info.dataQuota)
	}
	if info.remaining != 0 {
		t.Errorf("remaining = %d, want 0 for unlimited", info.remaining)
	}
	// Verify expiration is ~10 minutes from now
	if elapsed := time.Until(info.expiresAt); elapsed < 9*time.Minute || elapsed > 11*time.Minute {
		t.Errorf("expiresAt = %v, expected ~10 minutes from now", info.expiresAt)
	}

	// Authorize with data quota
	key2, _ := btcec.NewPrivateKey()
	pub2 := key2.PubKey()
	am.AuthorizePeer(pub2, 5*time.Minute, 1000000)
	keyStr2 := hexEncode(pub2)
	info2, ok2 := am.authorizedPeers[keyStr2]
	if !ok2 {
		t.Fatal("peer2 not authorized")
	}
	if info2.dataQuota != 1000000 {
		t.Errorf("dataQuota = %d, want 1000000", info2.dataQuota)
	}
	if info2.remaining != 1000000 {
		t.Errorf("remaining = %d, want 1000000", info2.remaining)
	}

	// Test extending authorization (same key with longer duration)
	am.AuthorizePeer(pub, 20*time.Minute, 0) // extend
	info3, ok3 := am.authorizedPeers[keyStr]
	if !ok3 {
		t.Fatal("peer not found after extension")
	}
	// Should be later than original expiration
	if !info3.expiresAt.After(info.expiresAt) {
		t.Error("expiration not extended")
	}

	// Test extending with additional data quota
	am.AuthorizePeer(pub2, 10*time.Minute, 500000)
	info4, ok4 := am.authorizedPeers[keyStr2]
	if !ok4 {
		t.Fatal("peer2 not found after quota extension")
	}
	// When existing has quota, we add to it
	if info4.dataQuota != 1500000 {
		t.Errorf("dataQuota after extension = %d, want 1500000", info4.dataQuota)
	}
	if info4.remaining != 1500000 {
		t.Errorf("remaining after extension = %d, want 1500000", info4.remaining)
	}
}

func TestAuthManager_IsPeerAuthorized(t *testing.T) {
	am := NewAuthManager()
	key, _ := btcec.NewPrivateKey()
	pub := key.PubKey()

	// Not authorized initially
	if am.IsPeerAuthorized(pub) {
		t.Error("IsPeerAuthorized = true for unauthorized peer")
	}

	// Authorize with long duration, unlimited data
	am.AuthorizePeer(pub, 10*time.Minute, 0)
	if !am.IsPeerAuthorized(pub) {
		t.Error("IsPeerAuthorized = false for authorized peer")
	}

	// Test expiration (use very short duration and sleep)
	key2, _ := btcec.NewPrivateKey()
	pub2 := key2.PubKey()
	am.AuthorizePeer(pub2, 150*time.Millisecond, 0)
	if !am.IsPeerAuthorized(pub2) {
		t.Error("IsPeerAuthorized = false for newly authorized peer with short expiry")
	}
	time.Sleep(200 * time.Millisecond)
	if am.IsPeerAuthorized(pub2) {
		t.Error("IsPeerAuthorized = true after expiration")
	}

	// Test data quota exhaustion
	key3, _ := btcec.NewPrivateKey()
	pub3 := key3.PubKey()
	am.AuthorizePeer(pub3, 10*time.Minute, 100)
	// Consume all data
	rem := am.ConsumeData(pub3, 100)
	if rem != 0 {
		t.Errorf("ConsumeData returned %d, want 0", rem)
	}
	if am.IsPeerAuthorized(pub3) {
		t.Error("IsPeerAuthorized = true when data quota exhausted")
	}
}

func TestAuthManager_DeauthorizePeer(t *testing.T) {
	am := NewAuthManager()
	key, _ := btcec.NewPrivateKey()
	pub := key.PubKey()

	am.AuthorizePeer(pub, 10*time.Minute, 0)
	if !am.IsPeerAuthorized(pub) {
		t.Fatal("peer not authorized before deauthorize")
	}

	am.DeauthorizePeer(pub)
	if am.IsPeerAuthorized(pub) {
		t.Error("IsPeerAuthorized = true after DeauthorizePeer")
	}
}

func TestAuthManager_DeauthorizeAllPeers(t *testing.T) {
	am := NewAuthManager()
	keys := make([]*btcec.PublicKey, 3)
	for i := 0; i < 3; i++ {
		priv, _ := btcec.NewPrivateKey()
		keys[i] = priv.PubKey()
		am.AuthorizePeer(keys[i], 10*time.Minute, 0)
	}

	if len(am.authorizedPeers) != 3 {
		t.Fatalf("expected 3 peers, got %d", len(am.authorizedPeers))
	}

	am.DeauthorizeAllPeers()

	if len(am.authorizedPeers) != 0 {
		t.Fatalf("expected 0 peers after DeauthorizeAllPeers, got %d", len(am.authorizedPeers))
	}
}

func TestAuthManager_GetPeerExpiration(t *testing.T) {
	am := NewAuthManager()
	key, _ := btcec.NewPrivateKey()
	pub := key.PubKey()

	exp, ok := am.GetPeerExpiration(pub)
	if ok {
		t.Error("GetPeerExpiration returned true for unauthorized peer")
	}
	if !exp.IsZero() {
		t.Errorf("exp = %v, expected zero time", exp)
	}

	am.AuthorizePeer(pub, 5*time.Minute, 0)
	exp2, ok2 := am.GetPeerExpiration(pub)
	if !ok2 {
		t.Fatal("GetPeerExpiration returned false for authorized peer")
	}
	// Check expiration is ~5 minutes from now
	if elapsed := time.Until(exp2); elapsed < 4*time.Minute || elapsed > 6*time.Minute {
		t.Errorf("exp = %v, expected ~5 minutes from now", exp2)
	}

	// After deauthorization
	am.DeauthorizePeer(pub)
	exp3, ok3 := am.GetPeerExpiration(pub)
	if ok3 {
		t.Error("GetPeerExpiration returned true after deauthorize")
	}
	if !exp3.IsZero() {
		t.Errorf("exp = %v after deauth, expected zero", exp3)
	}
}

func TestAuthManager_GetAuthorizedPeers(t *testing.T) {
	am := NewAuthManager()

	// Empty map
	peers := am.GetAuthorizedPeers()
	if len(peers) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(peers))
	}

	// Add peers
	key1, _ := btcec.NewPrivateKey()
	pub1 := key1.PubKey()
	key2, _ := btcec.NewPrivateKey()
	pub2 := key2.PubKey()

	am.AuthorizePeer(pub1, 10*time.Minute, 0)
	am.AuthorizePeer(pub2, 10*time.Minute, 0)

	peers = am.GetAuthorizedPeers()
	if len(peers) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(peers))
	}
	if !peers[hexEncode(pub1)] {
		t.Error("peer1 not in authorized peers map")
	}
	if !peers[hexEncode(pub2)] {
		t.Error("peer2 not in authorized peers map")
	}

	// Deauthorize one peer - tests that GetAuthorizedPeers only returns active ones
	am.DeauthorizePeer(pub1)
	peers = am.GetAuthorizedPeers()
	if len(peers) != 1 {
		t.Fatalf("expected 1 peer after deauth, got %d", len(peers))
	}
	if peers[hexEncode(pub1)] {
		t.Error("deauthorized peer still appears in GetAuthorizedPeers")
	}
}

func TestAuthManager_ConsumeData(t *testing.T) {
	am := NewAuthManager()
	key, _ := btcec.NewPrivateKey()
	pub := key.PubKey()

	// Peer with unlimited data
	am.AuthorizePeer(pub, 10*time.Minute, 0)
	rem := am.ConsumeData(pub, 1000)
	if rem != ^uint64(0) {
		t.Errorf("ConsumeData returned %d for unlimited, want max uint64", rem)
	}

	// Peer with quota
	am2 := NewAuthManager()
	key2, _ := btcec.NewPrivateKey()
	pub2 := key2.PubKey()
	am2.AuthorizePeer(pub2, 10*time.Minute, 1000)

	rem2 := am2.ConsumeData(pub2, 300)
	if rem2 != 700 {
		t.Errorf("ConsumeData returned %d, want 700", rem2)
	}
	// Verify remaining in map
	info, _ := am2.authorizedPeers[hexEncode(pub2)]
	if info.remaining != 700 {
		t.Errorf("remaining in map = %d, want 700", info.remaining)
	}

	// Consume more
	rem3 := am2.ConsumeData(pub2, 700)
	if rem3 != 0 {
		t.Errorf("ConsumeData returned %d after full consumption, want 0", rem3)
	}

	// Consume when already at 0
	rem4 := am2.ConsumeData(pub2, 100)
	if rem4 != 0 {
		t.Errorf("ConsumeData returned %d when at 0, want 0", rem4)
	}

	// Consume for non-existent peer
	key3, _ := btcec.NewPrivateKey()
	pub3 := key3.PubKey()
	rem5 := am2.ConsumeData(pub3, 100)
	if rem5 != 0 {
		t.Errorf("ConsumeData returned %d for non-existent peer, want 0", rem5)
	}
}

func TestAuthManager_GetPeerDataRemaining(t *testing.T) {
	am := NewAuthManager()
	key, _ := btcec.NewPrivateKey()
	pub := key.PubKey()

	// Unlimited
	am.AuthorizePeer(pub, 10*time.Minute, 0)
	rem := am.GetPeerDataRemaining(pub)
	if rem != ^uint64(0) {
		t.Errorf("GetPeerDataRemaining = %d for unlimited, want max uint64", rem)
	}

	// Limited
	key2, _ := btcec.NewPrivateKey()
	pub2 := key2.PubKey()
	am.AuthorizePeer(pub2, 10*time.Minute, 5000)
	rem2 := am.GetPeerDataRemaining(pub2)
	if rem2 != 5000 {
		t.Errorf("GetPeerDataRemaining = %d, want 5000", rem2)
	}

	// After consumption (use ConsumeData to modify)
	am2 := NewAuthManager()
	key3, _ := btcec.NewPrivateKey()
	pub3 := key3.PubKey()
	am2.AuthorizePeer(pub3, 10*time.Minute, 5000)
	am2.ConsumeData(pub3, 2000)
	rem3 := am2.GetPeerDataRemaining(pub3)
	if rem3 != 3000 {
		t.Errorf("GetPeerDataRemaining = %d after consumption, want 3000", rem3)
	}

	// Non-existent peer
	key4, _ := btcec.NewPrivateKey()
	pub4 := key4.PubKey()
	rem4 := am2.GetPeerDataRemaining(pub4)
	if rem4 != 0 {
		t.Errorf("GetPeerDataRemaining = %d for non-existent, want 0", rem4)
	}
}

// hexEncode encodes a public key to its hex string representation
func hexEncode(k *btcec.PublicKey) string {
	return hex.EncodeToString(k.SerializeCompressed())
}
