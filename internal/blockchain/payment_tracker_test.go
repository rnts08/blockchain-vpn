package blockchain

import (
	"testing"

	"blockchain-vpn/internal/auth"

	"github.com/btcsuite/btcd/btcec/v2"
)

func TestTrackPayment_InvalidInputs(t *testing.T) {
	t.Parallel()
	tracker := newPaymentTracker()

	// Test empty txid
	peerKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	tracker.trackPayment("", peerKey.PubKey())
	if len(tracker.txToPeer) != 0 {
		t.Error("expected no tracking for empty txid")
	}

	// Test nil peer
	txid := "testtx123"
	tracker.trackPayment(txid, nil)
	if len(tracker.txToPeer) != 0 {
		t.Error("expected no tracking for nil peer")
	}
}

func TestTrackPayment_ValidTracking(t *testing.T) {
	t.Parallel()
	tracker := newPaymentTracker()

	peerKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	txid := "testtx123"
	tracker.trackPayment(txid, peerKey.PubKey())

	// Verify tx is tracked
	if len(tracker.txToPeer) != 1 {
		t.Errorf("expected 1 tracked tx, got %d", len(tracker.txToPeer))
	}

	if tracker.txToPeer[txid] == "" {
		t.Error("expected peer hex in txToPeer map")
	}

	// Verify peer count
	peerHex := tracker.txToPeer[txid]
	if tracker.peerTxCount[peerHex] != 1 {
		t.Errorf("expected peer count 1, got %d", tracker.peerTxCount[peerHex])
	}

	// Verify peer pubkey
	if tracker.peerPubKey[peerHex] == nil {
		t.Error("expected peer pubkey in peerPubKey map")
	}
}

func TestTrackPayment_DuplicateTx(t *testing.T) {
	t.Parallel()
	tracker := newPaymentTracker()

	peerKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	txid := "testtx123"

	// Track same tx twice
	tracker.trackPayment(txid, peerKey.PubKey())
	tracker.trackPayment(txid, peerKey.PubKey())

	// Verify only one entry
	if len(tracker.txToPeer) != 1 {
		t.Errorf("expected 1 tracked tx, got %d", len(tracker.txToPeer))
	}

	peerHex := tracker.txToPeer[txid]
	if tracker.peerTxCount[peerHex] != 1 {
		t.Errorf("expected peer count 1 (duplicate should not increment), got %d", tracker.peerTxCount[peerHex])
	}
}

func TestTrackPayment_MultiplePeers(t *testing.T) {
	t.Parallel()
	tracker := newPaymentTracker()

	peer1Key, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	peer2Key, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	tx1 := "tx1"
	tx2 := "tx2"

	tracker.trackPayment(tx1, peer1Key.PubKey())
	tracker.trackPayment(tx2, peer2Key.PubKey())

	// Verify both transactions tracked
	if len(tracker.txToPeer) != 2 {
		t.Errorf("expected 2 tracked tx, got %d", len(tracker.txToPeer))
	}

	// Verify peer counts
	peer1Hex := tracker.txToPeer[tx1]
	peer2Hex := tracker.txToPeer[tx2]

	if tracker.peerTxCount[peer1Hex] != 1 {
		t.Errorf("expected peer1 count 1, got %d", tracker.peerTxCount[peer1Hex])
	}

	if tracker.peerTxCount[peer2Hex] != 1 {
		t.Errorf("expected peer2 count 1, got %d", tracker.peerTxCount[peer2Hex])
	}
}

func TestTrackPayment_MultipleTxFromSamePeer(t *testing.T) {
	t.Parallel()
	tracker := newPaymentTracker()

	peerKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	tx1 := "tx1"
	tx2 := "tx2"
	tx3 := "tx3"

	tracker.trackPayment(tx1, peerKey.PubKey())
	tracker.trackPayment(tx2, peerKey.PubKey())
	tracker.trackPayment(tx3, peerKey.PubKey())

	// Verify all transactions tracked
	if len(tracker.txToPeer) != 3 {
		t.Errorf("expected 3 tracked tx, got %d", len(tracker.txToPeer))
	}

	peerHex := tracker.txToPeer[tx1]

	// Verify peer count reflects all transactions
	if tracker.peerTxCount[peerHex] != 3 {
		t.Errorf("expected peer count 3, got %d", tracker.peerTxCount[peerHex])
	}

	// Verify peer pubkey stored
	if tracker.peerPubKey[peerHex] == nil {
		t.Error("expected peer pubkey stored")
	}
}

func TestHandleRemovedTx_InvalidInputs(t *testing.T) {
	t.Parallel()
	tracker := newPaymentTracker()
	authMgr := auth.NewAuthManager()

	// Test empty txid
	tracker.handleRemovedTx("", authMgr)
	if len(tracker.txToPeer) != 0 {
		t.Error("expected no changes for empty txid")
	}
}

func TestHandleRemovedTx_NonExistentTx(t *testing.T) {
	t.Parallel()
	tracker := newPaymentTracker()
	authMgr := auth.NewAuthManager()

	txid := "nonexistenttx"
	tracker.handleRemovedTx(txid, authMgr)

	if len(tracker.txToPeer) != 0 {
		t.Error("expected no changes for nonexistent tx")
	}
}

func TestHandleRemovedTx_SingleTx(t *testing.T) {
	t.Parallel()
	tracker := newPaymentTracker()
	authMgr := auth.NewAuthManager()

	peerKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	txid := "testtx123"

	// Track transaction first
	tracker.trackPayment(txid, peerKey.PubKey())

	// Remove transaction
	tracker.handleRemovedTx(txid, authMgr)

	// Verify tx removed
	if len(tracker.txToPeer) != 0 {
		t.Errorf("expected tx removed, still have %d txs", len(tracker.txToPeer))
	}

	peerHex := ""
	for hex := range tracker.peerPubKey {
		peerHex = hex
		break
	}

	// Verify peer data cleaned up (since this was the only tx)
	if _, exists := tracker.peerTxCount[peerHex]; exists {
		t.Error("expected peer count cleaned up")
	}

	if _, exists := tracker.peerPubKey[peerHex]; exists {
		t.Error("expected peer pubkey cleaned up")
	}
}

func TestHandleRemovedTx_MultipleTxFromSamePeer(t *testing.T) {
	t.Parallel()
	tracker := newPaymentTracker()
	authMgr := auth.NewAuthManager()

	peerKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	tx1 := "tx1"
	tx2 := "tx2"
	tx3 := "tx3"

	// Track multiple transactions from same peer
	tracker.trackPayment(tx1, peerKey.PubKey())
	tracker.trackPayment(tx2, peerKey.PubKey())
	tracker.trackPayment(tx3, peerKey.PubKey())

	// Remove one transaction
	tracker.handleRemovedTx(tx1, authMgr)

	// Verify tx removed
	if len(tracker.txToPeer) != 2 {
		t.Errorf("expected 2 remaining txs, got %d", len(tracker.txToPeer))
	}

	peerHex := tracker.txToPeer[tx2]

	// Verify peer count decremented but not cleaned up
	if tracker.peerTxCount[peerHex] != 2 {
		t.Errorf("expected peer count 2, got %d", tracker.peerTxCount[peerHex])
	}

	if _, exists := tracker.peerPubKey[peerHex]; !exists {
		t.Error("expected peer pubkey still exists")
	}
}

func TestHandleRemovedTx_RemoveAllTxFromPeer(t *testing.T) {
	t.Parallel()
	tracker := newPaymentTracker()
	authMgr := auth.NewAuthManager()

	peerKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	tx1 := "tx1"
	tx2 := "tx2"

	// Track multiple transactions from same peer
	tracker.trackPayment(tx1, peerKey.PubKey())
	tracker.trackPayment(tx2, peerKey.PubKey())

	// Remove both transactions
	tracker.handleRemovedTx(tx1, authMgr)
	tracker.handleRemovedTx(tx2, authMgr)

	// Verify all txs removed
	if len(tracker.txToPeer) != 0 {
		t.Errorf("expected all txs removed, still have %d", len(tracker.txToPeer))
	}

	// Verify peer data cleaned up (peer should be deauthorized)
	peerHex := ""
	for hex := range tracker.peerPubKey {
		peerHex = hex
		break
	}

	if _, exists := tracker.peerTxCount[peerHex]; exists {
		t.Error("expected peer count cleaned up")
	}

	if _, exists := tracker.peerPubKey[peerHex]; exists {
		t.Error("expected peer pubkey cleaned up")
	}
}

func TestHandleRemovedTx_MultiplePeers(t *testing.T) {
	t.Parallel()
	tracker := newPaymentTracker()
	authMgr := auth.NewAuthManager()

	peer1Key, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	peer2Key, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	tx1 := "tx1"
	tx2 := "tx2"
	tx3 := "tx3"

	// Track transactions from different peers
	tracker.trackPayment(tx1, peer1Key.PubKey())
	tracker.trackPayment(tx2, peer2Key.PubKey())
	tracker.trackPayment(tx3, peer1Key.PubKey())

	// Remove one transaction from peer1
	tracker.handleRemovedTx(tx1, authMgr)

	// Verify peer1 still has one transaction remaining
	if len(tracker.txToPeer) != 2 {
		t.Errorf("expected 2 remaining txs, got %d", len(tracker.txToPeer))
	}

	peer1Hex := tracker.txToPeer[tx3]
	if tracker.peerTxCount[peer1Hex] != 1 {
		t.Errorf("expected peer1 count 1, got %d", tracker.peerTxCount[peer1Hex])
	}

	peer2Hex := tracker.txToPeer[tx2]
	if tracker.peerTxCount[peer2Hex] != 1 {
		t.Errorf("expected peer2 count 1, got %d", tracker.peerTxCount[peer2Hex])
	}
}

func TestHandleRemovedTx_DeauthBehavior(t *testing.T) {
	t.Parallel()
	tracker := newPaymentTracker()
	authMgr := auth.NewAuthManager()

	peerKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	txid := "testtx123"

	// Track transaction
	tracker.trackPayment(txid, peerKey.PubKey())

	// Remove transaction (should trigger deauth)
	tracker.handleRemovedTx(txid, authMgr)

	// Verify peer was deauthorized (should not be in authMgr)
	authorizedPeers := authMgr.GetAuthorizedPeers()
	peerHex := peerKey.PubKey().SerializeCompressed()

	if authorizedPeers[string(peerHex)] {
		t.Error("expected peer to be deauthorized")
	}
}
