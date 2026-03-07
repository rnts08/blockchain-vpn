//go:build linux

package tunnel

import "fmt"

const linuxKillSwitchChain = "BCVPN-KILLSWITCH"

func setupKillSwitch(tunIfName, providerHost string) (func(), error) {
	if tunIfName == "" {
		return nil, fmt.Errorf("kill switch requires client TUN interface name")
	}
	providerIP, err := resolveProviderIPv4(providerHost)
	if err != nil {
		return nil, err
	}

	_, _ = runLinuxCommand("iptables", "-D", "OUTPUT", "-j", linuxKillSwitchChain)
	_, _ = runLinuxCommand("iptables", "-F", linuxKillSwitchChain)
	_, _ = runLinuxCommand("iptables", "-X", linuxKillSwitchChain)

	if _, err := runLinuxCommand("iptables", "-N", linuxKillSwitchChain); err != nil {
		return nil, fmt.Errorf("failed to create kill switch chain: %w", err)
	}
	rules := [][]string{
		{"-A", linuxKillSwitchChain, "-o", "lo", "-j", "ACCEPT"},
		{"-A", linuxKillSwitchChain, "-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "ACCEPT"},
		{"-A", linuxKillSwitchChain, "-d", providerIP + "/32", "-j", "ACCEPT"},
		{"-A", linuxKillSwitchChain, "-o", tunIfName, "-j", "ACCEPT"},
		{"-A", linuxKillSwitchChain, "-j", "DROP"},
	}
	for _, rule := range rules {
		if _, err := runLinuxCommand("iptables", rule...); err != nil {
			cleanupKillSwitchLinux()
			return nil, fmt.Errorf("failed to install kill switch rule %v: %w", rule, err)
		}
	}
	if _, err := runLinuxCommand("iptables", "-I", "OUTPUT", "1", "-j", linuxKillSwitchChain); err != nil {
		cleanupKillSwitchLinux()
		return nil, fmt.Errorf("failed to hook kill switch chain: %w", err)
	}

	return func() {
		cleanupKillSwitchLinux()
	}, nil
}

func cleanupKillSwitchLinux() {
	_, _ = runLinuxCommand("iptables", "-D", "OUTPUT", "-j", linuxKillSwitchChain)
	_, _ = runLinuxCommand("iptables", "-F", linuxKillSwitchChain)
	_, _ = runLinuxCommand("iptables", "-X", linuxKillSwitchChain)
}
