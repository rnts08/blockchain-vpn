//go:build linux

package main

import (
	"fmt"

	"github.com/vishvananda/netlink"
)

// getDefaultGateway uses netlink to find the default gateway.
// This is a Go-native, Linux-specific implementation.
func getDefaultGateway() (string, error) {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return "", fmt.Errorf("failed to list routes: %w", err)
	}

	for _, route := range routes {
		if route.Dst == nil { // A nil destination signifies the default route
			return route.Gw.String(), nil
		}
	}

	return "", fmt.Errorf("default route not found")
}