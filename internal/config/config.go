package config

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"blockchain-vpn/internal/util"
)

type Config struct {
	RPC      RPCConfig      `json:"rpc"`
	Provider ProviderConfig `json:"provider"`
	Client   ClientConfig   `json:"client"`
	Security SecurityConfig `json:"security"`
	Logging  LoggingConfig  `json:"logging"`
	DemoMode bool           `json:"demo_mode"` // Enable simulation mode for GUI testing without backend
}

type RPCConfig struct {
	Host        string `json:"host"`
	User        string `json:"user"`
	Pass        string `json:"pass"`
	EnableTLS   bool   `json:"enable_tls"`   // Enable TLS for RPC connections (recommended for production)
	Network     string `json:"network"`      // "mainnet", "testnet", "regtest", "simnet", or "auto"
	TokenSymbol string `json:"token_symbol"` // Display symbol for amounts (e.g., "BTC", "LTC", "ORDEX")
	CookieFile  string `json:"cookie_file"`  // Path to RPC cookie file (optional, auto-detected if empty)
}

type LoggingConfig struct {
	Format string `json:"format"` // "text" or "json"
	Level  string `json:"level"`  // "debug", "info", "warn", "error"
}

type SecurityConfig struct {
	KeyStorageMode        string   `json:"key_storage_mode"`         // "file", "keychain", "libsecret", "dpapi", or "auto"
	KeyStorageService     string   `json:"key_storage_service"`      // service/namespace for secure store records
	RevocationCacheFile   string   `json:"revocation_cache_file"`    // optional file containing revoked pubkeys (hex, one per line)
	TLSMinVersion         string   `json:"tls_min_version"`          // "1.2" or "1.3"
	TLSProfile            string   `json:"tls_profile"`              // "modern" or "compat"
	TlsCustomCipherSuites []string `json:"tls_custom_cipher_suites"` // custom cipher suite names (e.g., ["ECDHE-RSA-AES128-GCM-SHA256"])
	MetricsAuthToken      string   `json:"metrics_auth_token"`       // optional token required by /metrics.json
}

type ProviderConfig struct {
	InterfaceName                string   `json:"interface_name"`
	ListenPort                   int      `json:"listen_port"`
	AutoRotatePort               bool     `json:"auto_rotate_port"` // Automatically rotate to unprivileged port if bind fails
	AnnounceIP                   string   `json:"announce_ip"`
	Country                      string   `json:"country"` // 2-letter country code (e.g., "US"). Leave empty to auto-detect.
	Price                        uint64   `json:"price_sats_per_session"`
	MaxConsumers                 int      `json:"max_consumers"` // 0 means unlimited
	PrivateKeyFile               string   `json:"private_key_file"`
	BandwidthLimit               string   `json:"bandwidth_limit"`     // e.g. "10mbit", "0" or empty = auto-test
	BandwidthAutoTest            bool     `json:"bandwidth_auto_test"` // Run speed test to determine max bandwidth
	DNSServers                   []string `json:"dns_servers"`         // Custom DNS servers (e.g., ["1.1.1.1", "8.8.8.8"])
	EnableNAT                    bool     `json:"enable_nat"`
	EnableEgressNAT              bool     `json:"enable_egress_nat"`
	NATOutboundInterface         string   `json:"nat_outbound_interface"`
	NATTraversalMethod           string   `json:"nat_traversal_method"` // "auto", "upnp", "natpmp", "none"
	IsolationMode                string   `json:"isolation_mode"`       // "none" or "sandbox"
	AllowlistFile                string   `json:"allowlist_file"`
	DenylistFile                 string   `json:"denylist_file"`
	CertLifetimeHours            int      `json:"cert_lifetime_hours"`
	CertRotateBeforeHours        int      `json:"cert_rotate_before_hours"`
	HealthCheckEnabled           bool     `json:"health_check_enabled"`
	HealthCheckInterval          string   `json:"health_check_interval"` // e.g. "30s"
	BandwidthMonitorInterval     string   `json:"bandwidth_monitor_interval"`
	AnnouncementInterval         string   `json:"announcement_interval"` // e.g. "24h"
	TunIP                        string   `json:"tun_ip"`
	TunSubnet                    string   `json:"tun_subnet"`
	MetricsListenAddr            string   `json:"metrics_listen_addr"`       // e.g. "127.0.0.1:9090"
	MaxSessionDurationSecs       int      `json:"max_session_duration_secs"` // 0 = no limit
	AnnouncementFeeTargetBlocks  int      `json:"announcement_fee_target_blocks"`
	AnnouncementFeeMode          string   `json:"announcement_fee_mode"` // "conservative" or "economical"
	ThroughputProbePort          int      `json:"throughput_probe_port"` // 0 = disable provider-assisted probes
	WebSocketFallbackPort        int      `json:"websocket_fallback_port"`
	HeartbeatInterval            string   `json:"heartbeat_interval"`             // e.g. "5m"
	PaymentMonitorInterval       string   `json:"payment_monitor_interval"`       // e.g. "1m"
	PaymentRequiredConfirmations int      `json:"payment_required_confirmations"` // Min confirmations before granting access (0-6, default 1)
	ShutdownTimeout              string   `json:"shutdown_timeout"`               // e.g. "10s"
	PIDFile                      string   `json:"pid_file"`                       // Path to PID file (default: config dir/provider.pid)

	// New fields for flexible pricing
	PricingMethod   string `json:"pricing_method"`    // "session", "time", "data"
	BillingTimeUnit string `json:"billing_time_unit"` // "minute", "hour"
	BillingDataUnit string `json:"billing_data_unit"` // "MB", "GB"
}

type ClientConfig struct {
	InterfaceName              string   `json:"interface_name"`
	TunIP                      string   `json:"tun_ip"`
	TunSubnet                  string   `json:"tun_subnet"`
	DNSServers                 []string `json:"dns_servers"` // Custom DNS servers (e.g., ["1.1.1.1", "8.8.8.8"])
	EnableKillSwitch           bool     `json:"enable_kill_switch"`
	MetricsListenAddr          string   `json:"metrics_listen_addr"` // e.g. "127.0.0.1:9091"
	StrictVerification         bool     `json:"strict_verification"`
	VerifyThroughputAfterSetup bool     `json:"verify_throughput_after_connect"`
	MaxParallelTunnels         int      `json:"max_parallel_tunnels"`
	EnableWebSocketFallback    bool     `json:"enable_websocket_fallback"`

	// Favorite providers (pubkeys)
	FavoriteProviders []string `json:"favorite_providers"` // List of trusted provider pubkeys (hex)

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

	// Auto-reconnect
	AutoReconnectEnabled     bool   `json:"auto_reconnect_enabled"`      // Enable automatic reconnection on disconnect
	AutoReconnectMaxAttempts int    `json:"auto_reconnect_max_attempts"` // Max reconnection attempts (0 = infinite)
	AutoReconnectInterval    string `json:"auto_reconnect_interval"`     // Base interval between retries (e.g., "5s", "30s")
	AutoReconnectMaxInterval string `json:"auto_reconnect_max_interval"` // Max interval cap (e.g., "5m")
}

const AppConfigDirName = "blockchain-vpn"
const LegacyAppConfigDirName = "BlockchainVPN"
const DefaultConfigFileName = "config.json"
const DefaultProviderKeyFileName = "provider.key"
const DefaultProviderPIDFileName = "provider.pid"

var configDirVariants = []string{
	"blockchain-vpn",
	"BlockchainVPN",
	"blockchainvpn",
	"BlockchainVPN ",
}

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

func ResolveConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not resolve user config dir: %w", err)
	}

	for _, variant := range configDirVariants {
		dir := filepath.Join(base, variant)
		info, err := os.Stat(dir)
		if err == nil && info.IsDir() {
			if variant != AppConfigDirName {
				log.Printf("Note: Found config directory at legacy location %q, consider migrating to %q", dir, filepath.Join(base, AppConfigDirName))
			}
			return dir, nil
		}
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

func ResolveConfigPath() (string, error) {
	dir, err := ResolveConfigDir()
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

func ResolveDefaultProviderKeyPath() (string, error) {
	dir, err := ResolveConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DefaultProviderKeyFileName), nil
}

func ResolveProviderPIDFilePath(cfg *Config, cfgPath string) (string, error) {
	// If provider.pid_file is set explicitly, use it (relative to config dir or absolute)
	if strings.TrimSpace(cfg.Provider.PIDFile) != "" {
		if filepath.IsAbs(cfg.Provider.PIDFile) {
			return cfg.Provider.PIDFile, nil
		}
		// Relative to config directory
		dir := filepath.Dir(cfgPath)
		return filepath.Join(dir, cfg.Provider.PIDFile), nil
	}
	// Default: use config directory with standard name
	dir, err := ResolveConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DefaultProviderPIDFileName), nil
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

// GenerateRandomRPCPassword generates a secure random password for RPC authentication.
func GenerateRandomRPCPassword() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random password: %w", err)
	}
	return hex.EncodeToString(bytes), nil
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

	applyConfigDefaults(cfg)

	return cfg, nil
}

// applyConfigDefaults applies default values to all config fields that need them.
func applyConfigDefaults(cfg *Config) {
	if cfg.Client.MaxParallelTunnels <= 0 {
		cfg.Client.MaxParallelTunnels = 1
	}

	if cfg.Provider.HealthCheckInterval == "" {
		cfg.Provider.HealthCheckInterval = "30s"
	}
	if cfg.Provider.BandwidthMonitorInterval == "" {
		cfg.Provider.BandwidthMonitorInterval = "30s"
	}
	if cfg.Provider.AnnouncementInterval == "" {
		cfg.Provider.AnnouncementInterval = "24h"
	}
	if cfg.Provider.ShutdownTimeout == "" {
		cfg.Provider.ShutdownTimeout = "10s"
	}
	if len(cfg.Provider.DNSServers) == 0 {
		cfg.Provider.DNSServers = []string{"1.1.1.1", "8.8.8.8"}
	}
	if len(cfg.Client.DNSServers) == 0 {
		cfg.Client.DNSServers = []string{"1.1.1.1", "8.8.8.8"}
	}
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

	rpcPass, err := GenerateRandomRPCPassword()
	if err != nil {
		return err
	}

	cfg := Config{
		RPC: RPCConfig{
			Host:        "localhost:25173",
			User:        "rpcuser",
			Pass:        rpcPass,
			EnableTLS:   false,
			Network:     "mainnet",
			TokenSymbol: "OXC",
		},
		Logging: LoggingConfig{
			Format: "text",
			Level:  "info",
		},
		Security: SecurityConfig{
			KeyStorageMode:    "file",
			KeyStorageService: "BlockchainVPN",
			TLSMinVersion:     "1.3",
			TLSProfile:        "modern",
		},
		Provider: ProviderConfig{
			InterfaceName:  "bcvpn0",
			ListenPort:     51820,
			Price:          1000,
			PrivateKeyFile: keyPath,
			TunIP:          "10.0.0.1",
			TunSubnet:      "24",
			PricingMethod:  "session",
		},
		Client: ClientConfig{
			InterfaceName: "bcvpn1",
			TunIP:         "10.10.0.2",
			TunSubnet:     "24",
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
