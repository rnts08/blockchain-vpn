package main

import (
	"log"
	"sync"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// AuthManager manages the list of authorized client public keys.
type AuthManager struct {
	mu              sync.RWMutex
	authorizedPeers map[wgtypes.Key]time.Time
}

// NewAuthManager creates a new authorization manager.
func NewAuthManager() *AuthManager {
	return &AuthManager{
		authorizedPeers: make(map[wgtypes.Key]time.Time),
	}
}

// AuthorizePeer adds a peer to the authorized list for a given duration.
func (am *AuthManager) AuthorizePeer(peerKey wgtypes.Key, duration time.Duration) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.authorizedPeers[peerKey] = time.Now().Add(duration)
	log.Printf("Authorized peer %s for %v", peerKey.String(), duration)
}

// IsPeerAuthorized checks if a peer is currently authorized.
func (am *AuthManager) IsPeerAuthorized(peerKey wgtypes.Key) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()
	expiration, ok := am.authorizedPeers[peerKey]
	return ok && time.Now().Before(expiration)
}

// GetAuthorizedPeers returns a map of currently authorized peer keys.
func (am *AuthManager) GetAuthorizedPeers() map[wgtypes.Key]bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	activePeers := make(map[wgtypes.Key]bool)
	now := time.Now()
	for key, expiration := range am.authorizedPeers {
		if now.Before(expiration) {
			activePeers[key] = true
		}
	}
	return activePeers
}