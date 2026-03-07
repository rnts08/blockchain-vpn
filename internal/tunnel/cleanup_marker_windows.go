//go:build windows

package tunnel

import (
	"fmt"
	"strings"
)

func recoverPendingNetworkStateFromMarker(m *networkCleanupMarker) error {
	if m == nil {
		return nil
	}
	tunIdx := m.TunIfIndex
	if tunIdx == 0 && m.IfaceName != "" {
		if idx, err := getWindowsInterfaceIndex(m.IfaceName); err == nil {
			tunIdx = idx
		}
	}
	if tunIdx != 0 && m.ProviderHost != "" {
		restoreWindowsRouting(tunIdx, m.ProviderHost)
	}
	if m.DNSConfigured && m.DefaultIfAlias != "" {
		if len(m.DNSServers) == 0 {
			_, _ = windowsRunPowerShell(fmt.Sprintf(`Set-DnsClientServerAddress -InterfaceAlias '%s' -ResetServerAddresses`, psEscape(m.DefaultIfAlias)))
		} else {
			var quoted []string
			for _, s := range m.DNSServers {
				quoted = append(quoted, fmt.Sprintf("'%s'", psEscape(s)))
			}
			_, _ = windowsRunPowerShell(fmt.Sprintf(`Set-DnsClientServerAddress -InterfaceAlias '%s' -ServerAddresses @(%s)`, psEscape(m.DefaultIfAlias), strings.Join(quoted, ",")))
		}
	}
	return nil
}
