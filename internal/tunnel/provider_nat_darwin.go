//go:build darwin

package tunnel

import (
	"fmt"
	"strings"
)

const pfAnchorName = "com.apple/blockchainvpn"

func setupProviderEgressNAT(tunIfName, tunIP, tunSubnet, outboundIf string) (func(), error) {
	if strings.TrimSpace(tunIfName) == "" {
		return nil, fmt.Errorf("provider TUN interface name is required")
	}

	if strings.TrimSpace(outboundIf) == "" {
		defIface, err := getDefaultOutboundInterfaceDarwin()
		if err != nil {
			return nil, err
		}
		outboundIf = defIface
	}

	cidr, err := cidrFromIPPrefix(tunIP, tunSubnet)
	if err != nil {
		return nil, err
	}

	prevForwarding, _ := runCmd("sysctl", "-n", "net.inet.ip.forwarding")
	if _, err := runCmd("sysctl", "-w", "net.inet.ip.forwarding=1"); err != nil {
		return nil, fmt.Errorf("failed to enable ip forwarding: %w", err)
	}

	// Ensure PF is enabled. macOS generally keeps it enabled when needed.
	if _, err := runCmd("pfctl", "-E"); err != nil {
		restoreProviderNATDarwin(prevForwarding)
		return nil, fmt.Errorf("failed to enable pf: %w", err)
	}

	// Use an anchor under com.apple/*, which is referenced by the default pf config.
	rules := fmt.Sprintf("nat on %s inet from %s to any -> (%s)\n", outboundIf, cidr, outboundIf)
	if _, err := runCmd("sh", "-c", fmt.Sprintf("printf %%s %q | pfctl -a %s -f -", rules, pfAnchorName)); err != nil {
		restoreProviderNATDarwin(prevForwarding)
		return nil, fmt.Errorf("failed to load PF NAT rules: %w", err)
	}

	cleanup := func() {
		_, _ = runCmd("pfctl", "-a", pfAnchorName, "-F", "all")
		restoreProviderNATDarwin(prevForwarding)
	}
	return cleanup, nil
}

func restoreProviderNATDarwin(previousForwarding string) {
	prev := strings.TrimSpace(previousForwarding)
	if prev == "0" || prev == "1" {
		_, _ = runCmd("sysctl", "-w", "net.inet.ip.forwarding="+prev)
	}
}

func getDefaultOutboundInterfaceDarwin() (string, error) {
	_, iface, err := getDefaultRouteInfo()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(iface) == "" {
		return "", fmt.Errorf("could not determine default outbound interface")
	}
	return iface, nil
}
