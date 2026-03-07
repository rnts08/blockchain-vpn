package config

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
	"unicode"
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
	if country := strings.TrimSpace(cfg.Provider.Country); country != "" && !isISOAlpha2(country) {
		errs = append(errs, fmt.Errorf("provider.country must be a valid 2-letter country code when set"))
	}
	if cfg.Provider.MaxConsumers < 0 {
		errs = append(errs, fmt.Errorf("provider.max_consumers must be >= 0"))
	}
	if cfg.Provider.MaxSessionDurationSecs < 0 {
		errs = append(errs, fmt.Errorf("provider.max_session_duration_secs must be >= 0"))
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
	switch strings.ToLower(strings.TrimSpace(cfg.Logging.Level)) {
	case "", "debug", "info", "warn", "error":
	default:
		errs = append(errs, fmt.Errorf("logging.level must be one of: debug, info, warn, error"))
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Security.KeyStorageMode)) {
	case "", "file", "auto", "keychain", "libsecret", "dpapi":
	default:
		errs = append(errs, fmt.Errorf("security.key_storage_mode must be one of: file, auto, keychain, libsecret, dpapi"))
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Security.TLSMinVersion)) {
	case "", "1.2", "1.3":
	default:
		errs = append(errs, fmt.Errorf("security.tls_min_version must be one of: 1.2, 1.3"))
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Security.TLSProfile)) {
	case "", "modern", "compat":
	default:
		errs = append(errs, fmt.Errorf("security.tls_profile must be one of: modern, compat"))
	}
	if tok := strings.TrimSpace(cfg.Security.MetricsAuthToken); tok != "" && len(tok) < 12 {
		errs = append(errs, fmt.Errorf("security.metrics_auth_token must be at least 12 characters when set"))
	}

	// Cross-field: detect subnet overlap between provider and client TUN.
	if overlapErr := checkSubnetOverlap(
		cfg.Provider.TunIP, cfg.Provider.TunSubnet,
		cfg.Client.TunIP, cfg.Client.TunSubnet,
		"provider", "client",
	); overlapErr != nil {
		errs = append(errs, overlapErr)
	}

	// Cross-field: detect port collision between provider listen port and metrics endpoints.
	providerPort := cfg.Provider.ListenPort
	if p := parsePort(cfg.Provider.MetricsListenAddr); p > 0 && p == providerPort {
		errs = append(errs, fmt.Errorf("provider.listen_port and provider.metrics_listen_addr share the same port (%d)", p))
	}
	if p := parsePort(cfg.Client.MetricsListenAddr); p > 0 && p == providerPort {
		errs = append(errs, fmt.Errorf("provider.listen_port and client.metrics_listen_addr share the same port (%d)", p))
	}

	return errors.Join(errs...)
}

// checkSubnetOverlap returns an error if the two IP+prefix strings describe overlapping
// but distinct networks. Provider and client intentionally share the same TUN subnet
// (provider is gateway, client gets a host IP), so we skip when they resolve to the
// same network prefix.
func checkSubnetOverlap(ip1, prefix1, ip2, prefix2, label1, label2 string) error {
	ip1 = strings.TrimSpace(ip1)
	ip2 = strings.TrimSpace(ip2)
	if ip1 == "" || ip2 == "" {
		return nil
	}
	_, net1, err1 := net.ParseCIDR(ip1 + "/" + strings.TrimSpace(prefix1))
	_, net2, err2 := net.ParseCIDR(ip2 + "/" + strings.TrimSpace(prefix2))
	if err1 != nil || err2 != nil {
		return nil // already caught by individual field validators
	}
	// If both IPs resolve to the same network prefix it is the intentional shared
	// TUN subnet (provider = gateway, clients = hosts). Not an error.
	if net1.String() == net2.String() {
		return nil
	}
	// Different network prefixes that still contain each other's IPs → real overlap.
	if net1.Contains(net.ParseIP(ip2)) || net2.Contains(net.ParseIP(ip1)) {
		return fmt.Errorf("%s and %s TUN subnets overlap (%s/%s and %s/%s)",
			label1, label2, ip1, prefix1, ip2, prefix2)
	}
	return nil
}

// parsePort extracts the numeric port from a host:port addr string.
func parsePort(addr string) int {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return 0
	}
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return 0
	}
	p, err := strconv.Atoi(portStr)
	if err != nil {
		return 0
	}
	return p
}

func isValidPrefix(v string) bool {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	return err == nil && n >= 0 && n <= 32
}

func isISOAlpha2(v string) bool {
	v = strings.TrimSpace(v)
	if len(v) != 2 {
		return false
	}
	for _, r := range v {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}
