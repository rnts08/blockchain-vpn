package tunnel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"blockchain-vpn/internal/config"
)

type networkCleanupMarker struct {
	OS             string   `json:"os"`
	CreatedAt      string   `json:"created_at"`
	IfaceName      string   `json:"iface_name"`
	ProviderHost   string   `json:"provider_host"`
	DefaultGW      string   `json:"default_gw,omitempty"`
	DefaultIfAlias string   `json:"default_if_alias,omitempty"`
	TunIfIndex     int      `json:"tun_if_index,omitempty"`
	DNSService     string   `json:"dns_service,omitempty"`
	DNSServers     []string `json:"dns_servers,omitempty"`
	DNSConfigured  bool     `json:"dns_configured"`
}

func cleanupMarkerPath() (string, error) {
	dir, err := config.AppConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "network-cleanup-marker.json"), nil
}

func writeCleanupMarker(m networkCleanupMarker) error {
	p, err := cleanupMarkerPath()
	if err != nil {
		return err
	}
	m.OS = runtime.GOOS
	m.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

func readCleanupMarker() (*networkCleanupMarker, error) {
	p, err := cleanupMarkerPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var m networkCleanupMarker
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func clearCleanupMarker() {
	p, err := cleanupMarkerPath()
	if err != nil {
		return
	}
	_ = os.Remove(p)
}

func RecoverPendingNetworkState() error {
	m, err := readCleanupMarker()
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if m.OS != runtime.GOOS {
		clearCleanupMarker()
		return nil
	}
	if err := recoverPendingNetworkStateFromMarker(m); err != nil {
		return fmt.Errorf("failed to recover pending network state: %w", err)
	}
	clearCleanupMarker()
	return nil
}
