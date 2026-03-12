package config

import (
	"testing"
)

func TestValidateDefaultLikeConfig(t *testing.T) {
	cfg := &Config{
		RPC: RPCConfig{
			Host: "localhost:18443",
			User: "user",
			Pass: "pass",
		},
		Provider: ProviderConfig{
			InterfaceName:            "bcvpn0",
			ListenPort:               51820,
			AnnounceIP:               "",
			Price:                    1000,
			HealthCheckInterval:      "30s",
			BandwidthMonitorInterval: "30s",
			AnnouncementInterval:     "24h",
			IsolationMode:            "none",
			TunIP:                    "10.10.0.1",
			TunSubnet:                "24",
			CertLifetimeHours:        720,
			CertRotateBeforeHours:    24,
			MaxSessionDurationSecs:   3600,
		},
		Client: ClientConfig{
			InterfaceName: "bcvpn1",
			TunIP:         "10.10.0.2",
			TunSubnet:     "24",
		},
	}

	if err := Validate(cfg); err != nil {
		t.Fatalf("expected config to validate, got: %v", err)
	}
}

func TestValidateInvalidConfig(t *testing.T) {
	cfg := &Config{
		RPC: RPCConfig{
			Host: "",
		},
		Provider: ProviderConfig{
			InterfaceName:       "",
			ListenPort:          70000,
			AnnounceIP:          "not-an-ip",
			Country:             "USA",
			MaxConsumers:        -1,
			HealthCheckInterval: "not-a-duration",
			IsolationMode:       "invalid-mode",
			TunIP:               "bad-ip",
			TunSubnet:           "99",
		},
		Client: ClientConfig{
			InterfaceName: "",
			TunIP:         "bad-ip",
			TunSubnet:     "33",
		},
		Security: SecurityConfig{
			KeyStorageMode:   "bad-mode",
			TLSMinVersion:    "1.0",
			TLSProfile:       "legacy",
			MetricsAuthToken: "short",
		},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestValidateDurationBounds(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		wantErr     bool
		errContains string
	}{
		{
			name: "health_check_interval too short",
			cfg: &Config{
				RPC: RPCConfig{Host: "localhost:18443"},
				Provider: ProviderConfig{
					InterfaceName:       "bcvpn0",
					ListenPort:          51820,
					HealthCheckInterval: "500ms",
					TunIP:               "10.10.0.1",
					TunSubnet:           "24",
				},
				Client: ClientConfig{
					InterfaceName: "bcvpn1",
					TunIP:         "10.10.0.2",
					TunSubnet:     "24",
				},
			},
			wantErr:     true,
			errContains: "health_check_interval must be at least 1s",
		},
		{
			name: "bandwidth_monitor_interval too long",
			cfg: &Config{
				RPC: RPCConfig{Host: "localhost:18443"},
				Provider: ProviderConfig{
					InterfaceName:            "bcvpn0",
					ListenPort:               51820,
					BandwidthMonitorInterval: "25h",
					TunIP:                    "10.10.0.1",
					TunSubnet:                "24",
				},
				Client: ClientConfig{
					InterfaceName: "bcvpn1",
					TunIP:         "10.10.0.2",
					TunSubnet:     "24",
				},
			},
			wantErr:     true,
			errContains: "bandwidth_monitor_interval must be at most 24h",
		},
		{
			name: "announcement_interval too short",
			cfg: &Config{
				RPC: RPCConfig{Host: "localhost:18443"},
				Provider: ProviderConfig{
					InterfaceName:        "bcvpn0",
					ListenPort:           51820,
					AnnouncementInterval: "30m",
					TunIP:                "10.10.0.1",
					TunSubnet:            "24",
				},
				Client: ClientConfig{
					InterfaceName: "bcvpn1",
					TunIP:         "10.10.0.2",
					TunSubnet:     "24",
				},
			},
			wantErr:     true,
			errContains: "announcement_interval must be at least 1h",
		},
		{
			name: "valid duration bounds",
			cfg: &Config{
				RPC: RPCConfig{Host: "localhost:18443"},
				Provider: ProviderConfig{
					InterfaceName:            "bcvpn0",
					ListenPort:               51820,
					HealthCheckInterval:      "30s",
					BandwidthMonitorInterval: "1m",
					AnnouncementInterval:     "12h",
					TunIP:                    "10.10.0.1",
					TunSubnet:                "24",
				},
				Client: ClientConfig{
					InterfaceName: "bcvpn1",
					TunIP:         "10.10.0.2",
					TunSubnet:     "24",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" && !contains(err.Error(), tt.errContains) {
				t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
			}
		})
	}
}

func TestValidateCrossField(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		wantErr     bool
		errContains string
	}{
		{
			name: "cert_rotate_before >= cert_lifetime",
			cfg: &Config{
				RPC: RPCConfig{Host: "localhost:18443"},
				Provider: ProviderConfig{
					InterfaceName:         "bcvpn0",
					ListenPort:            51820,
					CertLifetimeHours:     24,
					CertRotateBeforeHours: 48,
					TunIP:                 "10.10.0.1",
					TunSubnet:             "24",
				},
				Client: ClientConfig{
					InterfaceName: "bcvpn1",
					TunIP:         "10.10.0.2",
					TunSubnet:     "24",
				},
			},
			wantErr:     true,
			errContains: "cert_rotate_before_hours",
		},
		{
			name: "max_session >= cert_lifetime",
			cfg: &Config{
				RPC: RPCConfig{Host: "localhost:18443"},
				Provider: ProviderConfig{
					InterfaceName:          "bcvpn0",
					ListenPort:             51820,
					CertLifetimeHours:      24,
					MaxSessionDurationSecs: 90000,
					TunIP:                  "10.10.0.1",
					TunSubnet:              "24",
				},
				Client: ClientConfig{
					InterfaceName: "bcvpn1",
					TunIP:         "10.10.0.2",
					TunSubnet:     "24",
				},
			},
			wantErr:     true,
			errContains: "max_session_duration_secs",
		},
		{
			name: "valid cross-field",
			cfg: &Config{
				RPC: RPCConfig{Host: "localhost:18443"},
				Provider: ProviderConfig{
					InterfaceName:          "bcvpn0",
					ListenPort:             51820,
					CertLifetimeHours:      720,
					CertRotateBeforeHours:  24,
					MaxSessionDurationSecs: 3600,
					TunIP:                  "10.10.0.1",
					TunSubnet:              "24",
				},
				Client: ClientConfig{
					InterfaceName: "bcvpn1",
					TunIP:         "10.10.0.2",
					TunSubnet:     "24",
				},
			},
			wantErr: false,
		},
		{
			name: "zero values are valid (no limit)",
			cfg: &Config{
				RPC: RPCConfig{Host: "localhost:18443"},
				Provider: ProviderConfig{
					InterfaceName:          "bcvpn0",
					ListenPort:             51820,
					CertLifetimeHours:      0,
					CertRotateBeforeHours:  0,
					MaxSessionDurationSecs: 0,
					TunIP:                  "10.10.0.1",
					TunSubnet:              "24",
				},
				Client: ClientConfig{
					InterfaceName: "bcvpn1",
					TunIP:         "10.10.0.2",
					TunSubnet:     "24",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" && !contains(err.Error(), tt.errContains) {
				t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
			}
		})
	}
}

func TestGenerateRandomRPCPassword(t *testing.T) {
	t.Parallel()

	pw1, err := GenerateRandomRPCPassword()
	if err != nil {
		t.Fatalf("GenerateRandomRPCPassword() error = %v", err)
	}
	if len(pw1) != 64 {
		t.Errorf("expected 64 hex chars, got %d", len(pw1))
	}

	pw2, err := GenerateRandomRPCPassword()
	if err != nil {
		t.Fatalf("GenerateRandomRPCPassword() error = %v", err)
	}
	if pw1 == pw2 {
		t.Error("expected different passwords")
	}
}

func TestApplyConfigDefaults(t *testing.T) {
	cfg := &Config{
		Provider: ProviderConfig{
			TunIP:     "10.10.0.1",
			TunSubnet: "24",
		},
		Client: ClientConfig{
			TunIP:     "10.10.0.2",
			TunSubnet: "24",
		},
	}

	applyConfigDefaults(cfg)

	if cfg.Client.MaxParallelTunnels != 1 {
		t.Errorf("expected MaxParallelTunnels=1, got %d", cfg.Client.MaxParallelTunnels)
	}
	if cfg.Provider.HealthCheckInterval != "30s" {
		t.Errorf("expected HealthCheckInterval=30s, got %s", cfg.Provider.HealthCheckInterval)
	}
	if cfg.Provider.BandwidthMonitorInterval != "30s" {
		t.Errorf("expected BandwidthMonitorInterval=30s, got %s", cfg.Provider.BandwidthMonitorInterval)
	}
	if cfg.Provider.AnnouncementInterval != "24h" {
		t.Errorf("expected AnnouncementInterval=24h, got %s", cfg.Provider.AnnouncementInterval)
	}
	if len(cfg.Provider.DNSServers) != 2 {
		t.Errorf("expected DNSServers len=2, got %d", len(cfg.Provider.DNSServers))
	}
	if len(cfg.Client.DNSServers) != 2 {
		t.Errorf("expected Client DNSServers len=2, got %d", len(cfg.Client.DNSServers))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
