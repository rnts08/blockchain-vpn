package tunnel

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
)

type revocationCache struct {
	mu         sync.RWMutex
	path       string
	lastLoad   time.Time
	lastMtime  time.Time
	lastErr    error
	revokedSet map[string]struct{}
}

var globalRevocationCache = &revocationCache{
	revokedSet: map[string]struct{}{},
}

func (c *revocationCache) IsRevoked(path string, key *btcec.PublicKey) (bool, error) {
	if key == nil || strings.TrimSpace(path) == "" {
		return false, nil
	}
	if err := c.refresh(path); err != nil {
		return false, err
	}
	k := hex.EncodeToString(key.SerializeCompressed())
	c.mu.RLock()
	_, ok := c.revokedSet[k]
	c.mu.RUnlock()
	return ok, nil
}

func (c *revocationCache) refresh(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.path != path {
		c.path = path
		c.lastLoad = time.Time{}
		c.lastMtime = time.Time{}
		c.revokedSet = map[string]struct{}{}
		c.lastErr = nil
	}

	now := time.Now()
	if !c.lastLoad.IsZero() && now.Sub(c.lastLoad) < 5*time.Second {
		return c.lastErr
	}
	c.lastLoad = now

	info, err := os.Stat(path)
	if err != nil {
		c.lastErr = fmt.Errorf("revocation cache stat failed: %w", err)
		return c.lastErr
	}
	if info.ModTime().Equal(c.lastMtime) && c.lastErr == nil {
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		c.lastErr = fmt.Errorf("revocation cache open failed: %w", err)
		return c.lastErr
	}
	defer f.Close()

	next := map[string]struct{}{}
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.ToLower(line)
		raw, decErr := hex.DecodeString(line)
		if decErr != nil || len(raw) != 33 {
			c.lastErr = fmt.Errorf("invalid revoked key entry at line %d: %q", lineNo, line)
			return c.lastErr
		}
		if _, exists := next[line]; exists {
			c.lastErr = fmt.Errorf("duplicate revoked key entry at line %d: %q", lineNo, line)
			return c.lastErr
		}
		next[line] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		c.lastErr = fmt.Errorf("revocation cache parse failed: %w", err)
		return c.lastErr
	}

	c.revokedSet = next
	c.lastMtime = info.ModTime()
	c.lastErr = nil
	return nil
}
