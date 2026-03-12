package config

import (
	"testing"
)

func TestConfigRegistry(t *testing.T) {
	t.Parallel()

	// Test that registry is populated
	fields := ListConfigFields()
	if len(fields) == 0 {
		t.Error("expected fields in registry, got none")
	}

	// Check for expected fields
	expectedFields := []string{
		"rpc.host",
		"provider.listen_port",
		"client.enable_kill_switch",
		"logging.format",
		"security.tls_min_version",
	}

	for _, f := range expectedFields {
		found := false
		for _, field := range fields {
			if field == f {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected field %q in registry", f)
		}
	}
}

func TestGetConfigField(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		RPC: RPCConfig{
			Host: "localhost:8332",
		},
		Provider: ProviderConfig{
			ListenPort: 51820,
		},
		Client: ClientConfig{
			EnableKillSwitch: true,
		},
	}

	tests := []struct {
		key     string
		want    any
		wantErr bool
	}{
		{"rpc.host", "localhost:8332", false},
		{"provider.listen_port", 51820, false},
		{"client.enable_kill_switch", true, false},
		{"unknown.key", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, err := GetConfigField(cfg, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfigField() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("GetConfigField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetConfigField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key     string
		value   string
		wantErr bool
		check   func(*Config) bool
	}{
		{
			key:   "rpc.host",
			value: "localhost:9999",
			check: func(c *Config) bool { return c.RPC.Host == "localhost:9999" },
		},
		{
			key:   "provider.listen_port",
			value: "12345",
			check: func(c *Config) bool { return c.Provider.ListenPort == 12345 },
		},
		{
			key:   "client.enable_kill_switch",
			value: "false",
			check: func(c *Config) bool { return c.Client.EnableKillSwitch == false },
		},
		{
			key:     "unknown.key",
			value:   "value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			cfg := &Config{}
			err := SetConfigField(cfg, tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetConfigField() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !tt.check(cfg) {
				t.Errorf("SetConfigField() did not set value correctly for %s", tt.key)
			}
		})
	}
}

func TestSetConfigField_StringSlice(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	err := SetConfigField(cfg, "provider.dns_servers", "1.1.1.1, 8.8.8.8")
	if err != nil {
		t.Fatalf("SetConfigField() error = %v", err)
	}

	if len(cfg.Provider.DNSServers) != 2 {
		t.Errorf("expected 2 DNS servers, got %d", len(cfg.Provider.DNSServers))
	}
	if cfg.Provider.DNSServers[0] != "1.1.1.1" {
		t.Errorf("expected first DNS server to be 1.1.1.1, got %s", cfg.Provider.DNSServers[0])
	}
}
