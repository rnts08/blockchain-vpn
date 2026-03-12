package auth

import (
	"encoding/hex"
	"log"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
)

type peerAuthInfo struct {
	expiresAt time.Time
	dataQuota uint64 // 0 = unlimited data
	remaining uint64 // for quota-limited peers; if dataQuota>0, starts at dataQuota and counts down
}

// AuthManager manages the list of authorized client public keys with optional data quota.
type AuthManager struct {
	mu              sync.RWMutex
	authorizedPeers map[string]peerAuthInfo // Key is hex-encoded compressed public key
}

// NewAuthManager creates a new authorization manager.
func NewAuthManager() *AuthManager {
	return &AuthManager{
		authorizedPeers: make(map[string]peerAuthInfo),
	}
}

// AuthorizePeer adds a peer to the authorized list for a given duration and optional data quota.
// Pass dataQuota > 0 to enforce a data limit in bytes. Pass 0 for unlimited data.
func (am *AuthManager) AuthorizePeer(peerKey *btcec.PublicKey, duration time.Duration, dataQuota uint64) {
	am.mu.Lock()
	defer am.mu.Unlock()
	keyStr := hex.EncodeToString(peerKey.SerializeCompressed())
	now := time.Now()
	newExp := now.Add(duration)

	if existing, exists := am.authorizedPeers[keyStr]; exists {
		// Extend expiration if the new one is later
		if newExp.After(existing.expiresAt) {
			existing.expiresAt = newExp
		}
		if dataQuota > 0 {
			if existing.dataQuota == 0 {
				existing.dataQuota = dataQuota
				existing.remaining = dataQuota
			} else {
				existing.dataQuota += dataQuota
				existing.remaining += dataQuota
			}
		}
		am.authorizedPeers[keyStr] = existing
		log.Printf("Extended authorization for peer %s: exp=%s, dataQuota=%d (remaining=%d)", keyStr, existing.expiresAt.Format(time.RFC3339), existing.dataQuota, existing.remaining)
	} else {
		info := peerAuthInfo{
			expiresAt: newExp,
			dataQuota: dataQuota,
		}
		if dataQuota > 0 {
			info.remaining = dataQuota
		}
		am.authorizedPeers[keyStr] = info
		log.Printf("Authorized peer %s: exp=%s, dataQuota=%d", keyStr, newExp.Format(time.RFC3339), dataQuota)
	}
}

// IsPeerAuthorized checks if a peer is currently authorized (not expired and data quota not exhausted).
func (am *AuthManager) IsPeerAuthorized(peerKey *btcec.PublicKey) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()
	keyStr := hex.EncodeToString(peerKey.SerializeCompressed())
	info, ok := am.authorizedPeers[keyStr]
	if !ok {
		return false
	}
	if !time.Now().Before(info.expiresAt) {
		return false
	}
	if info.dataQuota > 0 && info.remaining == 0 {
		return false
	}
	return true
}

// DeauthorizePeer removes a peer from the authorized list.
func (am *AuthManager) DeauthorizePeer(peerKey *btcec.PublicKey) {
	am.mu.Lock()
	defer am.mu.Unlock()
	keyStr := hex.EncodeToString(peerKey.SerializeCompressed())
	delete(am.authorizedPeers, keyStr)
	log.Printf("De-authorized peer %s", keyStr)
}

// DeauthorizeAllPeers clears the entire authorization list. Used for reorgs.
func (am *AuthManager) DeauthorizeAllPeers() {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.authorizedPeers = make(map[string]peerAuthInfo)
	log.Println("All peers have been de-authorized due to a potential reorg.")
}

// GetPeerExpiration returns the expiration time for a given peer.
func (am *AuthManager) GetPeerExpiration(peerKey *btcec.PublicKey) (time.Time, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	keyStr := hex.EncodeToString(peerKey.SerializeCompressed())
	info, ok := am.authorizedPeers[keyStr]
	if !ok {
		return time.Time{}, false
	}
	return info.expiresAt, true
}

// GetAuthorizedPeers returns a map of currently authorized peer public keys (hex-encoded).
func (am *AuthManager) GetAuthorizedPeers() map[string]bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	activePeers := make(map[string]bool)
	now := time.Now()
	for key, info := range am.authorizedPeers {
		if now.Before(info.expiresAt) && !(info.dataQuota > 0 && info.remaining == 0) {
			activePeers[key] = true
		}
	}
	return activePeers
}

// ConsumeData subtracts the given number of bytes from the peer's remaining data quota.
// Returns the new remaining quota, or ^uint64(0) if unlimited, or 0 if insufficient.
func (am *AuthManager) ConsumeData(peerKey *btcec.PublicKey, bytes uint64) uint64 {
	am.mu.Lock()
	defer am.mu.Unlock()
	keyStr := hex.EncodeToString(peerKey.SerializeCompressed())
	info, ok := am.authorizedPeers[keyStr]
	if !ok {
		return 0
	}
	if info.dataQuota == 0 {
		// Unlimited data
		return ^uint64(0)
	}
	if info.remaining < bytes {
		info.remaining = 0
		am.authorizedPeers[keyStr] = info
		return 0
	}
	info.remaining -= bytes
	am.authorizedPeers[keyStr] = info
	return info.remaining
}

// GetPeerDataRemaining returns the remaining data quota for a peer, or ^uint64(0) if unlimited.
func (am *AuthManager) GetPeerDataRemaining(peerKey *btcec.PublicKey) uint64 {
	am.mu.RLock()
	defer am.mu.RUnlock()
	keyStr := hex.EncodeToString(peerKey.SerializeCompressed())
	info, ok := am.authorizedPeers[keyStr]
	if !ok {
		return 0
	}
	if info.dataQuota == 0 {
		return ^uint64(0) // unlimited
	}
	return info.remaining
}
