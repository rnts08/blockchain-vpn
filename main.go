package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"flag"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const defaultConfigPath = "config.json"

func main() {
	// Handle generate-config before loading config
	if len(os.Args) >= 2 && os.Args[1] == "generate-config" {
		if err := GenerateDefaultConfig(defaultConfigPath); err != nil {
			log.Fatalf("Failed to generate config: %v", err)
		}
		log.Printf("Generated default config at %s", defaultConfigPath)
		return
	}

	// Load configuration
	cfg, err := LoadConfig(defaultConfigPath)
	if err != nil {
		log.Fatalf("Failed to load config file at %s: %v. Please create one.", defaultConfigPath, err)
	}

	// Subcommands
	scanCmd := flag.NewFlagSet("scan", flag.ExitOnError)
	connectCmd := flag.NewFlagSet("connect", flag.ExitOnError)
	disconnectCmd := flag.NewFlagSet("disconnect", flag.ExitOnError)
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	monitorCmd := flag.NewFlagSet("monitor", flag.ExitOnError)
	startProviderCmd := flag.NewFlagSet("start-provider", flag.ExitOnError)
	historyCmd := flag.NewFlagSet("history", flag.ExitOnError)

	// Scan specific flags
	scanStartBlock := scanCmd.Int64("startblock", 0, "Block height to start scanning from (0 for full scan)")
	scanSortBy := scanCmd.String("sort", "latency", "Sort providers by 'price', 'country', or 'latency'")
	scanCountry := scanCmd.String("country", "", "Filter providers by country code (e.g., US, DE)")

	// Connect specific flags
	connectPeerKey := connectCmd.String("peerkey", "", "The public key of the VPN endpoint")
	connectEndpoint := connectCmd.String("endpoint", "", "The endpoint address of the VPN (ip:port)")
	connectMaxLatency := connectCmd.Duration("max-latency", 0, "Automatically disconnect if latency exceeds this duration (e.g. 500ms)")

	// Disconnect specific flags
	disconnectIface := disconnectCmd.String("iface", cfg.Client.InterfaceName, "The name of the local tunnel interface")

	// Status specific flags
	statusIface := statusCmd.String("iface", cfg.Client.InterfaceName, "The name of the local tunnel interface")

	// Monitor specific flags
	monitorIface := monitorCmd.String("iface", cfg.Client.InterfaceName, "The name of the local tunnel interface to monitor")

	if len(os.Args) < 2 {
		fmt.Println("expected 'generate-config', 'start-provider', 'scan', 'connect', 'disconnect', 'status', 'monitor', or 'history' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "start-provider":
		startProviderCmd.Parse(os.Args[2:])
		client := connectRPC(cfg.RPC.Host, cfg.RPC.User, cfg.RPC.Pass)
		defer client.Shutdown()

		// Setup the WireGuard interface
		if err := SetupTunnel(cfg.Provider.InterfaceName); err != nil {
			log.Fatalf("Failed to setup tunnel interface: %v", err)
		}

		// Apply bandwidth limit if configured
		if err := SetBandwidthLimit(cfg.Provider.InterfaceName, cfg.Provider.BandwidthLimit); err != nil {
			log.Printf("Warning: Failed to set bandwidth limit: %v", err)
		}

		// Create the authorization manager
		authManager := NewAuthManager()

		// Load or generate provider's private key
		providerKey, err := loadOrGenerateKey(cfg.Provider.PrivateKeyFile)
		if err != nil {
			log.Fatalf("Could not load or generate provider key: %v", err)
		}

		var announceIP net.IP
		if cfg.Provider.AnnounceIP != "" {
			announceIP = net.ParseIP(cfg.Provider.AnnounceIP)
			if announceIP == nil {
				log.Fatalf("Invalid AnnounceIP in config: %s", cfg.Provider.AnnounceIP)
			}
		} else {
			log.Println("AnnounceIP not set, attempting to detect public IP...")
			var err error
			announceIP, err = GetPublicIP()
			if err != nil {
				log.Fatalf("Failed to detect public IP: %v. Please set announce_ip in config.json", err)
			}
			log.Printf("Detected public IP: %s", announceIP.String())
		}

		endpoint := &VPNEndpoint{
			IP:        announceIP,
			Port:      uint16(cfg.Provider.ListenPort),
			Price:     cfg.Provider.Price,
			PublicKey: providerKey.PublicKey().Bytes(),
		}

		// Start re-announcement loop in a goroutine
		go func() {
			// Announce immediately on start
			if err := AnnounceService(client, endpoint); err != nil {
				log.Printf("Initial service announcement failed: %v", err)
			}
			// Then announce on a schedule
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()
			for range ticker.C {
				if err := AnnounceService(client, endpoint); err != nil {
					log.Printf("Scheduled re-announcement failed: %v", err)
				}
			}
		}()

		// Start payment monitor in a goroutine
		go MonitorPayments(client, authManager, cfg.Provider.Price)

		// Start the WireGuard server to manage authorized peers
		go func() {
			log.Println("Starting WireGuard peer manager...")
			if err := StartVPNServer(&cfg.Provider, providerKey, authManager); err != nil {
				log.Fatalf("VPN Server failed: %v", err)
			}
		}()

		// Then start the echo server, which blocks forever.
		// In a real app, this would run alongside the WireGuard listener.
		StartEchoServer(cfg.Provider.ListenPort)

	case "scan":
		scanCmd.Parse(os.Args[2:])
		client := connectRPC(cfg.RPC.Host, cfg.RPC.User, cfg.RPC.Pass)
		defer client.Shutdown()

		// Detect Chain
		chainInfo, err := client.GetBlockChainInfo()
		if err != nil {
			log.Fatalf("Failed to get blockchain info: %v", err)
		}

		var chainParams *chaincfg.Params
		switch chainInfo.Chain {
		case "main":
			chainParams = &chaincfg.MainNetParams
		case "test":
			chainParams = &chaincfg.TestNet3Params
		case "regtest":
			chainParams = &chaincfg.RegressionNetParams
		default:
			log.Printf("Warning: Unknown chain '%s', defaulting to MainNet parameters", chainInfo.Chain)
			chainParams = &chaincfg.MainNetParams
		}

		fmt.Println("Scanning for VPN providers...")
		endpoints, err := ScanForVPNs(client, *scanStartBlock)
		if err != nil {
			log.Fatalf("Failed to scan for VPNs: %v", err)
		}

		if len(endpoints) == 0 {
			fmt.Println("No VPN endpoints found.")
			return
		}

		// Enrich endpoints with GeoIP data
		fmt.Println("Enriching providers with GeoIP and latency tests...")
		enrichedEndpoints := EnrichEndpoints(endpoints)

		// Filter by country if specified
		var filteredEndpoints []*EnrichedVPNEndpoint
		if *scanCountry != "" {
			for _, ep := range enrichedEndpoints {
				if strings.EqualFold(ep.Country, *scanCountry) {
					filteredEndpoints = append(filteredEndpoints, ep)
				}
			}
		} else {
			filteredEndpoints = enrichedEndpoints
		}

		if len(filteredEndpoints) == 0 {
			fmt.Println("No VPN endpoints found matching your criteria.")
			return
		}

		// Sort the results
		switch strings.ToLower(*scanSortBy) {
		case "price":
			sort.Slice(filteredEndpoints, func(i, j int) bool {
				return filteredEndpoints[i].Price < filteredEndpoints[j].Price
			})
			fmt.Println("Sorted by price (lowest first).")
		case "country":
			sort.Slice(filteredEndpoints, func(i, j int) bool {
				return filteredEndpoints[i].Country < filteredEndpoints[j].Country
			})
			fmt.Println("Sorted by country.")
		case "latency":
			sort.Slice(filteredEndpoints, func(i, j int) bool {
				return filteredEndpoints[i].Latency < filteredEndpoints[j].Latency
			})
			fmt.Println("Sorted by latency (lowest first).")
		}

		fmt.Printf("\nAvailable VPN endpoints:\n")
		for i, ep := range filteredEndpoints {
			fmt.Printf("  [%d] Country: %s, Latency: %v, IP: %s, Port: %d, Price: %d sats/session\n", i, ep.Country, ep.Latency.Round(time.Millisecond), ep.Endpoint.IP, ep.Endpoint.Port, ep.Endpoint.Price)
		}
		fmt.Println()

		// Prompt for selection
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter the number of the provider to connect to (or press Enter to quit): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			fmt.Println("Exiting.")
			return
		}

		choice, err := strconv.Atoi(input)
		if err != nil || choice < 0 || choice >= len(filteredEndpoints) {
			log.Fatalf("Invalid selection: %q", input)
		}

		selectedEndpoint := filteredEndpoints[choice]

		// Generate a local key pair for this session *before* paying.
		// In a real app, you might save and reuse this private key.
		localKey, err := wgtypes.GeneratePrivateKey()
		if err != nil {
			log.Fatalf("Failed to generate local private key: %v", err)
		}
		fmt.Printf("\nGenerated temporary client public key: %s\n", localKey.PublicKey().String())

		// Get provider's payment address
		fmt.Println("\nDeriving provider's payment address from announcement transaction...")
		providerAddr, err := GetProviderPaymentAddress(client, selectedEndpoint.TxID, chainParams)
		if err != nil {
			log.Fatalf("Could not get provider payment address: %v", err)
		}
		fmt.Printf("Provider's payment address: %s\n", providerAddr.String())
		fmt.Printf("Payment required: %d satoshis\n", selectedEndpoint.ProviderAnnouncement.Endpoint.Price)

		// Confirm payment
		fmt.Print("Proceed with payment? (y/n): ")
		confirm, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(confirm)) != "y" {
			fmt.Println("Payment cancelled. Exiting.")
			return
		}

		// Send payment
		fmt.Println("Sending payment...")
		var paymentTxID *chainhash.Hash
		const maxRetries = 3
		for i := 0; i < maxRetries; i++ {
			paymentTxID, err = SendPayment(client, providerAddr, selectedEndpoint.ProviderAnnouncement.Endpoint.Price, localKey.PublicKey())
			if err == nil {
				break
			}
			log.Printf("Payment failed (attempt %d/%d): %v. Retrying in 2s...", i+1, maxRetries, err)
			time.Sleep(2 * time.Second)
		}
		if err != nil {
			log.Fatalf("Failed to send payment after %d attempts: %v", maxRetries, err)
		}
		fmt.Printf("Payment sent successfully! Transaction ID: %s\n", paymentTxID.String())

		if len(selectedEndpoint.ProviderAnnouncement.Endpoint.PublicKey) != wgtypes.KeyLen {
			log.Fatal("Provider public key is too short.")
		}
		var peerKey wgtypes.Key
		copy(peerKey[:], selectedEndpoint.ProviderAnnouncement.Endpoint.PublicKey)
		peerKeyB64 := base64.StdEncoding.EncodeToString(peerKey[:])

		endpointAddr := fmt.Sprintf("%s:%d", selectedEndpoint.ProviderAnnouncement.Endpoint.IP.String(), selectedEndpoint.ProviderAnnouncement.Endpoint.Port)

		fmt.Printf("\nConnecting to %s...\n", endpointAddr)
		handleConnectWithKey(cfg.Client.InterfaceName, localKey, peerKeyB64, endpointAddr, *connectMaxLatency)

	case "connect":
		connectCmd.Parse(os.Args[2:])
		// This command is now less useful with the interactive scan, but kept for manual connections.
		// We'll use the client interface name from the config.
		connectIface := cfg.Client.InterfaceName

		if *connectPeerKey == "" || *connectEndpoint == "" {
			log.Fatal("The --peerkey and --endpoint flags are required for the connect command.")
		}
		handleConnect(*connectIface, *connectPeerKey, *connectEndpoint, *connectMaxLatency)

	case "disconnect":
		disconnectCmd.Parse(os.Args[2:])
		fmt.Printf("Tearing down interface %s...\n", *disconnectIface)
		if err := TeardownTunnel(*disconnectIface); err != nil {
			log.Fatalf("Failed to disconnect: %v", err)
		}
		fmt.Printf("Successfully disconnected %s.\n", *disconnectIface)

	case "status":
		statusCmd.Parse(os.Args[2:])
		device, err := GetTunnelStatus(*statusIface)
		if err != nil {
			log.Fatalf("Failed to get status for %s: %v", *statusIface, err)
		}
		printStatus(device)

	case "monitor":
		monitorCmd.Parse(os.Args[2:])
		for {
			fmt.Print("\033[H\033[2J") // Clear screen
			device, err := GetTunnelStatus(*monitorIface)
			if err != nil {
				log.Printf("Failed to get status for %s: %v. Is the interface up?", *monitorIface, err)
			} else {
				printStatus(device)
			}
			fmt.Printf("\nMonitoring... (Press Ctrl+C to exit)\n")
			time.Sleep(2 * time.Second)
		}

	case "history":
		historyCmd.Parse(os.Args[2:])
		records, err := LoadHistory()
		if err != nil {
			log.Fatalf("Failed to load history: %v", err)
		}

		if len(records) == 0 {
			fmt.Println("No payment history found.")
			return
		}

		fmt.Printf("%-25s %-15s %-40s %s\n", "Timestamp", "Amount (sats)", "Provider", "TxID")
		fmt.Println(strings.Repeat("-", 120))
		for _, r := range records {
			fmt.Printf("%-25s %-15d %-40s %s\n", r.Timestamp.Format("2006-01-02 15:04:05"), r.Amount, r.Provider, r.TxID)
		}

	default:
		fmt.Println("expected 'generate-config', 'start-provider', 'scan', 'connect', 'disconnect', 'status', 'monitor', or 'history' subcommands")
		os.Exit(1)
	}
}

func printStatus(device *wgtypes.Device) {
	fmt.Printf("Interface: %s\n", device.Name)
	fmt.Printf("  Public Key: %s\n", device.PublicKey.String())
	fmt.Printf("  Listen Port: %d\n", device.ListenPort)
	fmt.Printf("  Peers: %d\n", len(device.Peers))
	for _, peer := range device.Peers {
		fmt.Printf("    - Peer: %s\n", peer.PublicKey.String())
		fmt.Printf("      Endpoint: %s\n", peer.Endpoint.String())
		fmt.Printf("      Allowed IPs: %v\n", peer.AllowedIPs)
		if peer.LastHandshakeTime.IsZero() {
			fmt.Printf("      Latest Handshake: Never\n")
		} else {
			fmt.Printf("      Latest Handshake: %s ago\n", time.Since(peer.LastHandshakeTime).Round(time.Second))
		}
		fmt.Printf("      Transfer: Rx %s, Tx %s\n", formatBytes(peer.ReceiveBytes), formatBytes(peer.TransmitBytes))
	}
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func connectRPC(host, user, pass string) *rpcclient.Client {
	connCfg := &rpcclient.ConnConfig{
		Host:         host,
		User:         user,
		Pass:         pass,
		HTTPPostMode: true,
		DisableTLS:   true,
	}

	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatalf("Error creating new RPC client: %v", err)
	}
	return client
}

func handleConnectWithKey(ifaceName string, localKey wgtypes.Key, peerKeyB64 string, endpointAddr string, maxLatency time.Duration) {
	fmt.Printf("Using local public key: %s\n", localKey.PublicKey().String())

	// 2. Parse the peer's public key.
	peerKey, err := wgtypes.ParseKey(peerKeyB64)
	if err != nil {
		log.Fatalf("Failed to parse peer public key: %v", err)
	}

	// 3. Define the allowed IPs for the peer. For a full tunnel, this is 0.0.0.0/0.
	_, fullTunnel, _ := net.ParseCIDR("0.0.0.0/0")
	allowedIPs := []net.IPNet{*fullTunnel}

	// 4. Configure the tunnel.
	// Note: This function requires elevated privileges to modify network interfaces.
	fmt.Println("Attempting to configure WireGuard interface. This may require root/administrator privileges.")
	err = ConfigureTunnel(ifaceName, localKey, peerKey, endpointAddr, allowedIPs)
	if err != nil {
		log.Fatalf("Failed to configure tunnel: %v", err)
	}
	log.Printf("Successfully configured WireGuard interface %q. You may now need to configure IP addresses and routes.", ifaceName)

	// Monitor and Wait
	log.Println("Tunnel established. Press Ctrl+C to disconnect.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start latency monitor if enabled
	if maxLatency > 0 {
		go func() {
			log.Printf("Monitoring latency (limit: %v)...", maxLatency)
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			// Parse endpoint for measurement
			host, portStr, _ := net.SplitHostPort(endpointAddr)
			port, _ := strconv.Atoi(portStr)
			ep := &VPNEndpoint{IP: net.ParseIP(host), Port: uint16(port)}

			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					lat := measureLatency(ep)
					if lat > maxLatency {
						log.Printf("Latency %v exceeded limit %v. Disconnecting...", lat, maxLatency)
						stop <- os.Interrupt // Trigger shutdown
						return
					}
				}
			}
		}()
	}

	<-stop
	log.Println("Shutting down...")
	TeardownTunnel(ifaceName)
}

func handleConnect(ifaceName, peerKeyB64, endpointAddr string, maxLatency time.Duration) {
	// 1. Generate a local private key for the client for a non-payment connection.
	localKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}
	handleConnectWithKey(ifaceName, localKey, peerKeyB64, endpointAddr, maxLatency)
}

func loadOrGenerateKey(path string) (wgtypes.Key, error) {
	keyData, err := os.ReadFile(path)
	if err == nil {
		return wgtypes.NewKey(keyData)
	}

	log.Printf("Private key not found at %s, generating a new one.", path)
	newKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return wgtypes.Key{}, err
	}
	return newKey, os.WriteFile(path, newKey[:], 0600)
}
