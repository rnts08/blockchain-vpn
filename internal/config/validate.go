package config

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	var errs []error

	if strings.TrimSpace(cfg.RPC.Host) == "" {
		errs = append(errs, fmt.Errorf("rpc.host is required"))
	}

	if strings.TrimSpace(cfg.Provider.InterfaceName) == "" {
		errs = append(errs, fmt.Errorf("provider.interface_name is required"))
	}
	if cfg.Provider.ListenPort <= 0 || cfg.Provider.ListenPort > 65535 {
		errs = append(errs, fmt.Errorf("provider.listen_port must be between 1 and 65535"))
	}
	if strings.TrimSpace(cfg.Provider.AnnounceIP) != "" && net.ParseIP(strings.TrimSpace(cfg.Provider.AnnounceIP)) == nil {
		errs = append(errs, fmt.Errorf("provider.announce_ip must be a valid IP address when set"))
	}
	if strings.TrimSpace(cfg.Provider.TunIP) == "" || net.ParseIP(strings.TrimSpace(cfg.Provider.TunIP)) == nil {
		errs = append(errs, fmt.Errorf("provider.tun_ip must be a valid IP address"))
	}
	if !isValidPrefix(cfg.Provider.TunSubnet) {
		errs = append(errs, fmt.Errorf("provider.tun_subnet must be a valid IPv4 prefix length"))
	}
	if strings.TrimSpace(cfg.Provider.HealthCheckInterval) != "" {
		if _, err := time.ParseDuration(strings.TrimSpace(cfg.Provider.HealthCheckInterval)); err != nil {
			errs = append(errs, fmt.Errorf("provider.health_check_interval is invalid: %w", err))
		}
	}
	if strings.TrimSpace(cfg.Provider.MetricsListenAddr) != "" {
		if _, err := net.ResolveTCPAddr("tcp", strings.TrimSpace(cfg.Provider.MetricsListenAddr)); err != nil {
			errs = append(errs, fmt.Errorf("provider.metrics_listen_addr is invalid: %w", err))
		}
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Provider.IsolationMode)) {
	case "", "none", "sandbox":
	default:
		errs = append(errs, fmt.Errorf("provider.isolation_mode must be one of: none, sandbox"))
	}

	if strings.TrimSpace(cfg.Client.InterfaceName) == "" {
		errs = append(errs, fmt.Errorf("client.interface_name is required"))
	}
	if strings.TrimSpace(cfg.Client.TunIP) == "" || net.ParseIP(strings.TrimSpace(cfg.Client.TunIP)) == nil {
		errs = append(errs, fmt.Errorf("client.tun_ip must be a valid IP address"))
	}
	if !isValidPrefix(cfg.Client.TunSubnet) {
		errs = append(errs, fmt.Errorf("client.tun_subnet must be a valid IPv4 prefix length"))
	}
	if strings.TrimSpace(cfg.Client.MetricsListenAddr) != "" {
		if _, err := net.ResolveTCPAddr("tcp", strings.TrimSpace(cfg.Client.MetricsListenAddr)); err != nil {
			errs = append(errs, fmt.Errorf("client.metrics_listen_addr is invalid: %w", err))
		}
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Logging.Format)) {
	case "", "text", "json":
	default:
		errs = append(errs, fmt.Errorf("logging.format must be one of: text, json"))
	}

	return errors.Join(errs...)
}

func isValidPrefix(v string) bool {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	return err == nil && n >= 0 && n <= 32
}
