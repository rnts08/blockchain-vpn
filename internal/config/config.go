package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	RPC      RPCConfig      `json:"rpc"`
	Provider ProviderConfig `json:"provider"`
	Client   ClientConfig   `json:"client"`
	Logging  LoggingConfig  `json:"logging"`
}

type RPCConfig struct {
	Host string `json:"host"`
	User string `json:"user"`
	Pass string `json:"pass"`
}

type LoggingConfig struct {
	Format string `json:"format"` // "text" or "json"
}

type ProviderConfig struct {
	InterfaceName            string `json:"interface_name"`
	ListenPort               int    `json:"listen_port"`
	AnnounceIP               string `json:"announce_ip"`
	Country                  string `json:"country"` // 2-letter country code (e.g., "US"). Leave empty to auto-detect.
	Price                    uint64 `json:"price_sats_per_session"`
	PrivateKeyFile           string `json:"private_key_file"`
	BandwidthLimit           string `json:"bandwidth_limit"` // e.g. "10mbit"
	EnableNAT                bool   `json:"enable_nat"`
	EnableEgressNAT          bool   `json:"enable_egress_nat"`
	NATOutboundInterface     string `json:"nat_outbound_interface"`
	IsolationMode            string `json:"isolation_mode"` // "none" or "sandbox"
	AllowlistFile            string `json:"allowlist_file"`
	DenylistFile             string `json:"denylist_file"`
	CertLifetimeHours        int    `json:"cert_lifetime_hours"`
	CertRotateBeforeHours    int    `json:"cert_rotate_before_hours"`
	HealthCheckEnabled       bool   `json:"health_check_enabled"`
	HealthCheckInterval      string `json:"health_check_interval"` // e.g. "30s"
	BandwidthMonitorInterval string `json:"bandwidth_monitor_interval"`
	TunIP                    string `json:"tun_ip"`
	TunSubnet                string `json:"tun_subnet"`
	MetricsListenAddr        string `json:"metrics_listen_addr"` // e.g. "127.0.0.1:9090"
}

type ClientConfig struct {
	InterfaceName     string `json:"interface_name"`
	TunIP             string `json:"tun_ip"`
	TunSubnet         string `json:"tun_subnet"`
	EnableKillSwitch  bool   `json:"enable_kill_switch"`
	MetricsListenAddr string `json:"metrics_listen_addr"` // e.g. "127.0.0.1:9091"
}

const AppConfigDirName = "BlockchainVPN"
const DefaultConfigFileName = "config.json"
const DefaultProviderKeyFileName = "provider.key"

func AppConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not resolve user config dir: %w", err)
	}
	dir := filepath.Join(base, AppConfigDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("could not create app config dir: %w", err)
	}
	return dir, nil
}

func DefaultConfigPath() (string, error) {
	dir, err := AppConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DefaultConfigFileName), nil
}

func DefaultProviderKeyPath() (string, error) {
	dir, err := AppConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DefaultProviderKeyFileName), nil
}

func ResolveProviderKeyPath(cfg *Config, cfgPath string) error {
	if cfg == nil {
		return nil
	}

	if strings.TrimSpace(cfg.Provider.PrivateKeyFile) == "" {
		p, err := DefaultProviderKeyPath()
		if err != nil {
			return err
		}
		cfg.Provider.PrivateKeyFile = p
		return nil
	}

	if filepath.IsAbs(cfg.Provider.PrivateKeyFile) {
		return nil
	}

	base := filepath.Dir(cfgPath)
	cfg.Provider.PrivateKeyFile = filepath.Join(base, cfg.Provider.PrivateKeyFile)
	return nil
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := &Config{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func GenerateDefaultConfig(path string) error {
	if path == "" {
		defaultPath, err := DefaultConfigPath()
		if err != nil {
			return err
		}
		path = defaultPath
	}

	keyPath, err := DefaultProviderKeyPath()
	if err != nil {
		return err
	}

	cfg := Config{
		RPC: RPCConfig{
			Host: "localhost:18443",
			User: "yourrpcuser",
			Pass: "yourrpcpassword",
		},
		Logging: LoggingConfig{
			Format: "text",
		},
		Provider: ProviderConfig{
			InterfaceName:            "bcvpn0",
			ListenPort:               51820,
			AnnounceIP:               "", // Leave empty to auto-detect public IP
			Country:                  "", // Leave empty to auto-detect country
			Price:                    1000,
			PrivateKeyFile:           keyPath,
			BandwidthLimit:           "10mbit",
			EnableNAT:                true,
			EnableEgressNAT:          false,
			NATOutboundInterface:     "",
			IsolationMode:            "none",
			AllowlistFile:            "",
			DenylistFile:             "",
			CertLifetimeHours:        720,
			CertRotateBeforeHours:    24,
			HealthCheckEnabled:       true,
			HealthCheckInterval:      "30s",
			BandwidthMonitorInterval: "30s",
			TunIP:                    "10.10.0.1",
			TunSubnet:                "24",
			MetricsListenAddr:        "",
		},
		Client: ClientConfig{
			InterfaceName:     "bcvpn1",
			TunIP:             "10.10.0.2",
			TunSubnet:         "24",
			EnableKillSwitch:  false,
			MetricsListenAddr: "",
		},
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(cfg)
}
