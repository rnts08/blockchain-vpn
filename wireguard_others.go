//go:build !linux

package main

import "fmt"

// SetupTunnel is a stub for non-Linux platforms.
func SetupTunnel(ifaceName string) error {
	return fmt.Errorf("automatic interface creation is not supported on this platform. Please create the WireGuard interface %q manually or use wg-quick", ifaceName)
}

// TeardownTunnel is a stub for non-Linux platforms.
func TeardownTunnel(ifaceName string) error {
	return fmt.Errorf("automatic interface deletion is not supported on this platform. Please delete the WireGuard interface %q manually", ifaceName)
}

func SetBandwidthLimit(ifaceName, limit string) error {
	if limit != "" {
		return fmt.Errorf("bandwidth limiting is not supported on this platform")
	}
	return nil
}
