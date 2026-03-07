package tunnel

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
)

type accessPolicy struct {
	allow map[string]bool
	deny  map[string]bool
}

func loadAccessPolicy(allowlistFile, denylistFile string) (*accessPolicy, error) {
	allow, err := loadKeyFile(allowlistFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load allowlist: %w", err)
	}
	deny, err := loadKeyFile(denylistFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load denylist: %w", err)
	}
	return &accessPolicy{allow: allow, deny: deny}, nil
}

func (p *accessPolicy) check(peer *btcec.PublicKey) error {
	if p == nil || peer == nil {
		return nil
	}
	key := hex.EncodeToString(peer.SerializeCompressed())
	if p.deny[key] {
		return fmt.Errorf("peer key denied by denylist")
	}
	if len(p.allow) > 0 && !p.allow[key] {
		return fmt.Errorf("peer key not present in allowlist")
	}
	return nil
}

func loadKeyFile(path string) (map[string]bool, error) {
	keys := make(map[string]bool)
	if strings.TrimSpace(path) == "" {
		return keys, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		keyHex := strings.ToLower(line)
		raw, err := hex.DecodeString(keyHex)
		if err != nil {
			return nil, fmt.Errorf("%s:%d invalid hex: %w", path, lineNo, err)
		}
		if _, err := btcec.ParsePubKey(raw); err != nil {
			return nil, fmt.Errorf("%s:%d invalid compressed secp256k1 pubkey: %w", path, lineNo, err)
		}
		keys[keyHex] = true
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}
