//go:build linux

package tunnel

import (
	"net"
	"strings"
)

func recoverPendingNetworkStateFromMarker(m *networkCleanupMarker) error {
	if m == nil {
		return nil
	}
	if m.IfaceName != "" && m.ProviderHost != "" {
		if net.ParseIP(m.ProviderHost) != nil {
			linuxRestoreRouting(m.IfaceName, m.ProviderHost)
		}
	}
	if m.DNSConfigured {
		if !strings.HasPrefix(m.DNSServers[0], "1.1.1.1") && !strings.HasPrefix(m.DNSServers[0], "8.8.8.8") {
			linuxRestoreDNS()
		}
	}
	return nil
}
