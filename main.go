package main

import (
	"bufio"
	"context"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"time"

	"flag"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
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
	startProviderCmd := flag.NewFlagSet("start-provider", flag.ExitOnError)
	historyCmd := flag.NewFlagSet("history", flag.ExitOnError)
	rebroadcastCmd := flag.NewFlagSet("rebroadcast", flag.ExitOnError)

	// Scan specific flags
	scanStartBlock := scanCmd.Int64("startblock", 0, "Block height to start scanning from (0 for full scan)")
	scanSortBy := scanCmd.String("sort", "latency", "Sort providers by 'price', 'country', or 'latency'")
	scanCountry := scanCmd.String("country", "", "Filter providers by country code (e.g., US, DE)")
	scanDryRun := scanCmd.Bool("dry-run", false, "Simulate connection without spending funds or modifying interfaces")
	historySinceLast := historyCmd.Bool("since-last-payment", false, "Show wallet transactions since the last recorded payment")

	if len(os.Args) < 2 {
		fmt.Println("expected 'generate-config', 'start-provider', 'rebroadcast', 'scan', or 'history' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "start-provider":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		startProviderCmd.Parse(os.Args[2:])
		client := connectRPC(cfg.RPC.Host, cfg.RPC.User, cfg.RPC.Pass)
		defer client.Shutdown()

		// Create the authorization manager
		authManager := NewAuthManager()

		endpoint, providerKey, err := buildProviderEndpoint(&cfg.Provider, cfg.Provider.PrivateKeyFile)
		if err != nil {
			log.Fatalf("Failed to build provider endpoint: %v", err)
		}

		// Start re-announcement loop in a goroutine
		go func() {
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()

			// Announce immediately on start
			if err := AnnounceService(client, endpoint); err != nil {
				log.Printf("Initial service announcement failed: %v", err)
			}

			// Then announce on a schedule, checking for shutdown signal
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := AnnounceService(client, endpoint); err != nil {
						log.Printf("Scheduled re-announcement failed: %v", err)
					}
				}
			}
		}()

		// Start payment monitor in a goroutine
		go MonitorPayments(ctx, client, authManager, cfg.Provider.Price)

		// Start the provider's main server loop
		go StartProviderServer(ctx, &cfg.Provider, providerKey, authManager)

		// Start echo server and block until shutdown
		go StartEchoServer(ctx, cfg.Provider.ListenPort)

		<-ctx.Done()
		log.Println("Shutting down provider...")

	case "rebroadcast":
		rebroadcastCmd.Parse(os.Args[2:])
		client := connectRPC(cfg.RPC.Host, cfg.RPC.User, cfg.RPC.Pass)
		defer client.Shutdown()

		endpoint, _, err := buildProviderEndpoint(&cfg.Provider, cfg.Provider.PrivateKeyFile)
		if err != nil {
			log.Fatalf("Failed to build provider endpoint for rebroadcast: %v", err)
		}

		log.Println("Re-broadcasting service announcement...")
		if err := AnnounceService(client, endpoint); err != nil {
			log.Fatalf("Service announcement failed: %v", err)
		}
		log.Println("Service announcement re-broadcasted successfully.")

	case "scan":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		scanCmd.Parse(os.Args[2:])
		client := connectRPC(cfg.RPC.Host, cfg.RPC.User, cfg.RPC.Pass)
		defer client.Shutdown()

		// Detect Chain by genesis block hash for accuracy.
		genesisHash, err := client.GetBlockHash(0)
		if err != nil {
			log.Fatalf("Failed to get genesis block hash from RPC: %v", err)
		}

		var chainParams *chaincfg.Params
		switch *genesisHash {
		case *chaincfg.MainNetParams.GenesisHash:
			chainParams = &chaincfg.MainNetParams
			log.Println("Detected chain: Bitcoin Mainnet")
		case *chaincfg.TestNet3Params.GenesisHash:
			chainParams = &chaincfg.TestNet3Params
			log.Println("Detected chain: Bitcoin Testnet3")
		case *chaincfg.RegressionNetParams.GenesisHash:
			chainParams = &chaincfg.RegressionNetParams
			log.Println("Detected chain: Bitcoin Regtest")
		case *chaincfg.SimNetParams.GenesisHash:
			chainParams = &chaincfg.SimNetParams
			log.Println("Detected chain: Bitcoin Simnet")
		default:
			log.Fatalf("Unknown blockchain. Genesis hash %s does not match any known Bitcoin chain. This tool currently only supports Bitcoin-based chains.", genesisHash.String())
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

		interactiveConnect(ctx, client, chainParams, filteredEndpoints, &cfg.Client, *scanDryRun)

	case "history":
		historyCmd.Parse(os.Args[2:])

		if *historySinceLast {
			records, err := LoadHistory()
			if err != nil {
				log.Fatalf("Failed to load history: %v", err)
			}
			if len(records) == 0 {
				log.Println("No payment history found to use as a time reference.")
				return
			}

			sort.Slice(records, func(i, j int) bool {
				return records[i].Timestamp.After(records[j].Timestamp)
			})
			lastPayment := records[0]
			log.Printf("Checking for wallet transactions since last payment on %v", lastPayment.Timestamp.Format(time.RFC3339))

			client := connectRPC(cfg.RPC.Host, cfg.RPC.User, cfg.RPC.Pass)
			defer client.Shutdown()

			transactions, err := client.ListTransactionsCount("*", 1000, 0)
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
		} else {
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
		}

	default:
		fmt.Println("expected 'generate-config', 'start-provider', 'rebroadcast', 'scan', or 'history' subcommands")
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

func loadOrGenerateKey(path string) (*btcec.PrivateKey, error) {
	keyDER, err := os.ReadFile(path)
	if err == nil {
		privKey, err := x509.ParseECPrivateKey(keyDER)
		if err != nil {
			return nil, fmt.Errorf("failed to parse existing key: %w", err)
		}
		return btcec.NewPrivateKeyFromECDSA(privKey), nil
	}

	log.Printf("Private key not found at %s, generating a new one.", path)
	newKey, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, err
	}

	// Save the key in a standard format (e.g., marshaled DER)
	derKey, err := x509.MarshalECPrivateKey(newKey.ToECDSA())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal new key: %w", err)
	}

	if err := os.WriteFile(path, derKey, 0600); err != nil {
		return nil, fmt.Errorf("failed to save new key: %w", err)
	}
	return newKey, nil
}

func buildProviderEndpoint(cfg *ProviderConfig, keyPath string) (*VPNEndpoint, *btcec.PrivateKey, error) {
	// Load or generate provider's private key
	providerKey, err := loadOrGenerateKey(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("could not load or generate provider key: %w", err)
	}

	// Auto-locate if country is not configured
	var loc *GeoLocation
	if cfg.Country == "" {
		var err error
		log.Println("Country not set in config, attempting to auto-locate...")
		loc, err = AutoLocate()
		if err != nil {
			log.Printf("Warning: Failed to auto-locate: %v", err)
		} else {
			cfg.Country = loc.CountryCode
			log.Printf("Auto-detected location: %s, %s", loc.City, loc.Country)
		}
	}

	var announceIP net.IP
	if cfg.AnnounceIP != "" {
		announceIP = net.ParseIP(cfg.AnnounceIP)
		if announceIP == nil {
			return nil, nil, fmt.Errorf("invalid AnnounceIP in config: %s", cfg.AnnounceIP)
		}
	} else if loc != nil && loc.Query != "" {
		announceIP = net.ParseIP(loc.Query)
		log.Printf("Using auto-detected public IP: %s", announceIP.String())
	} else {
		log.Println("AnnounceIP not set, attempting to detect public IP...")
		var err error
		announceIP, err = GetPublicIP()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to detect public IP: %w. Please set announce_ip in config.json", err)
		}
		log.Printf("Detected public IP: %s", announceIP.String())
	}

	endpoint := &VPNEndpoint{
		IP:        announceIP,
		Port:      uint16(cfg.ListenPort),
		Price:     cfg.Price,
		PublicKey: providerKey.PubKey().SerializeCompressed(),
	}

	return endpoint, providerKey, nil
}

func interactiveConnect(ctx context.Context, client *rpcclient.Client, chainParams *chaincfg.Params, endpoints []*EnrichedVPNEndpoint, clientCfg *ClientConfig, dryRun bool) {
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
	if err != nil || choice < 0 || choice >= len(endpoints) {
		log.Fatalf("Invalid selection: %q", input)
	}

	selectedEndpoint := endpoints[choice]

	// Generate a local key pair for this session *before* paying.
	// In a real app, you might save and reuse this private key.
	localKey, err := btcec.NewPrivateKey()
	if err != nil {
		log.Fatalf("Failed to generate local private key: %v", err)
	}
	fmt.Printf("\nGenerated temporary client public key: %s\n", localKey.PubKey().String())

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
	if dryRun {
		fmt.Print("(Dry Run) ")
	}
	confirm, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(confirm)) != "y" {
		fmt.Println("Payment cancelled. Exiting.")
		return
	}

	// Send payment
	var paymentTxID *chainhash.Hash
	if dryRun {
		fmt.Println("[Dry Run] Simulation: Payment skipped. No funds spent.")
	} else {
		fmt.Println("Sending payment...")
		const maxRetries = 3
		for i := 0; i < maxRetries; i++ {
			paymentTxID, err = SendPayment(client, providerAddr, selectedEndpoint.ProviderAnnouncement.Endpoint.Price, localKey.PubKey())
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
	}

	peerPubKey, err := btcec.ParsePubKey(selectedEndpoint.ProviderAnnouncement.Endpoint.PublicKey)
	if err != nil {
		log.Fatalf("Failed to parse provider public key: %v", err)
	}

	endpointAddr := fmt.Sprintf("%s:%d", selectedEndpoint.ProviderAnnouncement.Endpoint.IP.String(), selectedEndpoint.ProviderAnnouncement.Endpoint.Port)

	fmt.Printf("\nConnecting to %s...\n", endpointAddr)

	if dryRun {
		fmt.Printf("[Dry Run] Simulation: Would create TUN interface %s and connect to %s.\n", clientCfg.InterfaceName, endpointAddr)
	} else {
		err := ConnectToProvider(ctx, clientCfg, localKey, peerPubKey, endpointAddr)
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
