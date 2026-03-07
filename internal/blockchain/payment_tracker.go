package blockchain

import (
	"encoding/hex"
	"sync"

	"blockchain-vpn/internal/auth"

	"github.com/btcsuite/btcd/btcec/v2"
)

type paymentTracker struct {
	mu          sync.Mutex
	txToPeer    map[string]string           // txid -> peer key hex
	peerTxCount map[string]int              // peer key hex -> active payment tx count
	peerPubKey  map[string]*btcec.PublicKey // peer key hex -> parsed pubkey for deauth
}

func newPaymentTracker() *paymentTracker {
	return &paymentTracker{
		txToPeer:    make(map[string]string),
		peerTxCount: make(map[string]int),
		peerPubKey:  make(map[string]*btcec.PublicKey),
	}
}

func (t *paymentTracker) trackPayment(txid string, peer *btcec.PublicKey) {
	if txid == "" || peer == nil {
		return
	}
	peerHex := hex.EncodeToString(peer.SerializeCompressed())

	t.mu.Lock()
	defer t.mu.Unlock()
	if _, exists := t.txToPeer[txid]; exists {
		return
	}
	t.txToPeer[txid] = peerHex
	t.peerTxCount[peerHex]++
	t.peerPubKey[peerHex] = peer
}

func (t *paymentTracker) handleRemovedTx(txid string, am *auth.AuthManager) {
	if txid == "" {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	peerHex, exists := t.txToPeer[txid]
	if !exists {
		return
	}
	delete(t.txToPeer, txid)

	if t.peerTxCount[peerHex] > 0 {
		t.peerTxCount[peerHex]--
	}
	if t.peerTxCount[peerHex] > 0 {
		return
	}

	delete(t.peerTxCount, peerHex)
	if peerKey := t.peerPubKey[peerHex]; peerKey != nil {
		am.DeauthorizePeer(peerKey)
	}
	delete(t.peerPubKey, peerHex)
}
