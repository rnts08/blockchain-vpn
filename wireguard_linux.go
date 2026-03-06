//go:build linux

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// SetupTunnel creates the WireGuard interface and brings it up.
func SetupTunnel(ifaceName string) error {
	// ip link add dev <ifaceName> type wireguard
	cmd := exec.Command("ip", "link", "add", "dev", ifaceName, "type", "wireguard")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// If the interface already exists, we can proceed.
		if !strings.Contains(string(out), "File exists") {
			return fmt.Errorf("failed to create interface %q: %v, output: %s", ifaceName, err, out)
		}
	}

	// ip link set up dev <ifaceName>
	cmd = exec.Command("ip", "link", "set", "up", "dev", ifaceName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring up interface %q: %v, output: %s", ifaceName, err, out)
	}
	return nil
}

// TeardownTunnel removes the WireGuard interface.
func TeardownTunnel(ifaceName string) error {
	cmd := exec.Command("ip", "link", "delete", "dev", ifaceName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete interface %q: %v, output: %s", ifaceName, err, out)
	}
	return nil
}

// SetBandwidthLimit applies a traffic control rule to limit egress bandwidth.
func SetBandwidthLimit(ifaceName, limit string) error {
	// Clear existing qdiscs (ignore error if none exist)
	exec.Command("tc", "qdisc", "del", "dev", ifaceName, "root").Run()

	if limit == "" {
		return nil
	}

	// Add TBF qdisc to limit the interface output rate
	cmd := exec.Command("tc", "qdisc", "add", "dev", ifaceName, "root", "tbf", "rate", limit, "burst", "32kbit", "latency", "400ms")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set bandwidth limit: %v, output: %s", err, out)
	}
	return nil
}