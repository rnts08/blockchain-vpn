//go:build linux

package tunnel

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func setupProviderEgressNAT(tunIfName, tunIP, tunSubnet, outboundIf string) (func(), error) {
	if tunIfName == "" {
		return nil, fmt.Errorf("provider TUN interface name is required")
	}

	if outboundIf == "" {
		var err error
		outboundIf, err = getDefaultOutboundInterface()
		if err != nil {
			return nil, err
		}
	}

	cidr, err := cidrFromIPPrefix(tunIP, tunSubnet)
	if err != nil {
		return nil, err
	}

	previousForwarding, _ := os.ReadFile("/proc/sys/net/ipv4/ip_forward")
	if _, err := runLinuxCommand("sysctl", "-w", "net.ipv4.ip_forward=1"); err != nil {
		return nil, fmt.Errorf("failed to enable ipv4 forwarding: %w", err)
	}

	rules := [][]string{
		{"-A", "FORWARD", "-i", tunIfName, "-o", outboundIf, "-j", "ACCEPT"},
		{"-A", "FORWARD", "-i", outboundIf, "-o", tunIfName, "-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "ACCEPT"},
		{"-t", "nat", "-A", "POSTROUTING", "-s", cidr, "-o", outboundIf, "-j", "MASQUERADE"},
	}
	for _, rule := range rules {
		if _, err := runLinuxCommand("iptables", rule...); err != nil {
			restoreProviderNATRules(tunIfName, outboundIf, cidr, previousForwarding)
			return nil, fmt.Errorf("failed to apply iptables rule %v: %w", rule, err)
		}
	}

	cleanup := func() {
		restoreProviderNATRules(tunIfName, outboundIf, cidr, previousForwarding)
	}
	return cleanup, nil
}

func restoreProviderNATRules(tunIfName, outboundIf, cidr string, previousForwarding []byte) {
	_, _ = runLinuxCommand("iptables", "-t", "nat", "-D", "POSTROUTING", "-s", cidr, "-o", outboundIf, "-j", "MASQUERADE")
	_, _ = runLinuxCommand("iptables", "-D", "FORWARD", "-i", outboundIf, "-o", tunIfName, "-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "ACCEPT")
	_, _ = runLinuxCommand("iptables", "-D", "FORWARD", "-i", tunIfName, "-o", outboundIf, "-j", "ACCEPT")
	if len(previousForwarding) > 0 {
		_, _ = runLinuxCommand("sysctl", "-w", fmt.Sprintf("net.ipv4.ip_forward=%s", strings.TrimSpace(string(previousForwarding))))
	}
}

func getDefaultOutboundInterface() (string, error) {
	out, err := runLinuxCommand("ip", "route", "show", "default")
	if err != nil {
		return "", fmt.Errorf("failed to read default route: %w", err)
	}

	// Expected: "default via 192.168.1.1 dev eth0 proto dhcp ..."
	parts := strings.Fields(out)
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "dev" {
			return parts[i+1], nil
		}
	}
	return "", fmt.Errorf("could not determine outbound interface from default route: %s", out)
}

func runLinuxCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		return "", fmt.Errorf("%s %v failed: %w (%s)", name, args, err, msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}
