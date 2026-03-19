package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyConfigDefaults_ClientOnly(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		RPC: RPCConfig{
			Host: "localhost:18443",
			User: "rpcuser",
			Pass: "secret",
		},
		Client: ClientConfig{
			InterfaceName: "bcvpn1",
			TunIP:         "10.10.0.2",
			TunSubnet:     "24",
		},
	}

	applyConfigDefaults(cfg)

	if cfg.Client.MaxParallelTunnels != 1 {
		t.Errorf("expected MaxParallelTunnels=1, got %d", cfg.Client.MaxParallelTunnels)
	}
	if cfg.Client.EnableKillSwitch != false {
		t.Errorf("expected EnableKillSwitch=false, got %v", cfg.Client.EnableKillSwitch)
	}
	if len(cfg.Client.DNSServers) != 2 {
		t.Errorf("expected 2 DNS servers, got %d", len(cfg.Client.DNSServers))
	}
}

func TestApplyConfigDefaults_ProviderOnly(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		RPC: RPCConfig{
			Host: "localhost:18443",
			User: "rpcuser",
			Pass: "secret",
		},
		Provider: ProviderConfig{
			InterfaceName: "bcvpn0",
			ListenPort:    51820,
			TunIP:         "10.0.0.1",
			TunSubnet:     "24",
		},
	}

	applyConfigDefaults(cfg)

	if cfg.Provider.BandwidthLimit != "" {
		t.Errorf("expected BandwidthLimit empty, got %s", cfg.Provider.BandwidthLimit)
	}
	if cfg.Provider.HealthCheckInterval != "30s" {
		t.Errorf("expected HealthCheckInterval=30s, got %s", cfg.Provider.HealthCheckInterval)
	}
	if cfg.Provider.ShutdownTimeout != "10s" {
		t.Errorf("expected ShutdownTimeout=10s, got %s", cfg.Provider.ShutdownTimeout)
	}
}

func TestApplyConfigDefaults_NilFields(t *testing.T) {
	t.Parallel()
	cfg := &Config{}

	applyConfigDefaults(cfg)

	if cfg.Client.MaxParallelTunnels != 1 {
		t.Errorf("expected MaxParallelTunnels=1, got %d", cfg.Client.MaxParallelTunnels)
	}
	if cfg.Provider.HealthCheckInterval != "30s" {
		t.Errorf("expected HealthCheckInterval=30s, got %s", cfg.Provider.HealthCheckInterval)
	}
}

func TestApplyConfigDefaults_PreservesExisting(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Client: ClientConfig{
			EnableKillSwitch:   true,
			MaxParallelTunnels: 5,
			DNSServers:         []string{"9.9.9.9"},
		},
		Provider: ProviderConfig{
			HealthCheckInterval: "5m",
			BandwidthLimit:      "100mbit",
		},
	}

	applyConfigDefaults(cfg)

	if cfg.Client.EnableKillSwitch != true {
		t.Errorf("expected EnableKillSwitch=true preserved, got %v", cfg.Client.EnableKillSwitch)
	}
	if cfg.Client.MaxParallelTunnels != 5 {
		t.Errorf("expected MaxParallelTunnels=5 preserved, got %d", cfg.Client.MaxParallelTunnels)
	}
	if cfg.Provider.BandwidthLimit != "100mbit" {
		t.Errorf("expected BandwidthLimit=100mbit preserved, got %s", cfg.Provider.BandwidthLimit)
	}
}

func TestSetupWizardConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "client only config is valid",
			cfg: &Config{
				RPC: RPCConfig{
					Host: "localhost:18443",
					User: "user",
					Pass: "pass",
				},
				Client: ClientConfig{
					InterfaceName:      "bcvpn1",
					TunIP:              "10.10.0.2",
					TunSubnet:          "24",
					MaxParallelTunnels: 1,
					DNSServers:         []string{"1.1.1.1", "8.8.8.8"},
				},
				Logging: LoggingConfig{
					Format: "text",
					Level:  "info",
				},
				Security: SecurityConfig{
					KeyStorageMode: "file",
					TLSMinVersion:  "1.3",
					TLSProfile:     "modern",
				},
			},
			wantErr: false,
		},
		{
			name: "provider only config is valid",
			cfg: &Config{
				RPC: RPCConfig{
					Host: "localhost:18443",
					User: "user",
					Pass: "pass",
				},
				Provider: ProviderConfig{
					InterfaceName:            "bcvpn0",
					ListenPort:               51820,
					TunIP:                    "10.0.0.1",
					TunSubnet:                "24",
					CertLifetimeHours:        720,
					CertRotateBeforeHours:    24,
					HealthCheckInterval:      "30s",
					BandwidthMonitorInterval: "30s",
					AnnouncementInterval:     "24h",
					HeartbeatInterval:        "5m",
					ShutdownTimeout:          "10s",
					DNSServers:               []string{"1.1.1.1", "8.8.8.8"},
				},
				Logging: LoggingConfig{
					Format: "text",
					Level:  "info",
				},
				Security: SecurityConfig{
					KeyStorageMode: "file",
					TLSMinVersion:  "1.3",
					TLSProfile:     "modern",
				},
			},
			wantErr: false,
		},
		{
			name: "both mode config is valid",
			cfg: &Config{
				RPC: RPCConfig{
					Host:    "localhost:18443",
					User:    "user",
					Pass:    "secret",
					Network: "mainnet",
				},
				Provider: ProviderConfig{
					InterfaceName:            "bcvpn0",
					ListenPort:               51820,
					TunIP:                    "10.0.0.1",
					TunSubnet:                "24",
					CertLifetimeHours:        720,
					CertRotateBeforeHours:    24,
					HealthCheckInterval:      "30s",
					BandwidthMonitorInterval: "30s",
					AnnouncementInterval:     "24h",
					HeartbeatInterval:        "5m",
					ShutdownTimeout:          "10s",
					DNSServers:               []string{"1.1.1.1", "8.8.8.8"},
				},
				Client: ClientConfig{
					InterfaceName:      "bcvpn1",
					TunIP:              "10.10.0.2",
					TunSubnet:          "24",
					MaxParallelTunnels: 1,
					DNSServers:         []string{"1.1.1.1", "8.8.8.8"},
				},
				Logging: LoggingConfig{
					Format: "text",
					Level:  "info",
				},
				Security: SecurityConfig{
					KeyStorageMode: "file",
					TLSMinVersion:  "1.3",
					TLSProfile:     "modern",
				},
			},
			wantErr: false,
		},
		{
			name: "zero port fails",
			cfg: &Config{
				RPC: RPCConfig{
					Host: "localhost:18443",
					User: "user",
					Pass: "pass",
				},
				Provider: ProviderConfig{
					InterfaceName: "bcvpn0",
					ListenPort:    0,
					TunIP:         "10.0.0.1",
					TunSubnet:     "24",
				},
				Client: ClientConfig{
					InterfaceName: "bcvpn1",
					TunIP:         "10.10.0.2",
					TunSubnet:     "24",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid TLS version fails",
			cfg: &Config{
				RPC: RPCConfig{
					Host: "localhost:18443",
					User: "user",
					Pass: "pass",
				},
				Client: ClientConfig{
					InterfaceName: "bcvpn1",
					TunIP:         "10.10.0.2",
					TunSubnet:     "24",
				},
				Security: SecurityConfig{
					TLSMinVersion: "1.1",
					TLSProfile:    "modern",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log format fails",
			cfg: &Config{
				RPC: RPCConfig{
					Host: "localhost:18443",
					User: "user",
					Pass: "pass",
				},
				Client: ClientConfig{
					InterfaceName: "bcvpn1",
					TunIP:         "10.10.0.2",
					TunSubnet:     "24",
				},
				Logging: LoggingConfig{
					Format: "xml",
					Level:  "info",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSetupConfigRoundTrip(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		RPC: RPCConfig{
			Host:    "localhost:25173",
			User:    "rpcuser",
			Pass:    "auto-generated-password",
			Network: "mainnet",
		},
		Logging: LoggingConfig{
			Format: "text",
			Level:  "info",
		},
		Security: SecurityConfig{
			KeyStorageMode: "file",
			TLSMinVersion:  "1.3",
			TLSProfile:     "modern",
		},
		Provider: ProviderConfig{
			InterfaceName:            "bcvpn0",
			ListenPort:               51820,
			TunIP:                    "10.0.0.1",
			TunSubnet:                "24",
			DNSServers:               []string{"1.1.1.1", "8.8.8.8"},
			CertLifetimeHours:        720,
			CertRotateBeforeHours:    24,
			HealthCheckInterval:      "30s",
			BandwidthMonitorInterval: "30s",
			AnnouncementInterval:     "24h",
			HeartbeatInterval:        "5m",
			ShutdownTimeout:          "10s",
			MetricsListenAddr:        "127.0.0.1:9090",
		},
		Client: ClientConfig{
			InterfaceName:      "bcvpn1",
			TunIP:              "10.10.0.2",
			TunSubnet:          "24",
			DNSServers:         []string{"1.1.1.1", "8.8.8.8"},
			MaxParallelTunnels: 1,
		},
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	dec := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	decoded := &Config{}
	if err := dec.Decode(decoded); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if decoded.RPC.Host != cfg.RPC.Host {
		t.Errorf("RPC.Host = %q, want %q", decoded.RPC.Host, cfg.RPC.Host)
	}
	if decoded.RPC.User != cfg.RPC.User {
		t.Errorf("RPC.User = %q, want %q", decoded.RPC.User, cfg.RPC.User)
	}
	if decoded.Logging.Format != cfg.Logging.Format {
		t.Errorf("Logging.Format = %q, want %q", decoded.Logging.Format, cfg.Logging.Format)
	}
	if decoded.Security.TLSMinVersion != cfg.Security.TLSMinVersion {
		t.Errorf("Security.TLSMinVersion = %q, want %q", decoded.Security.TLSMinVersion, cfg.Security.TLSMinVersion)
	}
	if decoded.Provider.ListenPort != cfg.Provider.ListenPort {
		t.Errorf("Provider.ListenPort = %d, want %d", decoded.Provider.ListenPort, cfg.Provider.ListenPort)
	}
	if decoded.Client.TunIP != cfg.Client.TunIP {
		t.Errorf("Client.TunIP = %q, want %q", decoded.Client.TunIP, cfg.Client.TunIP)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "invalid.json")
	if err := os.WriteFile(path, []byte("{invalid json}"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	_, err := LoadConfig(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadConfig_Valid(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		RPC: RPCConfig{
			Host: "localhost:18443",
			User: "user",
			Pass: "pass",
		},
		Client: ClientConfig{
			InterfaceName: "bcvpn1",
			TunIP:         "10.10.0.2",
			TunSubnet:     "24",
		},
	}

	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.json")

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		t.Fatalf("Encode error = %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loaded.RPC.Host != cfg.RPC.Host {
		t.Errorf("RPC.Host = %q, want %q", loaded.RPC.Host, cfg.RPC.Host)
	}
}

func TestResolveProviderKeyPath_Relative(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Provider: ProviderConfig{
			PrivateKeyFile: "provider.key",
		},
	}
	err := ResolveProviderKeyPath(cfg, "/etc/bcvpn/config.json")
	if err != nil {
		t.Fatalf("ResolveProviderKeyPath() error = %v", err)
	}
	expected := "/etc/bcvpn/provider.key"
	if cfg.Provider.PrivateKeyFile != expected {
		t.Errorf("PrivateKeyFile = %q, want %q", cfg.Provider.PrivateKeyFile, expected)
	}
}

func TestResolveProviderKeyPath_Absolute(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Provider: ProviderConfig{
			PrivateKeyFile: "/absolute/path/provider.key",
		},
	}
	err := ResolveProviderKeyPath(cfg, "/etc/bcvpn/config.json")
	if err != nil {
		t.Fatalf("ResolveProviderKeyPath() error = %v", err)
	}
	if cfg.Provider.PrivateKeyFile != "/absolute/path/provider.key" {
		t.Errorf("PrivateKeyFile = %q, want %q", cfg.Provider.PrivateKeyFile, "/absolute/path/provider.key")
	}
}

func TestResolveProviderKeyPath_Empty(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Provider: ProviderConfig{
			PrivateKeyFile: "",
		},
	}
	err := ResolveProviderKeyPath(cfg, "/etc/bcvpn/config.json")
	if err != nil {
		t.Fatalf("ResolveProviderKeyPath() error = %v", err)
	}
	if cfg.Provider.PrivateKeyFile == "" {
		t.Error("PrivateKeyFile should be set to default")
	}
}

func TestResolveProviderKeyPath_NilConfig(t *testing.T) {
	err := ResolveProviderKeyPath(nil, "/etc/bcvpn/config.json")
	if err != nil {
		t.Fatalf("ResolveProviderKeyPath(nil) error = %v", err)
	}
}

func TestResolveProviderPIDFilePath_Default(t *testing.T) {
	t.Parallel()
	cfg := &Config{}

	pidPath, err := ResolveProviderPIDFilePath(cfg, "/tmp/config.json")
	if err != nil {
		t.Fatalf("ResolveProviderPIDFilePath() error = %v", err)
	}

	if !strings.HasSuffix(pidPath, "provider.pid") {
		t.Errorf("PIDFile = %q, want to end with provider.pid", pidPath)
	}
}

func TestResolveProviderPIDFilePath_Custom(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Provider: ProviderConfig{
			PIDFile: "custom.pid",
		},
	}

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	pidPath, err := ResolveProviderPIDFilePath(cfg, cfgPath)
	if err != nil {
		t.Fatalf("ResolveProviderPIDFilePath() error = %v", err)
	}

	base := filepath.Dir(cfgPath)
	expected := filepath.Join(base, "custom.pid")
	if pidPath != expected {
		t.Errorf("PIDFile = %q, want %q", pidPath, expected)
	}
}

func TestResolveProviderPIDFilePath_Absolute(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Provider: ProviderConfig{
			PIDFile: "/var/run/bcvpn/provider.pid",
		},
	}

	pidPath, err := ResolveProviderPIDFilePath(cfg, "/etc/bcvpn/config.json")
	if err != nil {
		t.Fatalf("ResolveProviderPIDFilePath() error = %v", err)
	}

	if pidPath != "/var/run/bcvpn/provider.pid" {
		t.Errorf("PIDFile = %q, want %q", pidPath, "/var/run/bcvpn/provider.pid")
	}
}

func TestValidatePricingMethods(t *testing.T) {
	baseCfg := func() *Config {
		return &Config{
			RPC: RPCConfig{
				Host: "localhost:18443",
				User: "user",
				Pass: "pass",
			},
			Provider: ProviderConfig{
				InterfaceName: "bcvpn0",
				ListenPort:    51820,
				TunIP:         "10.0.0.1",
				TunSubnet:     "24",
			},
			Client: ClientConfig{
				InterfaceName: "bcvpn1",
				TunIP:         "10.10.0.2",
				TunSubnet:     "24",
			},
		}
	}

	tests := []struct {
		name    string
		method  string
		unit    string
		wantErr bool
	}{
		{"session pricing valid", "session", "", false},
		{"time pricing requires billing unit", "time", "", true},
		{"time pricing with unit valid", "time", "minute", false},
		{"data pricing requires billing unit", "data", "", true},
		{"data pricing with unit valid", "data", "mb", false},
		{"empty pricing valid (defaults)", "", "", false},
		{"invalid pricing method", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseCfg()
			cfg.Provider.PricingMethod = tt.method
			if tt.method == "time" {
				cfg.Provider.BillingTimeUnit = tt.unit
			}
			if tt.method == "data" {
				cfg.Provider.BillingDataUnit = tt.unit
			}
			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateIsolationModes(t *testing.T) {
	baseCfg := func() *Config {
		return &Config{
			RPC: RPCConfig{
				Host: "localhost:18443",
				User: "user",
				Pass: "pass",
			},
			Provider: ProviderConfig{
				InterfaceName: "bcvpn0",
				ListenPort:    51820,
				TunIP:         "10.0.0.1",
				TunSubnet:     "24",
			},
			Client: ClientConfig{
				InterfaceName: "bcvpn1",
				TunIP:         "10.10.0.2",
				TunSubnet:     "24",
			},
		}
	}

	tests := []struct {
		mode    string
		wantErr bool
	}{
		{"none", false},
		{"sandbox", false},
		{"", false},
		{"client", true},
		{"full", true},
		{"invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			cfg := baseCfg()
			cfg.Provider.IsolationMode = tt.mode
			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRPCNetwork(t *testing.T) {
	baseCfg := func() *Config {
		return &Config{
			RPC: RPCConfig{
				Host:    "localhost:18443",
				User:    "user",
				Pass:    "pass",
				Network: "",
			},
			Provider: ProviderConfig{
				InterfaceName: "bcvpn0",
				ListenPort:    51820,
				TunIP:         "10.0.0.1",
				TunSubnet:     "24",
			},
			Client: ClientConfig{
				InterfaceName: "bcvpn1",
				TunIP:         "10.10.0.2",
				TunSubnet:     "24",
			},
		}
	}

	tests := []struct {
		network string
		wantErr bool
	}{
		{"mainnet", false},
		{"testnet", false},
		{"regtest", false},
		{"simnet", false},
		{"", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			cfg := baseCfg()
			cfg.RPC.Network = tt.network
			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v for network %q, wantErr %v", err, tt.network, tt.wantErr)
			}
		})
	}
}

func TestValidateTLSProfiles(t *testing.T) {
	baseCfg := func() *Config {
		return &Config{
			RPC: RPCConfig{
				Host: "localhost:18443",
				User: "user",
				Pass: "pass",
			},
			Client: ClientConfig{
				InterfaceName: "bcvpn1",
				TunIP:         "10.10.0.2",
				TunSubnet:     "24",
			},
			Security: SecurityConfig{
				KeyStorageMode: "file",
				TLSMinVersion:  "1.3",
				TLSProfile:     "",
			},
		}
	}

	tests := []struct {
		profile string
		wantErr bool
	}{
		{"modern", false},
		{"compat", false},
		{"", false},
		{"legacy", true},
	}

	for _, tt := range tests {
		t.Run(tt.profile, func(t *testing.T) {
			cfg := baseCfg()
			cfg.Security.TLSProfile = tt.profile
			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v for profile %q, wantErr %v", err, tt.profile, tt.wantErr)
			}
		})
	}
}

func TestSetupWizardKillSwitchDefaults(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		RPC: RPCConfig{
			Host: "localhost:18443",
			User: "user",
			Pass: "pass",
		},
		Client: ClientConfig{
			InterfaceName: "bcvpn1",
			TunIP:         "10.10.0.2",
			TunSubnet:     "24",
		},
	}

	applyConfigDefaults(cfg)

	if cfg.Client.EnableKillSwitch != false {
		t.Errorf("EnableKillSwitch default = %v, want false", cfg.Client.EnableKillSwitch)
	}
}

func TestSetupWizardNATDefaults(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		RPC: RPCConfig{
			Host: "localhost:18443",
			User: "user",
			Pass: "pass",
		},
		Provider: ProviderConfig{
			InterfaceName: "bcvpn0",
			ListenPort:    51820,
			EnableNAT:     true,
			TunIP:         "10.0.0.1",
			TunSubnet:     "24",
		},
	}

	applyConfigDefaults(cfg)

	if cfg.Provider.EnableNAT != true {
		t.Errorf("EnableNAT should remain true when set, got %v", cfg.Provider.EnableNAT)
	}
}

func TestResolveConfigPath(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmp)
	defer os.Setenv("XDG_CONFIG_HOME", oldConfigHome)

	path, err := ResolveConfigPath()
	if err != nil {
		t.Fatalf("ResolveConfigPath() error = %v", err)
	}

	if !strings.HasSuffix(path, "config.json") {
		t.Errorf("ResolveConfigPath() = %q, expected to end with config.json", path)
	}
	if filepath.Dir(filepath.Dir(path)) != tmp {
		t.Errorf("ResolveConfigPath() parent dir = %q, want %q", filepath.Dir(filepath.Dir(path)), tmp)
	}
}

func TestConfigFieldGetSet_RoundTrip(t *testing.T) {
	t.Parallel()

	cfg := &Config{}

	fields := []struct {
		key   string
		value string
		check func(*Config) bool
	}{
		{"rpc.host", "myhost:9999", func(c *Config) bool { return c.RPC.Host == "myhost:9999" }},
		{"provider.listen_port", "12345", func(c *Config) bool { return c.Provider.ListenPort == 12345 }},
		{"client.enable_kill_switch", "true", func(c *Config) bool { return c.Client.EnableKillSwitch }},
		{"provider.price_sats_per_session", "5000", func(c *Config) bool { return c.Provider.Price == 5000 }},
		{"logging.level", "debug", func(c *Config) bool { return c.Logging.Level == "debug" }},
		{"security.tls_min_version", "1.2", func(c *Config) bool { return c.Security.TLSMinVersion == "1.2" }},
	}

	for _, f := range fields {
		t.Run(f.key, func(t *testing.T) {
			err := SetConfigField(cfg, f.key, f.value)
			if err != nil {
				t.Fatalf("SetConfigField() error = %v", err)
			}
			if !f.check(cfg) {
				t.Errorf("SetConfigField(%q, %q) did not set expected value", f.key, f.value)
			}

			got, err := GetConfigField(cfg, f.key)
			if err != nil {
				t.Fatalf("GetConfigField() error = %v", err)
			}
			want, _ := GetConfigField(cfg, f.key)
			if got != want {
				t.Errorf("GetConfigField() round-trip mismatch for %q", f.key)
			}
		})
	}
}

func TestValidateProviderBandwidthLimit(t *testing.T) {
	baseCfg := func() *Config {
		return &Config{
			RPC: RPCConfig{
				Host: "localhost:18443",
				User: "user",
				Pass: "pass",
			},
			Provider: ProviderConfig{
				InterfaceName: "bcvpn0",
				ListenPort:    51820,
				TunIP:         "10.0.0.1",
				TunSubnet:     "24",
			},
			Client: ClientConfig{
				InterfaceName: "bcvpn1",
				TunIP:         "10.10.0.2",
				TunSubnet:     "24",
			},
		}
	}

	tests := []struct {
		limit   string
		wantErr bool
	}{
		{"0", false},
		{"10mbit", false},
		{"100mbit", false},
		{"1gbit", false},
		{"", false},
		{"-1", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.limit, func(t *testing.T) {
			cfg := baseCfg()
			cfg.Provider.BandwidthLimit = tt.limit
			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v for bandwidth %q, wantErr %v", err, tt.limit, tt.wantErr)
			}
		})
	}
}

func TestConfigLoadAndValidate(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		RPC: RPCConfig{
			Host: "localhost:18443",
			User: "user",
			Pass: "pass",
		},
		Provider: ProviderConfig{
			InterfaceName:            "bcvpn0",
			ListenPort:               51820,
			BandwidthLimit:           "100mbit",
			EnableNAT:                true,
			CertLifetimeHours:        720,
			CertRotateBeforeHours:    24,
			HealthCheckInterval:      "30s",
			BandwidthMonitorInterval: "30s",
			AnnouncementInterval:     "24h",
			HeartbeatInterval:        "5m",
			ShutdownTimeout:          "10s",
			TunIP:                    "10.0.0.1",
			TunSubnet:                "24",
		},
		Client: ClientConfig{
			InterfaceName: "bcvpn1",
			TunIP:         "10.10.0.2",
			TunSubnet:     "24",
		},
		Logging: LoggingConfig{
			Format: "json",
			Level:  "debug",
		},
		Security: SecurityConfig{
			KeyStorageMode: "file",
			TLSMinVersion:  "1.2",
			TLSProfile:     "compat",
		},
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(cfg); err != nil {
		t.Fatalf("Encode error = %v", err)
	}

	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.json")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if err := Validate(loaded); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateNATTraversalMethods(t *testing.T) {
	baseCfg := func() *Config {
		return &Config{
			RPC: RPCConfig{
				Host: "localhost:18443",
				User: "user",
				Pass: "pass",
			},
			Provider: ProviderConfig{
				InterfaceName: "bcvpn0",
				ListenPort:    51820,
				TunIP:         "10.0.0.1",
				TunSubnet:     "24",
			},
			Client: ClientConfig{
				InterfaceName: "bcvpn1",
				TunIP:         "10.10.0.2",
				TunSubnet:     "24",
			},
		}
	}

	tests := []struct {
		method  string
		wantErr bool
	}{
		{"auto", false},
		{"upnp", false},
		{"natpmp", false},
		{"none", false},
		{"", false},
		{"invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			cfg := baseCfg()
			cfg.Provider.NATTraversalMethod = tt.method
			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v for NAT method %q, wantErr %v", err, tt.method, tt.wantErr)
			}
		})
	}
}
