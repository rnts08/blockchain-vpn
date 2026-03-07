package blockchain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"blockchain-vpn/internal/protocol"
)

// ScanCache stores the state of the blockchain scanner so it can resume
// from the last scanned block instead of rescanning the whole chain on every startup.
type ScanCache struct {
	LastScannedHeight int64                            `json:"last_scanned_height"`
	UpdatedTime       time.Time                        `json:"updated_time"`
	Announcements     map[string]*protocol.VPNEndpoint `json:"announcements"`
	PriceUpdates      map[string]uint64                `json:"price_updates"`
	Heartbeats        map[string]uint8                 `json:"heartbeats"`

	mu       sync.RWMutex
	filePath string
}

// NewScanCache creates a new cache instance saving to the given path.
func NewScanCache(path string) *ScanCache {
	return &ScanCache{
		filePath:      path,
		Announcements: make(map[string]*protocol.VPNEndpoint),
		PriceUpdates:  make(map[string]uint64),
		Heartbeats:    make(map[string]uint8),
	}
}

// Load reads the cache from disk. If the file doesn't exist, it returns success
// but leaves the cache empty.
func (c *ScanCache) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Normal on first run
		}
		return fmt.Errorf("failed to read scan cache: %w", err)
	}

	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to parse scan cache: %w", err)
	}

	if c.Announcements == nil {
		c.Announcements = make(map[string]*protocol.VPNEndpoint)
	}
	if c.PriceUpdates == nil {
		c.PriceUpdates = make(map[string]uint64)
	}
	if c.Heartbeats == nil {
		c.Heartbeats = make(map[string]uint8)
	}
	return nil
}

// Save writes the current cache state to disk atomically.
func (c *ScanCache) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal scan cache: %w", err)
	}

	dir := filepath.Dir(c.filePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	tmpFile := c.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
		return fmt.Errorf("failed to write scan cache tmp file: %w", err)
	}

	if err := os.Rename(tmpFile, c.filePath); err != nil {
		return fmt.Errorf("failed to commit scan cache file: %w", err)
	}
	return nil
}

// Update saves new data into the cache if it's newer, and sets the updated height.
func (c *ScanCache) Update(height int64, anns map[string]*ProviderAnnouncement, prices map[string]uint64, hbs map[string]heartbeatState) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if height > c.LastScannedHeight {
		c.LastScannedHeight = height
	}
	c.UpdatedTime = time.Now().UTC()

	for pk, a := range anns {
		// New announcements override old ones
		c.Announcements[pk] = a.Endpoint
	}
	for pk, p := range prices {
		c.PriceUpdates[pk] = p
	}
	for pk, h := range hbs {
		c.Heartbeats[pk] = h.flags
	}
}

// DefaultScanCachePath returns the standard path for the cache file in the OS config dir.
func DefaultScanCachePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(configDir, "BlockchainVPN")
	return filepath.Join(appDir, "scan_cache.json"), nil
}
