package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"blockchain-vpn/internal/util"
)

type Config struct {
	RPC      RPCConfig      `json:"rpc"`
	Provider ProviderConfig `json:"provider"`
	Client   ClientConfig   `json:"client"`
	Logging  LoggingConfig  `json:"logging"`
	Security SecurityConfig `json:"security"`
}

type RPCConfig struct {
	Host        string `json:"host"`
	User        string `json:"user"`
	Pass        string `json:"pass"`
	EnableTLS   bool   `json:"enable_tls"`   // Enable TLS for RPC connections (recommended for production)
	Network     string `json:"network"`      // "mainnet", "testnet", "regtest", "simnet", or "auto"
	TokenSymbol string `json:"token_symbol"` // Display symbol for amounts (e.g., "BTC", "LTC", "ORDEX")
}

type LoggingConfig struct {
	Format string `json:"format"` // "text" or "json"
	Level  string `json:"level"`  // "debug", "info", "warn", "error"
}

type SecurityConfig struct {
	KeyStorageMode      string `json:"key_storage_mode"`      // "file", "keychain", "libsecret", "dpapi", or "auto"
	KeyStorageService   string `json:"key_storage_service"`   // service/namespace for secure store records
	RevocationCacheFile string `json:"revocation_cache_file"` // optional file containing revoked pubkeys (hex, one per line)
	TLSMinVersion       string `json:"tls_min_version"`       // "1.2" or "1.3"
	TLSProfile          string `json:"tls_profile"`           // "modern" or "compat"
	MetricsAuthToken    string `json:"metrics_auth_token"`    // optional token required by /metrics.json
}

type ProviderConfig struct {
	InterfaceName               string `json:"interface_name"`
	ListenPort                  int    `json:"listen_port"`
	AutoRotatePort              bool   `json:"auto_rotate_port"` // Automatically rotate to unprivileged port if bind fails
	AnnounceIP                  string `json:"announce_ip"`
	Country                     string `json:"country"` // 2-letter country code (e.g., "US"). Leave empty to auto-detect.
	Price                       uint64 `json:"price_sats_per_session"`
	MaxConsumers                int    `json:"max_consumers"` // 0 means unlimited
	PrivateKeyFile              string `json:"private_key_file"`
	BandwidthLimit              string `json:"bandwidth_limit"` // e.g. "10mbit"
	EnableNAT                   bool   `json:"enable_nat"`
	EnableEgressNAT             bool   `json:"enable_egress_nat"`
	NATOutboundInterface        string `json:"nat_outbound_interface"`
	IsolationMode               string `json:"isolation_mode"` // "none" or "sandbox"
	AllowlistFile               string `json:"allowlist_file"`
	DenylistFile                string `json:"denylist_file"`
	CertLifetimeHours           int    `json:"cert_lifetime_hours"`
	CertRotateBeforeHours       int    `json:"cert_rotate_before_hours"`
	HealthCheckEnabled          bool   `json:"health_check_enabled"`
	HealthCheckInterval         string `json:"health_check_interval"` // e.g. "30s"
	BandwidthMonitorInterval    string `json:"bandwidth_monitor_interval"`
	TunIP                       string `json:"tun_ip"`
	TunSubnet                   string `json:"tun_subnet"`
	MetricsListenAddr           string `json:"metrics_listen_addr"`       // e.g. "127.0.0.1:9090"
	MaxSessionDurationSecs      int    `json:"max_session_duration_secs"` // 0 = no limit
	AnnouncementFeeTargetBlocks int    `json:"announcement_fee_target_blocks"`
	AnnouncementFeeMode         string `json:"announcement_fee_mode"` // "conservative" or "economical"
	ThroughputProbePort         int    `json:"throughput_probe_port"` // 0 = disable provider-assisted probes
	WebSocketFallbackPort       int    `json:"websocket_fallback_port"`
	HeartbeatInterval           string `json:"heartbeat_interval"`       // e.g. "5m"
	PaymentMonitorInterval      string `json:"payment_monitor_interval"` // e.g. "1m"

	// New fields for flexible pricing
	PricingMethod   string `json:"pricing_method"`    // "session", "time", "data"
	BillingTimeUnit string `json:"billing_time_unit"` // "minute", "hour"
	BillingDataUnit string `json:"billing_data_unit"` // "MB", "GB"
}

type ClientConfig struct {
	InterfaceName              string `json:"interface_name"`
	TunIP                      string `json:"tun_ip"`
	TunSubnet                  string `json:"tun_subnet"`
	EnableKillSwitch           bool   `json:"enable_kill_switch"`
	MetricsListenAddr          string `json:"metrics_listen_addr"` // e.g. "127.0.0.1:9091"
	StrictVerification         bool   `json:"strict_verification"`
	VerifyThroughputAfterSetup bool   `json:"verify_throughput_after_connect"`
	MaxParallelTunnels         int    `json:"max_parallel_tunnels"`
	EnableWebSocketFallback    bool   `json:"enable_websocket_fallback"`

	// Spending limits
	SpendingLimitEnabled   bool   `json:"spending_limit_enabled"`    // Enable total spending limit
	SpendingLimitSats      uint64 `json:"spending_limit_sats"`       // Total spending cap in satoshis
	SpendingWarningPercent uint32 `json:"spending_warning_percent"`  // Warning threshold (0-100)
	AutoDisconnectOnLimit  bool   `json:"auto_disconnect_on_limit"`  // Auto-disconnect when limit reached
	MaxSessionSpendingSats uint64 `json:"max_session_spending_sats"` // Max spend per session (0 = unlimited)

	// Auto-recharge (prepaid credit)
	AutoRechargeEnabled    bool   `json:"auto_recharge_enabled"`     // Enable automatic recharge when credits run low
	AutoRechargeThreshold  uint64 `json:"auto_recharge_threshold"`   // Sats remaining before auto-recharge triggers
	AutoRechargeAmount     uint64 `json:"auto_recharge_amount"`      // Sats to add on auto-recharge
	AutoRechargeMinBalance uint64 `json:"auto_recharge_min_balance"` // Minimum balance to maintain
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
	if cfg.Client.MaxParallelTunnels <= 0 {
		cfg.Client.MaxParallelTunnels = 1
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
			Host:      "localhost:25173",
			User:      "rpcuser",
			Pass:      "",
			EnableTLS: false,
		},
		Logging: LoggingConfig{
			Format: "text",
			Level:  "info",
		},
		Security: SecurityConfig{
			KeyStorageMode:      "file",
			KeyStorageService:   "BlockchainVPN",
			RevocationCacheFile: "",
			TLSMinVersion:       "1.3",
			TLSProfile:          "modern",
			MetricsAuthToken:    "",
		},
		Provider: ProviderConfig{
			InterfaceName:               "bcvpn0",
			ListenPort:                  51820,
			AnnounceIP:                  "", // Leave empty to auto-detect public IP
			Country:                     "", // Leave empty to auto-detect country
			Price:                       1000,
			MaxConsumers:                0,
			PrivateKeyFile:              keyPath,
			BandwidthLimit:              "10mbit",
			EnableNAT:                   true,
			EnableEgressNAT:             false,
			NATOutboundInterface:        "",
			IsolationMode:               "none",
			AllowlistFile:               "",
			DenylistFile:                "",
			CertLifetimeHours:           720,
			CertRotateBeforeHours:       24,
			HealthCheckEnabled:          true,
			MaxSessionDurationSecs:      0,
			AnnouncementFeeTargetBlocks: 6,
			AnnouncementFeeMode:         "conservative",
			ThroughputProbePort:         51821,
			WebSocketFallbackPort:       0, // Disabled by default
			HeartbeatInterval:           "5m",
			PaymentMonitorInterval:      "30s",
			// New flexible pricing fields (default to session-based)
			PricingMethod:   "session",
			BillingTimeUnit: "minute",
			BillingDataUnit: "GB",
		},
		Client: ClientConfig{
			InterfaceName:              "bcvpn1",
			TunIP:                      "10.10.0.2",
			TunSubnet:                  "24",
			EnableKillSwitch:           false,
			MetricsListenAddr:          "",
			StrictVerification:         false,
			VerifyThroughputAfterSetup: true,
			MaxParallelTunnels:         1,
			EnableWebSocketFallback:    false,
			// Spending limits
			SpendingLimitEnabled:   false,
			SpendingLimitSats:      0,
			SpendingWarningPercent: 80,
			AutoDisconnectOnLimit:  false,
			MaxSessionSpendingSats: 0,
			// Auto-recharge
			AutoRechargeEnabled:    false,
			AutoRechargeThreshold:  500,
			AutoRechargeAmount:     1000,
			AutoRechargeMinBalance: 100,
		},
	}

	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cfg); err != nil {
		return err
	}
	return util.WriteFileAtomic(path, out.Bytes(), 0o644)
}
