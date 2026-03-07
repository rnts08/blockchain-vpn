//go:build windows

package tunnel

import (
	"fmt"
	"strconv"
	"strings"
)

func setupKillSwitch(tunIfName, providerHost string) (func(), error) {
	if _, err := resolveProviderIPv4(providerHost); err != nil {
		return nil, err
	}

	defaultRoute, err := getWindowsDefaultRoute()
	if err != nil {
		return nil, err
	}
	defaultIf := strconv.Itoa(defaultRoute.InterfaceIndex)

	// Fallback blackhole routes on the default interface with high metric.
	// If tunnel routes disappear unexpectedly, these keep traffic blocked.
	routes := [][]string{
		{"ADD", "0.0.0.0", "MASK", "128.0.0.0", "127.0.0.1", "METRIC", "5000", "IF", defaultIf},
		{"ADD", "128.0.0.0", "MASK", "128.0.0.0", "127.0.0.1", "METRIC", "5000", "IF", defaultIf},
	}
	for _, r := range routes {
		if _, err := runWindowsCmd("route", r...); err != nil && !strings.Contains(strings.ToLower(err.Error()), "object already exists") {
			cleanupKillSwitchWindows(defaultIf)
			return nil, fmt.Errorf("failed to install windows kill switch route %v: %w", r, err)
		}
	}

	return func() {
		cleanupKillSwitchWindows(defaultIf)
	}, nil
}

func cleanupKillSwitchWindows(defaultIf string) {
	_, _ = runWindowsCmd("route", "DELETE", "0.0.0.0", "MASK", "128.0.0.0", "127.0.0.1", "IF", defaultIf)
	_, _ = runWindowsCmd("route", "DELETE", "128.0.0.0", "MASK", "128.0.0.0", "127.0.0.1", "IF", defaultIf)
}
