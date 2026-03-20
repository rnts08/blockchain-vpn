//go:build linux

package tunnel

import (
	"log"
	"net"
	"strings"

	"github.com/vishvananda/netlink"
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
		if len(m.DNSServers) > 0 && !strings.HasPrefix(m.DNSServers[0], "1.1.1.1") && !strings.HasPrefix(m.DNSServers[0], "8.8.8.8") {
			linuxRestoreDNS()
		}
	}
	return nil
}

func cleanupStaleTunInterfaces(prefixes []string) error {
	for _, prefix := range prefixes {
		links, err := netlink.LinkList()
		if err != nil {
			continue
		}
		for _, link := range links {
			if strings.HasPrefix(link.Attrs().Name, prefix) {
				log.Printf("Cleaning up stale TUN interface: %s", link.Attrs().Name)
				if err := netlink.LinkDel(link); err != nil {
					log.Printf("Warning: failed to delete stale interface %s: %v", link.Attrs().Name, err)
				}
			}
		}
	}
	return nil
}
