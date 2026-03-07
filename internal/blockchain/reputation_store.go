package blockchain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ReputationRecord holds the local scored reputation of a single provider.
type ReputationRecord struct {
	Score        int       `json:"score"` // Positive/negative reputation points
	LastUpdated  time.Time `json:"last_updated"`
	ReviewSource string    `json:"review_source"` // e.g., "self", "community"
}

// ReputationStore manages the local reputation database (JSON).
type ReputationStore struct {
	Records map[string]*ReputationRecord `json:"records"` // Key: hex compressed pubkey
	mu      sync.RWMutex
	path    string
}

// NewReputationStore unmarshals the reputation store from disk, or creates a new one.
func NewReputationStore(dbPath string) (*ReputationStore, error) {
	store := &ReputationStore{
		Records: make(map[string]*ReputationRecord),
		path:    dbPath,
	}
	if err := store.Load(); err != nil {
		return nil, fmt.Errorf("failed to load reputation store: %w", err)
	}
	return store, nil
}

// Load reads the reputation store from disk.
func (s *ReputationStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.Records = make(map[string]*ReputationRecord)
			return nil
		}
		return err
	}
	if err := json.Unmarshal(data, &s.Records); err != nil {
		return err
	}
	if s.Records == nil {
		s.Records = make(map[string]*ReputationRecord)
	}
	return nil
}

// Save writes the reputation store to disk atomically.
func (s *ReputationStore) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s.Records, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Score returns the current score for a provider key. 0 if unrated.
func (s *ReputationStore) Score(pubKeyHex string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if rec, ok := s.Records[pubKeyHex]; ok {
		return rec.Score
	}
	return 0
}

// Record applies a delta to the provider's score and saves to disk.
func (s *ReputationStore) Record(pubKeyHex string, delta int, source string) error {
	s.mu.Lock()
	rec, ok := s.Records[pubKeyHex]
	if !ok {
		rec = &ReputationRecord{Score: 0}
		s.Records[pubKeyHex] = rec
	}
	rec.Score += delta
	rec.LastUpdated = time.Now().UTC()
	if source != "" {
		rec.ReviewSource = source
	}
	s.mu.Unlock()
	return s.Save()
}

// DefaultReputationStorePath returns the standard path for the reputation.json file.
func DefaultReputationStorePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(configDir, "BlockchainVPN")
	return filepath.Join(appDir, "reputation.json"), nil
}
