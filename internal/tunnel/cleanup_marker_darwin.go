//go:build darwin

package tunnel

func recoverPendingNetworkStateFromMarker(m *networkCleanupMarker) error {
	if m == nil {
		return nil
	}
	if m.IfaceName != "" && m.ProviderHost != "" {
		restoreRouting(m.IfaceName, m.ProviderHost, m.DefaultGW)
	}
	if m.DNSConfigured {
		restoreDNSForService(m.DNSService, m.DNSServers, len(m.DNSServers) > 0)
	}
	return nil
}
