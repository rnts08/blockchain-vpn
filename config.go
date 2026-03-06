package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	RPC      RPCConfig      `json:"rpc"`
	Provider ProviderConfig `json:"provider"`
	Client   ClientConfig   `json:"client"`
}

type RPCConfig struct {
	Host string `json:"host"`
	User string `json:"user"`
	Pass string `json:"pass"`
}

type ProviderConfig struct {
	InterfaceName  string `json:"interface_name"`
	ListenPort     int    `json:"listen_port"`
	AnnounceIP     string `json:"announce_ip"`
	Country        string `json:"country"` // 2-letter country code (e.g., "US"). Leave empty to auto-detect.
	Price          uint64 `json:"price_sats_per_session"`
	PrivateKeyFile string `json:"private_key_file"`
	BandwidthLimit string `json:"bandwidth_limit"` // e.g. "10mbit"
	BandwidthMonitorInterval string `json:"bandwidth_monitor_interval"`
	TunIP                    string `json:"tun_ip"`
	TunSubnet                string `json:"tun_subnet"`
}

type ClientConfig struct {
	InterfaceName string `json:"interface_name"`
	TunIP         string `json:"tun_ip"`
	TunSubnet     string `json:"tun_subnet"`
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
	cfg := Config{
		RPC: RPCConfig{
			Host: "localhost:18443",
			User: "yourrpcuser",
			Pass: "yourrpcpassword",
		},
		Provider: ProviderConfig{
			InterfaceName:  "wg0",
			ListenPort:     51820,
			AnnounceIP:     "", // Leave empty to auto-detect public IP
			Country:        "", // Leave empty to auto-detect country
			Price:          1000,
			PrivateKeyFile: "provider.key",
			BandwidthLimit: "10mbit",
			BandwidthMonitorInterval: "30s",
			TunIP:          "10.10.0.1",
			TunSubnet:      "24",
		},
		Client: ClientConfig{
			InterfaceName: "wg0",
			TunIP:         "10.10.0.2",
			TunSubnet:     "24",
		},
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