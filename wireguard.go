package main

import (
	"fmt"
	"net"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// ConfigureTunnel sets up a WireGuard interface with a single peer.
// This function typically requires root/administrator privileges to run.
func ConfigureTunnel(ifaceName string, localKey wgtypes.Key, peerKey wgtypes.Key, peerEndpoint string, allowedIPs []net.IPNet) error {
	// Get a handle to the wgctrl client.
	wgClient, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("failed to open wgctrl: %w", err)
	}
	defer wgClient.Close()

	// Parse the peer's endpoint address.
	endpointAddr, err := net.ResolveUDPAddr("udp", peerEndpoint)
	if err != nil {
		return fmt.Errorf("failed to resolve peer endpoint address: %w", err)
	}

	// The configuration to apply to the interface.
	wgConfig := wgtypes.Config{
		PrivateKey: &localKey,
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey:  peerKey,
				Endpoint:   endpointAddr,
				AllowedIPs: allowedIPs,
			},
		},
	}

	// Configure the device. On Linux, this may require the interface to be created first,
	// e.g., `sudo ip link add dev <ifaceName> type wireguard`.
	return wgClient.ConfigureDevice(ifaceName, wgConfig)
}

// GetTunnelStatus returns the device information for the specified interface.
func GetTunnelStatus(ifaceName string) (*wgtypes.Device, error) {
	wgClient, err := wgctrl.New()
	if err != nil {
		return nil, fmt.Errorf("failed to open wgctrl: %w", err)
	}
	defer wgClient.Close()

	return wgClient.Device(ifaceName)
}