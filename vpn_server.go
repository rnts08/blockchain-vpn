package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// StartVPNServer is a long-running function that manages the WireGuard interface.
// It periodically checks the AuthManager for authorized peers and updates the
// interface configuration accordingly.
func StartVPNServer(cfg *ProviderConfig, privateKey wgtypes.Key, authManager *AuthManager) error {
	// 1. Get a handle to wgctrl.
	wgClient, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("failed to open wgctrl: %w", err)
	}
	defer wgClient.Close()

	log.Printf("Provider public key: %s", privateKey.PublicKey().String())

	// 3. Main reconciliation loop.
	ticker := time.NewTicker(10 * time.Second) // Reconcile peers every 10 seconds.
	defer ticker.Stop()

	for range ticker.C {
		// Get the current device configuration to ensure it exists.
		_, err := wgClient.Device(cfg.InterfaceName)
		if err != nil {
			// Device might not exist yet, let's try to create it.
			log.Printf("WireGuard interface %s not found, attempting to create and configure.", cfg.InterfaceName)
			// This is OS-specific. For now, we assume it's created externally or
			// we just apply the config which might create it on some systems.
		}

		// Build the list of desired peers from the AuthManager.
		authorizedPeers := authManager.GetAuthorizedPeers()
		var peerConfigs []wgtypes.PeerConfig

		for key := range authorizedPeers {
			peerConfigs = append(peerConfigs, wgtypes.PeerConfig{
				PublicKey: key,
				// For a real service, you would assign a unique IP from a pool for each peer.
				// For this example, we allow all traffic from any authorized peer.
				AllowedIPs:   []net.IPNet{{IP: net.IPv4(0, 0, 0, 0), Mask: net.CIDRMask(0, 32)}},
				ReplaceAllowedIPs: true,
			})
		}

		// Apply the new configuration.
		config := wgtypes.Config{
			PrivateKey:   &privateKey,
			ListenPort:   &cfg.ListenPort,
			Peers:        peerConfigs,
			ReplacePeers: true, // This is crucial: it removes peers not in the list.
		}

		if err := wgClient.ConfigureDevice(cfg.InterfaceName, config); err != nil {
			log.Printf("Failed to configure WireGuard device: %v", err)
		}
	}

	return nil
}