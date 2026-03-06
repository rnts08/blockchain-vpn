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
	Price          uint64 `json:"price_sats_per_session"`
	PrivateKeyFile string `json:"private_key_file"`
	BandwidthLimit string `json:"bandwidth_limit"` // e.g. "10mbit"
}

type ClientConfig struct {
	InterfaceName string `json:"interface_name"`
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
			AnnounceIP:     "1.2.3.4",
			Price:          1000,
			PrivateKeyFile: "provider.key",
			BandwidthLimit: "10mbit",
		},
		Client: ClientConfig{
			InterfaceName: "wg0",
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