//go:build linux

package tunnel

func recoverPendingNetworkStateFromMarker(m *networkCleanupMarker) error {
	if m == nil {
		return nil
	}
	if m.IfaceName != "" && m.ProviderHost != "" {
		restoreRouting(m.IfaceName, m.ProviderHost)
	}
	if m.DNSConfigured {
		restoreDNS()
	}
	return nil
}
