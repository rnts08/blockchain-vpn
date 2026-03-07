package config

import "testing"

func TestValidateDefaultLikeConfig(t *testing.T) {
	cfg := &Config{
		RPC: RPCConfig{
			Host: "localhost:18443",
			User: "user",
			Pass: "pass",
		},
		Provider: ProviderConfig{
			InterfaceName:         "bcvpn0",
			ListenPort:            51820,
			AnnounceIP:            "",
			Price:                 1000,
			HealthCheckInterval:   "30s",
			IsolationMode:         "none",
			TunIP:                 "10.10.0.1",
			TunSubnet:             "24",
			CertLifetimeHours:     720,
			CertRotateBeforeHours: 24,
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
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}
