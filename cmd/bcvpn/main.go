package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"time"

	"blockchain-vpn/internal/auth"
	"blockchain-vpn/internal/blockchain"
	"blockchain-vpn/internal/config"
	"blockchain-vpn/internal/crypto"
	"blockchain-vpn/internal/geoip"
	"blockchain-vpn/internal/history"
	"blockchain-vpn/internal/nat"
	"blockchain-vpn/internal/protocol"
	"blockchain-vpn/internal/tunnel"
	"blockchain-vpn/internal/util"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
)

const defaultConfigFilename = "config.json"

func main() {
	// Handle generate-config before loading config
	if len(os.Args) >= 2 && os.Args[1] == "generate-config" {
		if err := config.GenerateDefaultConfig(defaultConfigFilename); err != nil {
			log.Fatalf("Failed to generate config: %v", err)
		}
		log.Printf("Generated default config at %s", defaultConfigFilename)
		return
	}

	// Load configuration
	cfg, err := config.LoadConfig(defaultConfigFilename)
	if err != nil {
		log.Fatalf("Failed to load config file at %s: %v. Please create one or run 'generate-config'.", defaultConfigFilename, err)
	}

	// Subcommands
	scanCmd := flag.NewFlagSet("scan", flag.ExitOnError)
	startProviderCmd := flag.NewFlagSet("start-provider", flag.ExitOnError)
	historyCmd := flag.NewFlagSet("history", flag.ExitOnError)
	rebroadcastCmd := flag.NewFlagSet("rebroadcast", flag.ExitOnError)
	updatePriceCmd := flag.NewFlagSet("update-price", flag.ExitOnError)

	// Scan specific flags
	scanStartBlock := scanCmd.Int64("startblock", 0, "Block height to start scanning from (0 for full scan)")
	scanSortBy := scanCmd.String("sort", "latency", "Sort providers by 'price', 'country', or 'latency'")
	scanCountry := scanCmd.String("country", "", "Filter providers by country code (e.g., US, DE)")
	scanDryRun := scanCmd.Bool("dry-run", false, "Simulate connection without spending funds or modifying interfaces")
	historySinceLast := historyCmd.Bool("since-last-payment", false, "Show wallet transactions since the last recorded payment")

	// Update-price specific flags
	updatePriceNewPrice := updatePriceCmd.Uint64("price", 0, "The new price in satoshis per session")

	if len(os.Args) < 2 {
		fmt.Println("expected 'generate-config', 'start-provider', 'rebroadcast', 'update-price', 'scan', or 'history' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "start-provider":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		startProviderCmd.Parse(os.Args[2:])
		client := connectRPC(cfg.RPC.Host, cfg.RPC.User, cfg.RPC.Pass)
		defer client.Shutdown()

		authManager := auth.NewAuthManager()

		providerKey, err := getProviderKey(cfg.Provider.PrivateKeyFile)
		if err != nil {
			log.Fatalf("Failed to get provider key: %v", err)
		}

		announceIP, announcePort, natCleanup, err := determineAnnounceDetails(ctx, &cfg.Provider)
		if err != nil {
			log.Fatalf("Failed to determine announcement IP/Port: %v", err)
		}
		if natCleanup != nil {
			defer natCleanup()
		}

		endpoint := buildProviderEndpoint(cfg.Provider.Price, announceIP, announcePort, providerKey)

		go func() {
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()
			if err := blockchain.AnnounceService(client, endpoint); err != nil {
				log.Printf("Initial service announcement failed: %v", err)
			}
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := blockchain.AnnounceService(client, endpoint); err != nil {
						log.Printf("Scheduled re-announcement failed: %v", err)
					}
				}
			}
		}()

		go blockchain.MonitorPayments(ctx, client, authManager, cfg.Provider.Price)
		go tunnel.StartProviderServer(ctx, &cfg.Provider, providerKey, authManager)
		go blockchain.StartEchoServer(ctx, cfg.Provider.ListenPort)

		<-ctx.Done()
		log.Println("Shutting down provider...")

	case "rebroadcast":
		rebroadcastCmd.Parse(os.Args[2:])
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client := connectRPC(cfg.RPC.Host, cfg.RPC.User, cfg.RPC.Pass)
		defer client.Shutdown()

		providerKey, err := getProviderKey(cfg.Provider.PrivateKeyFile)
		if err != nil {
			log.Fatalf("Failed to get provider key: %v", err)
		}

		announceIP, announcePort, natCleanup, err := determineAnnounceDetails(ctx, &cfg.Provider)
		if err != nil {
			log.Fatalf("Failed to determine announcement IP/Port for rebroadcast: %v", err)
		}
		if natCleanup != nil {
			defer natCleanup()
		}

		endpoint := buildProviderEndpoint(cfg.Provider.Price, announceIP, announcePort, providerKey)

		log.Println("Re-broadcasting service announcement...")
		if err := blockchain.AnnounceService(client, endpoint); err != nil {
			log.Fatalf("Service announcement failed: %v", err)
		}
		log.Println("Service announcement re-broadcasted successfully.")

	case "update-price":
		updatePriceCmd.Parse(os.Args[2:])
		if *updatePriceNewPrice == 0 {
			log.Fatal("--price must be a positive value")
		}

		client := connectRPC(cfg.RPC.Host, cfg.RPC.User, cfg.RPC.Pass)
		defer client.Shutdown()

		providerKey, err := getProviderKey(cfg.Provider.PrivateKeyFile)
		if err != nil {
			log.Fatalf("Failed to get provider key: %v", err)
		}

		log.Printf("Broadcasting price update to %d satoshis...", *updatePriceNewPrice)
		if err := blockchain.AnnouncePriceUpdate(client, providerKey.PubKey(), *updatePriceNewPrice); err != nil {
			log.Fatalf("Price update announcement failed: %v", err)
		}
		log.Println("Price update broadcasted successfully.")

	case "scan":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		scanCmd.Parse(os.Args[2:])
		client := connectRPC(cfg.RPC.Host, cfg.RPC.User, cfg.RPC.Pass)
		defer client.Shutdown()

		genesisHash, err := client.GetBlockHash(0)
		if err != nil {
			log.Fatalf("Failed to get genesis block hash from RPC: %v", err)
		}

		chainParams := detectChain(genesisHash)

		fmt.Println("Scanning for VPN providers and price updates...")
		endpoints, priceUpdates, err := blockchain.ScanForVPNs(client, *scanStartBlock)
		if err != nil {
			log.Fatalf("Failed to scan for VPNs: %v", err)
		}

		if len(endpoints) == 0 {
			fmt.Println("No VPN endpoints found.")
			return
		}

		fmt.Println("Enriching providers with GeoIP and latency tests...")
		enrichedEndpoints := geoip.EnrichEndpoints(endpoints)

		var filteredEndpoints []*geoip.EnrichedVPNEndpoint
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

		sortEndpoints(filteredEndpoints, *scanSortBy)

		fmt.Println()
		for _, ep := range filteredEndpoints {
			pubKeyHex := hex.EncodeToString(ep.Endpoint.PublicKey.SerializeCompressed())
			if newPrice, ok := priceUpdates[pubKeyHex]; ok {
				log.Printf("  -> Price for provider %s updated from %d to %d sats", ep.Endpoint.IP, ep.Endpoint.Price, newPrice)
				ep.Endpoint.Price = newPrice
			}
		}

		fmt.Printf("\nAvailable VPN endpoints:\n")
		for i, ep := range filteredEndpoints {
			fmt.Printf("  [%d] Country: %s, Latency: %v, IP: %s, Port: %d, Price: %d sats/session\n", i, ep.Country, ep.Latency.Round(time.Millisecond), ep.Endpoint.IP, ep.Endpoint.Port, ep.Endpoint.Price)
		}
		fmt.Println()

		interactiveConnect(ctx, client, chainParams, filteredEndpoints, &cfg.Client, *scanDryRun)

	case "history":
		historyCmd.Parse(os.Args[2:])

		if *historySinceLast {
			handleHistorySinceLast(cfg)
		} else {
			handleFullHistory()
		}

	default:
		fmt.Println("expected 'generate-config', 'start-provider', 'rebroadcast', 'update-price', 'scan', or 'history' subcommands")
		os.Exit(1)
	}
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

func getProviderKey(keyPath string) (*btcec.PrivateKey, error) {
	reader := bufio.NewReader(os.Stdin)
	if _, err := os.Stat(keyPath); err == nil {
		fmt.Print("Enter password to decrypt provider key: ")
		pass, _ := reader.ReadString('\n')
		password := []byte(strings.TrimSpace(pass))
		key, err := crypto.LoadAndDecryptKey(keyPath, password)
		if err != nil {
			return nil, fmt.Errorf("failed to load and decrypt key: %w", err)
		}
		log.Println("Provider key successfully decrypted.")
		return key, nil
	}

	fmt.Println("No provider key found. Let's create a new encrypted key.")
	fmt.Print("Enter new password for provider key: ")
	pass1, _ := reader.ReadString('\n')
	fmt.Print("Confirm new password: ")
	pass2, _ := reader.ReadString('\n')
	if strings.TrimSpace(pass1) != strings.TrimSpace(pass2) {
		return nil, fmt.Errorf("passwords do not match")
	}
	password := []byte(strings.TrimSpace(pass1))
	if len(password) == 0 {
		return nil, fmt.Errorf("password cannot be empty")
	}

	key, err := crypto.GenerateAndEncryptKey(keyPath, password)
	if err != nil {
		return nil, fmt.Errorf("failed to generate and encrypt key: %w", err)
	}
	log.Printf("New encrypted provider key saved to %s", keyPath)
	return key, nil
}

func determineAnnounceDetails(ctx context.Context, cfg *config.ProviderConfig) (net.IP, int, func(), error) {
	announcePort := cfg.ListenPort
	if cfg.EnableNAT {
		log.Println("NAT traversal enabled, attempting to map ports...")
		mapping, err := nat.DiscoverAndMapPorts(ctx, cfg.ListenPort, cfg.ListenPort)
		if err == nil {
			log.Printf("NAT traversal successful. Announcing external IP %s and port %d", mapping.ExternalIP, mapping.TCPPort)
			return mapping.ExternalIP, mapping.TCPPort, mapping.Cleanup, nil
		}
		log.Printf("Warning: NAT traversal failed: %v. Falling back to other IP detection methods.", err)
	}

	log.Println("Attempting to determine public IP via web services or GeoIP...")
	if cfg.AnnounceIP != "" {
		if ip := net.ParseIP(cfg.AnnounceIP); ip != nil {
			return ip, announcePort, nil, nil
		}
	}

	if loc, err := geoip.AutoLocate(); err == nil && loc.Query != "" {
		if ip := net.ParseIP(loc.Query); ip != nil {
			return ip, announcePort, nil, nil
		}
	}

	ip, err := util.GetPublicIP()
	if err != nil {
		return nil, 0, nil, fmt.Errorf("all IP detection methods failed: %w", err)
	}
	return ip, announcePort, nil, nil
}

func buildProviderEndpoint(price uint64, announceIP net.IP, announcePort int, providerKey *btcec.PrivateKey) *protocol.VPNEndpoint {
	return &protocol.VPNEndpoint{
		IP:        announceIP,
		Port:      uint16(announcePort),
		Price:     price,
		PublicKey: providerKey.PubKey(),
	}
}

func interactiveConnect(ctx context.Context, client *rpcclient.Client, chainParams *chaincfg.Params, endpoints []*geoip.EnrichedVPNEndpoint, clientCfg *config.ClientConfig, dryRun bool) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter the number of the provider to connect to (or press Enter to quit): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		fmt.Println("Exiting.")
		return
	}
	choice, err := strconv.Atoi(input)
	if err != nil || choice < 0 || choice >= len(endpoints) {
		log.Fatalf("Invalid selection: %q", input)
	}
	selectedEndpoint := endpoints[choice]

	localKey, err := btcec.NewPrivateKey()
	if err != nil {
		log.Fatalf("Failed to generate local private key: %v", err)
	}
	fmt.Printf("\nGenerated temporary client public key: %s\n", hex.EncodeToString(localKey.PubKey().SerializeCompressed()))

	fmt.Println("\nDeriving provider's payment address from announcement transaction...")
	providerAddr, err := blockchain.GetProviderPaymentAddress(client, selectedEndpoint.TxID, chainParams)
	if err != nil {
		log.Fatalf("Could not get provider payment address: %v", err)
	}
	fmt.Printf("Provider's payment address: %s\n", providerAddr.String())
	fmt.Printf("Payment required: %d satoshis\n", selectedEndpoint.Endpoint.Price)

	fmt.Print("Proceed with payment? (y/n): ")
	if dryRun {
		fmt.Print("(Dry Run) ")
	}
	confirm, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(confirm)) != "y" {
		fmt.Println("Payment cancelled. Exiting.")
		return
	}

	if dryRun {
		fmt.Println("[Dry Run] Simulation: Payment skipped. No funds spent.")
	} else {
		fmt.Println("Sending payment...")
		_, err := blockchain.SendPayment(client, providerAddr, selectedEndpoint.Endpoint.Price, localKey.PubKey())
		if err != nil {
			log.Fatalf("Failed to send payment: %v", err)
		}
	}

	peerPubKey := selectedEndpoint.Endpoint.PublicKey
	endpointAddr := fmt.Sprintf("%s:%d", selectedEndpoint.Endpoint.IP.String(), selectedEndpoint.Endpoint.Port)
	fmt.Printf("\nConnecting to %s...\n", endpointAddr)

	if dryRun {
		fmt.Printf("[Dry Run] Simulation: Would create TUN interface %s and connect to %s.\n", clientCfg.InterfaceName, endpointAddr)
	} else {
		err := tunnel.ConnectToProvider(ctx, clientCfg, localKey, peerPubKey, endpointAddr)
		select {
		case <-ctx.Done():
			log.Println("Disconnecting...")
		default:
			if err != nil {
				log.Fatalf("Connection failed: %v", err)
			}
		}
	}
}

func detectChain(genesisHash *chainhash.Hash) *chaincfg.Params {
	switch *genesisHash {
	case *chaincfg.MainNetParams.GenesisHash:
		log.Println("Detected chain: Bitcoin Mainnet")
		return &chaincfg.MainNetParams
	case *chaincfg.TestNet3Params.GenesisHash:
		log.Println("Detected chain: Bitcoin Testnet3")
		return &chaincfg.TestNet3Params
	case *chaincfg.RegressionNetParams.GenesisHash:
		log.Println("Detected chain: Bitcoin Regtest")
		return &chaincfg.RegressionNetParams
	case *chaincfg.SimNetParams.GenesisHash:
		log.Println("Detected chain: Bitcoin Simnet")
		return &chaincfg.SimNetParams
	default:
		log.Fatalf("Unknown blockchain. Genesis hash %s does not match any known Bitcoin chain.", genesisHash.String())
		return nil
	}
}

func sortEndpoints(endpoints []*geoip.EnrichedVPNEndpoint, sortBy string) {
	switch strings.ToLower(sortBy) {
	case "price":
		sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].Endpoint.Price < endpoints[j].Endpoint.Price })
		fmt.Println("Sorted by price (lowest first).")
	case "country":
		sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].Country < endpoints[j].Country })
		fmt.Println("Sorted by country.")
	case "latency":
		sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].Latency < endpoints[j].Latency })
		fmt.Println("Sorted by latency (lowest first).")
	}
}

func handleFullHistory() {
	records, err := history.LoadHistory()
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
}

func handleHistorySinceLast(cfg *config.Config) {
	records, err := history.LoadHistory()
	if err != nil {
		log.Fatalf("Failed to load history: %v", err)
	}
	if len(records) == 0 {
		log.Println("No payment history found to use as a time reference.")
		return
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Timestamp.After(records[j].Timestamp) })
	lastPayment := records[0]
	log.Printf("Checking for wallet transactions since last payment on %v", lastPayment.Timestamp.Format(time.RFC3339))

	client := connectRPC(cfg.RPC.Host, cfg.RPC.User, cfg.RPC.Pass)
	defer client.Shutdown()

	transactions, err := client.ListTransactionsCount("*", 1000)
	if err != nil {
		log.Fatalf("Failed to list transactions: %v", err)
	}

	fmt.Println("\nRecent wallet transactions since last payment:")
	fmt.Printf("%-10s %-25s %-15s %s\n", "Category", "Timestamp", "Amount", "TxID")
	fmt.Println(strings.Repeat("-", 100))

	found := false
	for _, tx := range transactions {
		txTime := time.Unix(tx.Time, 0)
		if txTime.After(lastPayment.Timestamp) {
			found = true
			fmt.Printf("%-10s %-25s %-15.8f %s\n", tx.Category, txTime.Format("2006-01-02 15:04:05"), tx.Amount, tx.TxID)
		}
	}
	if !found {
		fmt.Println("No new transactions found.")
	}
}
