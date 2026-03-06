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

// DeauthorizePeer removes a peer from the authorized list.
func (am *AuthManager) DeauthorizePeer(peerKey *btcec.PublicKey) {
	am.mu.Lock()
	defer am.mu.Unlock()
	keyStr := hex.EncodeToString(peerKey.SerializeCompressed())
	delete(am.authorizedPeers, keyStr)
	log.Printf("De-authorized peer %s", peerKey.String())
}

// DeauthorizeAllPeers clears the entire authorization list. Used for reorgs.
func (am *AuthManager) DeauthorizeAllPeers() {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.authorizedPeers = make(map[string]time.Time)
	log.Println("All peers have been de-authorized due to a potential reorg.")
}

// GetPeerExpiration returns the expiration time for a given peer.
func (am *AuthManager) GetPeerExpiration(peerKey *btcec.PublicKey) (time.Time, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	keyStr := hex.EncodeToString(peerKey.SerializeCompressed())
	expiration, ok := am.authorizedPeers[keyStr]
	return expiration, ok
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