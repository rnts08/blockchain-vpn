package main

import (
	"encoding/hex"
	"log"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
)

// AuthManager manages the list of authorized client public keys.
type AuthManager struct {
	mu              sync.RWMutex
	authorizedPeers map[string]time.Time // Key is hex-encoded compressed public key
}

// NewAuthManager creates a new authorization manager.
func NewAuthManager() *AuthManager {
	return &AuthManager{
		authorizedPeers: make(map[string]time.Time),
	}
}

// AuthorizePeer adds a peer to the authorized list for a given duration.
func (am *AuthManager) AuthorizePeer(peerKey *btcec.PublicKey, duration time.Duration) {
	am.mu.Lock()
	defer am.mu.Unlock()
	keyStr := hex.EncodeToString(peerKey.SerializeCompressed())
	am.authorizedPeers[keyStr] = time.Now().Add(duration)
	log.Printf("Authorized peer %s for %v", peerKey.String(), duration)
}

// IsPeerAuthorized checks if a peer is currently authorized.
func (am *AuthManager) IsPeerAuthorized(peerKey *btcec.PublicKey) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()
	keyStr := hex.EncodeToString(peerKey.SerializeCompressed())
	expiration, ok := am.authorizedPeers[keyStr]
	return ok && time.Now().Before(expiration)
}

// GetAuthorizedPeers returns a map of currently authorized peer public keys (hex-encoded).
func (am *AuthManager) GetAuthorizedPeers() map[string]bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	activePeers := make(map[string]bool)
	now := time.Now()
	for key, expiration := range am.authorizedPeers {
		if now.Before(expiration) {
			activePeers[key] = true
		}
	}
	return activePeers
}