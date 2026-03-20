package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"blockchain-vpn/internal/auth"
	"blockchain-vpn/internal/blockchain"
	"blockchain-vpn/internal/config"
	"blockchain-vpn/internal/crypto"
	"blockchain-vpn/internal/geoip"
	"blockchain-vpn/internal/history"
	"blockchain-vpn/internal/nat"
	"blockchain-vpn/internal/obs"
	"blockchain-vpn/internal/protocol"
	"blockchain-vpn/internal/tunnel"
	"blockchain-vpn/internal/util"
	"blockchain-vpn/internal/version"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"golang.org/x/term"
)

// handleError logs the error and exits with code 1.
// This function is designed to replace direct log.Fatalf calls in command handlers,
// making them testable while maintaining the same behavior for CLI users.
func handleError(err error) {
	if err != nil {
		log.Fatalf("%v", err)
	}
}

// handleErrorFn calls the function and handles any error returned.
// This pattern allows command handlers to return errors for testability.
func handleErrorFn(fn func() error) {
	handleError(fn())
}

var (
	activeTunnelManager *tunnel.MultiTunnelManager // for disconnect command
	activeTunnelCtx     context.Context
	activeTunnelCancel  context.CancelFunc
)

func main() {
	// Handle help and version flags early, before requiring config
	if len(os.Args) >= 2 && (os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help") {
		if len(os.Args) >= 3 {
			switch os.Args[2] {
			case "config":
				printConfigHelp()
				os.Exit(0)
			case "scan":
				printScanHelp()
				os.Exit(0)
			case "start-provider":
				printStartProviderHelp()
				os.Exit(0)
			case "connect":
				printConnectHelp()
				os.Exit(0)
			case "generate-config":
				printGenerateConfigHelp()
				os.Exit(0)
			case "version":
				printVersionHelp()
				os.Exit(0)
			case "about":
				printAboutHelp()
				os.Exit(0)
			case "status":
				printStatusHelp()
				os.Exit(0)
			case "events":
				printEventsHelp()
				os.Exit(0)
			case "doctor":
				printDoctorHelp()
				os.Exit(0)
			case "diagnostics":
				printDiagnosticsHelp()
				os.Exit(0)
			case "history":
				printHistoryHelp()
				os.Exit(0)
			case "generate-send-address":
				printGenerateSendAddressHelp()
				os.Exit(0)
			case "generate-receive-address":
				printGenerateReceiveAddressHelp()
				os.Exit(0)
			case "generate-tls-keypair":
				printGenerateTLSKeypairHelp()
				os.Exit(0)
			case "favorite":
				printFavoriteHelp()
				os.Exit(0)
			case "rate":
				printRateHelp()
				os.Exit(0)
			case "generate-provider-key":
				printGenerateProviderKeyHelp()
				os.Exit(0)
			case "disconnect":
				printDisconnectHelp()
				os.Exit(0)
			case "stop-provider":
				printStopProviderHelp()
				os.Exit(0)
			case "restart-provider":
				printRestartProviderHelp()
				os.Exit(0)
			case "rotate-provider-key":
				printRotateProviderKeyHelp()
				os.Exit(0)
			case "rebroadcast":
				printRebroadcastHelp()
				os.Exit(0)
			case "broadcast":
				printRebroadcastHelp()
				os.Exit(0)
			case "setup":
				printSetupHelp()
				os.Exit(0)
			}
		}
		printHelp()
		os.Exit(0)
	}

	if len(os.Args) >= 2 && (os.Args[1] == "-v" || os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Println(version.String())
		os.Exit(0)
	}

	if len(os.Args) >= 2 && (os.Args[1] == "-a" || os.Args[1] == "--about") {
		handleAbout(false)
		os.Exit(0)
	}

	defaultConfigPath, err := config.ResolveConfigPath()
	if err != nil {
		log.Fatalf("Failed to resolve default config path: %v", err)
	}

	// Handle generate-config before loading config
	if len(os.Args) >= 2 && os.Args[1] == "generate-config" {
		if err := config.GenerateDefaultConfig(defaultConfigPath); err != nil {
			log.Fatalf("Failed to generate config: %v", err)
		}
		log.Printf("Generated default config at %s", defaultConfigPath)
		return
	}

	configPath := resolveConfigPath(defaultConfigPath)

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config file at %s: %v. Please create one or run 'generate-config'.", configPath, err)
	}
	if err := config.ResolveProviderKeyPath(cfg, configPath); err != nil {
		log.Fatalf("Failed to resolve provider key path: %v", err)
	}
	applyConfigDefaults(cfg)
	logFormat := strings.TrimSpace(cfg.Logging.Format)
	logLevel := strings.TrimSpace(cfg.Logging.Level)
	if env := strings.TrimSpace(os.Getenv("BCVPN_LOG_FORMAT")); env != "" {
		logFormat = env
	}
	if env := strings.TrimSpace(os.Getenv("BCVPN_LOG_LEVEL")); env != "" {
		logLevel = env
	}
	obs.ConfigureLogging(logFormat, logLevel, "bcvpn-cli")
	tunnel.RecoverPendingNetworkStateAndCleanupStaleInterfaces()

	// Subcommands
	scanCmd := flag.NewFlagSet("scan", flag.ExitOnError)
	startProviderCmd := flag.NewFlagSet("start-provider", flag.ExitOnError)
	historyCmd := flag.NewFlagSet("history", flag.ExitOnError)
	rebroadcastCmd := flag.NewFlagSet("rebroadcast", flag.ExitOnError)

	rotateKeyCmd := flag.NewFlagSet("rotate-provider-key", flag.ExitOnError)
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	configCmd := flag.NewFlagSet("config", flag.ExitOnError)
	versionCmd := flag.NewFlagSet("version", flag.ExitOnError)
	aboutCmd := flag.NewFlagSet("about", flag.ExitOnError)
	doctorCmd := flag.NewFlagSet("doctor", flag.ExitOnError)
	eventsCmd := flag.NewFlagSet("events", flag.ExitOnError)
	diagCmd := flag.NewFlagSet("diagnostics", flag.ExitOnError)
	connectCmd := flag.NewFlagSet("connect", flag.ExitOnError)
	// New commands per OPERATIONS.md
	generateSendAddressCmd := flag.NewFlagSet("generate-send-address", flag.ExitOnError)
	generateReceiveAddressCmd := flag.NewFlagSet("generate-receive-address", flag.ExitOnError)
	generateTLSKeypairCmd := flag.NewFlagSet("generate-tls-keypair", flag.ExitOnError)
	favoriteCmd := flag.NewFlagSet("favorite", flag.ExitOnError)
	rateCmd := flag.NewFlagSet("rate", flag.ExitOnError)
	rateBroadcast := rateCmd.Bool("broadcast", false, "Also broadcast rating on blockchain (requires RPC configured)")
	generateProviderKeyCmd := flag.NewFlagSet("generate-provider-key", flag.ExitOnError)
	disconnectCmd := flag.NewFlagSet("disconnect", flag.ExitOnError)
	restartProviderCmd := flag.NewFlagSet("restart-provider", flag.ExitOnError)
	stopProviderCmd := flag.NewFlagSet("stop-provider", flag.ExitOnError)
	setupCmd := flag.NewFlagSet("setup", flag.ExitOnError)

	// Scan specific flags
	scanStartBlock := scanCmd.Int64("startblock", -1000, "Block height to start scanning from (negative = relative to tip, e.g. -1000 = last 1000 blocks)")
	scanSortBy := scanCmd.String("sort", "latency", "Sort providers by 'price', 'country', 'latency', 'bandwidth', 'capacity', or 'score'")
	scanCountry := scanCmd.String("country", "", "Filter providers by country code (e.g., US, DE)")
	scanMaxPrice := scanCmd.Uint64("max-price", 0, "Filter providers with price <= this value (units depend on pricing method) (0 disables)")
	scanPricingMethod := scanCmd.String("pricing-method", "", "Filter providers by pricing method: session, time, data")
	scanMinBandwidthKB := scanCmd.Uint("min-bandwidth-kbps", 0, "Filter providers with advertised bandwidth >= this value in Kbps")
	scanMaxLatencyMS := scanCmd.Int("max-latency-ms", 0, "Filter providers with latency <= this value in ms (0 disables)")
	scanMinSlots := scanCmd.Int("min-available-slots", 0, "Filter providers with available slot capacity >= this value (0 disables)")
	scanDryRun := scanCmd.Bool("dry-run", false, "Simulate connection without spending funds or modifying interfaces")
	scanMinScore := scanCmd.Float64("min-score", 0, "Minimum provider score (0-100) to include")
	scanLimit := scanCmd.Int("limit", 0, "Maximum number of results to display (0 = unlimited)")
	scanRescan := scanCmd.Bool("rescan", false, "Force a fresh full scan, ignoring cached results")

	// Connect specific flags
	connectPort := connectCmd.Int("port", 51820, "Provider port")
	connectPubkey := connectCmd.String("pubkey", "", "Provider public key (hex)")
	connectPrice := connectCmd.Uint64("price", 0, "Expected price in satoshis for verification")
	connectDNS := connectCmd.String("dns", "", "Custom DNS servers (comma-separated)")
	connectNoAutoDNS := connectCmd.Bool("no-auto-dns", false, "Skip automatic DNS configuration")
	connectNoAutoRoute := connectCmd.Bool("no-auto-route", false, "Skip automatic routing configuration")
	connectDryRun := connectCmd.Bool("dry-run", false, "Simulate connection without spending funds")
	connectSpendingLimit := connectCmd.Uint64("spending-limit", 0, "Maximum total spending in satoshis")
	connectMaxSessionSpending := connectCmd.Uint64("max-session-spending", 0, "Maximum per-session spending in satoshis")
	connectInterface := connectCmd.String("interface", "", "TUN interface name")
	connectTunIP := connectCmd.String("tun-ip", "", "Client TUN IP address")
	connectKillSwitch := connectCmd.Bool("kill-switch", false, "Enable kill switch")
	connectStrictVerification := connectCmd.Bool("strict-verification", false, "Enable strict security verification")
	connectAutoReconnect := connectCmd.Bool("auto-reconnect", false, "Enable automatic reconnection on disconnect")
	connectAutoReconnectMaxAttempts := connectCmd.Int("auto-reconnect-max-attempts", 0, "Maximum reconnection attempts (0 = infinite)")
	connectAutoReconnectInterval := connectCmd.String("auto-reconnect-interval", "5s", "Base interval between reconnection attempts")
	connectAutoReconnectMaxInterval := connectCmd.String("auto-reconnect-max-interval", "5m", "Maximum interval between reconnection attempts")

	historySinceLast := historyCmd.Bool("since-last-payment", false, "Show wallet transactions since the last recorded payment")
	historyFrom := historyCmd.String("from", "", "Show transactions from this date/time (RFC3339 format)")
	historyTo := historyCmd.String("to", "", "Show transactions to this date/time (RFC3339 format)")
	historyJSON := historyCmd.Bool("json", false, "Output in machine-readable JSON format")
	historyTable := historyCmd.Bool("table", false, "Output in table format")
	startProviderKeyPassEnv := startProviderCmd.String("key-password-env", "", "Env var name containing provider key password (file mode)")
	rebroadcastKeyPassEnv := rebroadcastCmd.String("key-password-env", "", "Env var name containing provider key password (file mode)")

	// Update-price specific flags

	rotateKeyPath := rotateKeyCmd.String("key-file", "", "Provider private key file to rotate (defaults to provider.private_key_file from config)")
	rotateOldPassEnv := rotateKeyCmd.String("old-password-env", "", "Env var name containing current key password (file mode)")
	rotateNewPassEnv := rotateKeyCmd.String("new-password-env", "", "Env var name containing new key password (file mode)")
	statusJSON := statusCmd.Bool("json", false, "Output status in machine-readable JSON format")
	configJSON := configCmd.Bool("json", false, "Output in machine-readable JSON format")
	versionJSON := versionCmd.Bool("json", false, "Output version in machine-readable JSON format")
	doctorJSON := doctorCmd.Bool("json", false, "Output doctor results in machine-readable JSON format")
	eventsJSON := eventsCmd.Bool("json", false, "Output events in JSON format")
	eventsLimit := eventsCmd.Int("limit", 100, "Maximum number of recent events to show")
	diagOut := diagCmd.String("out", "", "Output path for diagnostics JSON bundle (default: app config dir)")
	startProviderDryRun := startProviderCmd.Bool("dry-run", false, "Simulate provider operations without making real transactions or writing PID file")
	rebroadcastDryRun := rebroadcastCmd.Bool("dry-run", false, "Simulate announcement without making real transaction")
	generateProviderKeyDryRun := generateProviderKeyCmd.Bool("dry-run", false, "Show what would be generated without creating files")

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "start-provider":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		startProviderCmd.Parse(os.Args[2:])
		if *startProviderDryRun {
			fmt.Println("[Dry Run] Simulation mode - no actual provider startup, no transactions, no PID file.")
			fmt.Println("Would perform the following actions:")
			fmt.Println(" - Check for elevated privileges")
			fmt.Println(" - Load/generate provider key")
			fmt.Println(" - Determine announce IP/port")
			fmt.Println(" - Build provider endpoint")
			fmt.Println(" - Announce service on blockchain")
			fmt.Println(" - Broadcast initial heartbeat")
			fmt.Println(" - Start payment monitor")
			fmt.Println(" - Start provider server")
			fmt.Println(" - Start echo server")
			fmt.Println(" - Write PID file")
			return
		}

		if err := tunnel.EnsureElevatedPrivileges(); err != nil {
			log.Printf("WARNING: Provider mode requires automatic networking privileges.")
			log.Printf("Provider functionality will be limited without elevated privileges.")
			log.Printf("Hint: Run with sudo or as Administrator for full functionality.")
			log.Fatalf("Provider mode requires automatic networking privileges: %v", err)
		}

		client := connectRPCWithConfig(cfg)
		defer client.Shutdown()

		addressType := cfg.Provider.AddressType
		if strings.ToLower(addressType) == "auto" || addressType == "" {
			detected, err := blockchain.DetectAddressType(client)
			if err != nil {
				log.Printf("Warning: could not auto-detect address type: %v (using p2pkh fallback)", err)
				addressType = "p2pkh"
			} else {
				addressType = detected
				log.Printf("Auto-detected wallet address type: %s", addressType)
			}
		} else {
			log.Printf("Using configured address type: %s", addressType)
		}

		feeCfg := blockchain.NewFeeConfig(cfg.RPC.MinRelayFee, cfg.RPC.DefaultFeeKb)

		authManager := auth.NewAuthManager()

		providerKey, err := getProviderKey(cfg, *startProviderKeyPassEnv)
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

		// Measure bandwidth if auto-test is enabled
		var measuredBandwidthKB uint32
		if cfg.Provider.BandwidthAutoTest {
			log.Printf("Running bandwidth auto-test...")
			if bw, err := tunnel.MeasureLocalBandwidthKbps(ctx); err != nil {
				log.Printf("Warning: bandwidth auto-test failed: %v (advertising configured limit)", err)
			} else {
				measuredBandwidthKB = bw
				log.Printf("Bandwidth auto-test result: %d Kbps", measuredBandwidthKB)
			}
		}

		endpoint := buildProviderEndpoint(&cfg.Provider, announceIP, announcePort, providerKey, measuredBandwidthKB)

		var providerWG sync.WaitGroup
		providerWG.Add(5)
		go func() {
			defer providerWG.Done()
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()
			if err := blockchain.AnnounceService(client, endpoint, cfg.Provider.AnnouncementFeeTargetBlocks, cfg.Provider.AnnouncementFeeMode, addressType, feeCfg); err != nil {
				log.Printf("Initial service announcement failed: %v", err)
			}
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := blockchain.AnnounceService(client, endpoint, cfg.Provider.AnnouncementFeeTargetBlocks, cfg.Provider.AnnouncementFeeMode, addressType, feeCfg); err != nil {
						log.Printf("Scheduled re-announcement failed: %v", err)
					}
				}
			}
		}()
		go func() {
			defer providerWG.Done()
			hbInterval, _ := time.ParseDuration(cfg.Provider.HeartbeatInterval)
			if hbInterval <= 0 {
				hbInterval = 5 * time.Minute
			}
			hbTicker := time.NewTicker(hbInterval)
			defer hbTicker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-hbTicker.C:
					if err := blockchain.AnnounceHeartbeat(client, providerKey.PubKey(), protocol.AvailabilityFlagAvailable, addressType, feeCfg); err != nil {
						log.Printf("Scheduled heartbeat broadcast failed: %v", err)
					}
				}
			}
		}()
		// Build payment monitor configuration from provider settings
		pmCfg := blockchain.PaymentMonitorCfg{
			Price:                 cfg.Provider.Price,
			PricingMethod:         strings.ToLower(strings.TrimSpace(cfg.Provider.PricingMethod)),
			MaxSessionSecs:        cfg.Provider.MaxSessionDurationSecs,
			RequiredConfirmations: cfg.Provider.PaymentRequiredConfirmations,
		}
		// Parse billing units based on pricing method
		switch pmCfg.PricingMethod {
		case "time":
			switch strings.ToLower(strings.TrimSpace(cfg.Provider.BillingTimeUnit)) {
			case "minute":
				pmCfg.TimeUnitSecs = 60
			case "hour":
				pmCfg.TimeUnitSecs = 3600
			default:
				pmCfg.TimeUnitSecs = 60
			}
		case "data":
			switch strings.ToLower(strings.TrimSpace(cfg.Provider.BillingDataUnit)) {
			case "mb":
				pmCfg.DataUnitBytes = 1_000_000
			case "gb":
				pmCfg.DataUnitBytes = 1_000_000_000
			default:
				pmCfg.DataUnitBytes = 1_000_000
			}
		}

		go func() {
			defer providerWG.Done()
			pmInterval, _ := time.ParseDuration(cfg.Provider.PaymentMonitorInterval)
			blockchain.MonitorPayments(ctx, client, authManager, pmCfg, pmInterval)
		}()
		go func() {
			defer providerWG.Done()
			if err := tunnel.StartProviderServer(ctx, &cfg.Provider, &cfg.Security, providerKey, authManager); err != nil {
				log.Printf("Provider server exited with error: %v", err)
				stop()
			}
		}()
		go func() {
			defer providerWG.Done()
			if err := blockchain.StartEchoServer(ctx, cfg.Provider.ListenPort); err != nil {
				log.Printf("Echo server exited with error: %v", err)
				stop()
			}
		}()
		go func() {
			defer providerWG.Done()
			if cfg.Provider.ThroughputProbePort > 0 {
				if err := tunnel.StartThroughputServer(ctx, cfg.Provider.ThroughputProbePort); err != nil {
					log.Printf("Throughput server exited with error: %v", err)
					stop()
				}
			}
		}()

		// Write PID file to allow stop/restart management
		pidPath, err := config.ResolveProviderPIDFilePath(cfg, configPath)
		if err != nil {
			log.Printf("Warning: failed to resolve PID file path: %v", err)
		} else {
			if err := writePIDFile(pidPath); err != nil {
				log.Printf("Warning: failed to write PID file: %v", err)
			} else {
				log.Printf("PID file written to %s (PID: %d)", pidPath, os.Getpid())
				// Ensure PID file is removed on shutdown
				defer os.Remove(pidPath)
			}
		}

		<-ctx.Done()
		log.Println("Shutting down provider...")
		done := make(chan struct{})
		go func() {
			defer close(done)
			providerWG.Wait()
		}()

		shutdownTimeout := 10 * time.Second
		if cfg.Provider.ShutdownTimeout != "" {
			if parsed, err := time.ParseDuration(cfg.Provider.ShutdownTimeout); err == nil {
				shutdownTimeout = parsed
			}
		}
		select {
		case <-done:
		case <-time.After(shutdownTimeout):
			log.Printf("Provider shutdown timeout reached; forcing exit.")
		}

	case "rebroadcast":
		fallthrough
	case "broadcast":
		rebroadcastCmd.Parse(os.Args[2:])
		if *rebroadcastDryRun {
			fmt.Println("[Dry Run] Simulation: would re-broadcast service announcement on blockchain.")
			fmt.Println("Would use provider key from config:", cfg.Provider.PrivateKeyFile)
			fmt.Println("Would announce at IP:", cfg.Provider.AnnounceIP)
			fmt.Println("Would announce at port:", cfg.Provider.ListenPort)
			fmt.Println("Would use price:", cfg.Provider.Price)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client := connectRPCWithConfig(cfg)
		defer client.Shutdown()

		addressType := cfg.Provider.AddressType
		if strings.ToLower(addressType) == "auto" || addressType == "" {
			detected, err := blockchain.DetectAddressType(client)
			if err != nil {
				log.Printf("Warning: could not auto-detect address type: %v (using p2pkh fallback)", err)
				addressType = "p2pkh"
			} else {
				addressType = detected
				log.Printf("Auto-detected wallet address type: %s", addressType)
			}
		} else {
			log.Printf("Using configured address type: %s", addressType)
		}

		feeCfg := blockchain.NewFeeConfig(cfg.RPC.MinRelayFee, cfg.RPC.DefaultFeeKb)

		providerKey, err := getProviderKey(cfg, *rebroadcastKeyPassEnv)
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

		endpoint := buildProviderEndpoint(&cfg.Provider, announceIP, announcePort, providerKey, 0)

		log.Println("Re-broadcasting service announcement...")
		if err := blockchain.AnnounceService(client, endpoint, cfg.Provider.AnnouncementFeeTargetBlocks, cfg.Provider.AnnouncementFeeMode, addressType, feeCfg); err != nil {
			log.Fatalf("Service announcement failed: %v", err)
		}
		log.Println("Service announcement re-broadcasted successfully.")

	case "scan":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		scanCmd.Parse(os.Args[2:])
		if *scanLimit > 100 {
			log.Fatal("--limit cannot exceed 100")
		}
		client := connectRPCWithConfig(cfg)
		defer client.Shutdown()

		addressType := cfg.Client.AddressType
		if strings.ToLower(addressType) == "auto" || addressType == "" {
			detected, err := blockchain.DetectAddressType(client)
			if err != nil {
				log.Printf("Warning: could not auto-detect address type: %v (using p2pkh fallback)", err)
				addressType = "p2pkh"
			} else {
				addressType = detected
				log.Printf("Auto-detected wallet address type: %s", addressType)
			}
		} else {
			log.Printf("Using configured address type: %s", addressType)
		}

		genesisHash, err := client.GetBlockHash(0)
		if err != nil {
			log.Fatalf("Failed to get genesis block hash from RPC: %v", err)
		}

		_ = detectChain(genesisHash, cfg.RPC.Network)

		cachePath, err := blockchain.DefaultScanCachePath()
		if err != nil {
			log.Printf("Warning: failed to get default scan cache path: %v", err)
		}
		var cache *blockchain.ScanCache
		if cachePath != "" && !*scanRescan {
			cache = blockchain.NewScanCache(cachePath)
			if err := cache.Load(); err != nil {
				log.Printf("Warning: failed to load scan cache: %v", err)
			}
		}

		repPath, err := blockchain.DefaultReputationStorePath()
		if err != nil {
			log.Printf("Warning: failed to get default reputation path: %v", err)
		}
		var repStore *blockchain.ReputationStore
		if repPath != "" {
			repStore, err = blockchain.NewReputationStore(repPath)
			if err != nil {
				log.Printf("Warning: failed to load reputation store: %v", err)
			}
		}

		fmt.Println("Scanning for VPN providers and price updates...")
		endpoints, priceUpdates, err := blockchain.ScanForVPNs(client, *scanStartBlock, cache, repStore)
		handleError(err)

		if len(endpoints) == 0 {
			fmt.Println("No VPN providers found on the blockchain.")
			fmt.Println()
			fmt.Println("Tips:")
			fmt.Println("  - Ensure ordexcoind is running and synchronized")
			fmt.Println("  - Run 'bcvpn start-provider' to become a provider")
			fmt.Println("  - Check 'bcvpn doctor' for diagnostics")
			return
		}

		fmt.Println("Enriching providers with GeoIP and latency tests...")
		enrichedEndpoints := geoip.EnrichEndpoints(endpoints)

		filteredEndpoints := filterEndpoints(
			enrichedEndpoints,
			strings.TrimSpace(*scanCountry),
			*scanMaxPrice,
			uint32(*scanMinBandwidthKB),
			time.Duration(*scanMaxLatencyMS)*time.Millisecond,
			*scanMinSlots,
			strings.TrimSpace(*scanPricingMethod),
			*scanMinScore,
		)

		if len(filteredEndpoints) == 0 {
			fmt.Println("No VPN providers match your filter criteria.")
			fmt.Println()
			fmt.Println("Try relaxing filters:")
			fmt.Println("  --max-price <value>      Increase max price")
			fmt.Println("  --country <code>        Change country filter")
			fmt.Println("  --max-latency-ms <ms>    Increase latency tolerance")
			fmt.Println("  --min-score <score>     Lower minimum score")
			return
		}

		sortEndpoints(filteredEndpoints, *scanSortBy)

		if *scanLimit > 0 && *scanLimit < len(filteredEndpoints) {
			filteredEndpoints = filteredEndpoints[:*scanLimit]
		}

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
			ratingStr := ""
			if ep.ReputationScore != 0 {
				ratingStr = fmt.Sprintf(", Rating: %d", ep.ReputationScore)
			}
			fmt.Printf(
				"  [%d] Country: %s, Latency: %v, IP: %s, Port: %d, Price: %d sats/session, Bandwidth: %d Kbps, Capacity: %s, Score: %.2f%s\n",
				i,
				effectiveCountry(ep),
				ep.Latency.Round(time.Millisecond),
				ep.Endpoint.IP,
				ep.Endpoint.Port,
				ep.Endpoint.Price,
				ep.AdvertisedBandwidthKB,
				displayCapacity(ep),
				computeProviderScore(ep),
				ratingStr,
			)
		}
		fmt.Println()

		interactiveConnect(ctx, client, blockchain.NewFeeConfig(cfg.RPC.MinRelayFee, cfg.RPC.DefaultFeeKb), addressType, filteredEndpoints, &cfg.Client, &cfg.Security, *scanDryRun, *connectAutoReconnect, *connectAutoReconnectMaxAttempts, *connectAutoReconnectInterval, *connectAutoReconnectMaxInterval)

	case "connect":
		if len(os.Args) < 3 {
			log.Fatal("Usage: bcvpn connect <provider-ip> [options]")
		}
		providerIP := os.Args[2]
		connectCmd.Parse(os.Args[3:])

		fmt.Printf("Connecting to provider at %s:%d...\n", providerIP, *connectPort)
		if *connectPubkey != "" {
			fmt.Printf("Provider pubkey: %s\n", *connectPubkey)
		}
		if *connectPrice > 0 {
			fmt.Printf("Expected price: %d sats\n", *connectPrice)
		}
		if *connectDNS != "" {
			fmt.Printf("Custom DNS servers: %s\n", *connectDNS)
		}
		if *connectDryRun {
			fmt.Println("[Dry Run] Simulation mode - no actual connection will be made")
		}
		if *connectNoAutoDNS || *connectNoAutoRoute {
			fmt.Println("Manual DNS/routing configuration requested")
		}
		if *connectSpendingLimit > 0 {
			fmt.Printf("Spending limit: %d sats\n", *connectSpendingLimit)
		}
		if *connectMaxSessionSpending > 0 {
			fmt.Printf("Max session spending: %d sats\n", *connectMaxSessionSpending)
		}
		if *connectInterface != "" {
			fmt.Printf("Interface: %s\n", *connectInterface)
		}
		if *connectTunIP != "" {
			fmt.Printf("TUN IP: %s\n", *connectTunIP)
		}
		if *connectKillSwitch {
			fmt.Println("Kill switch enabled")
		}
		if *connectStrictVerification {
			fmt.Println("Strict verification enabled")
		}

		fmt.Println()
		fmt.Println("========================================")
		fmt.Println("Direct connect requires provider details.")
		fmt.Println("For the best experience, use 'bcvpn scan' for")
		fmt.Println("interactive provider discovery and connection.")
		fmt.Println("========================================")
		fmt.Println()
		fmt.Println("Quick start:")
		fmt.Println("  bcvpn scan                    # Discover providers")
		fmt.Println("  bcvpn help scan               # View scan options")
		fmt.Println("  bcvpn help connect           # View direct connect options")
		fmt.Println()
		fmt.Println("Direct connect is available with --pubkey, --port, and --price flags.")
		fmt.Println("Example: bcvpn connect 1.2.3.4 --pubkey=... --port=51820 --price=1000")

	case "rotate-provider-key":
		rotateKeyCmd.Parse(os.Args[2:])
		keyPath := cfg.Provider.PrivateKeyFile
		if strings.TrimSpace(*rotateKeyPath) != "" {
			keyPath = strings.TrimSpace(*rotateKeyPath)
		}
		if err := rotateProviderKey(cfg, keyPath, *rotateOldPassEnv, *rotateNewPassEnv); err != nil {
			log.Fatalf("Provider key rotation failed: %v", err)
		}
		log.Printf("Provider key rotated successfully. Re-broadcast your service to publish the new public key.")

	case "history":
		historyCmd.Parse(os.Args[2:])
		handleHistory(cfg, *historySinceLast, *historyFrom, *historyTo, *historyJSON, *historyTable)

	case "status":
		statusCmd.Parse(os.Args[2:])
		handleStatus(cfg, configPath, *statusJSON)

	case "config":
		configCmd.Parse(os.Args[2:])
		handleConfigSubcommand(configPath, cfg, configCmd.Args(), *configJSON)
	case "version":
		versionCmd.Parse(os.Args[2:])
		handleVersion(*versionJSON)
	case "about":
		aboutCmd.Parse(os.Args[2:])
		handleAbout(*versionJSON)
	case "doctor":
		doctorCmd.Parse(os.Args[2:])
		handleDoctor(cfg, *doctorJSON)
	case "events":
		eventsCmd.Parse(os.Args[2:])
		handleEvents(*eventsLimit, *eventsJSON)
	case "diagnostics":
		diagCmd.Parse(os.Args[2:])
		if err := exportDiagnosticsBundle(cfg, configPath, strings.TrimSpace(*diagOut)); err != nil {
			log.Fatalf("diagnostics export failed: %v", err)
		}
		log.Printf("Diagnostics bundle written.")

	case "generate-send-address":
		generateSendAddressCmd.Parse(os.Args[2:])
		handleGenerateSendAddress(cfg)
	case "generate-receive-address":
		generateReceiveAddressCmd.Parse(os.Args[2:])
		handleGenerateReceiveAddress(cfg)
	case "generate-tls-keypair":
		generateTLSKeypairCmd.Parse(os.Args[2:])
		handleGenerateTLSKeypair()
	case "favorite":
		favoriteCmd.Parse(os.Args[2:])
		handleFavorite(cfg, configPath)
	case "rate":
		rateCmd.Parse(os.Args[2:])
		handleRate(cfg, configPath, *rateBroadcast)
	case "generate-provider-key":
		generateProviderKeyCmd.Parse(os.Args[2:])
		handleGenerateProviderKey(cfg, *generateProviderKeyDryRun)
	case "disconnect":
		disconnectCmd.Parse(os.Args[2:])
		handleDisconnect(cfg)
	case "restart-provider":
		restartProviderCmd.Parse(os.Args[2:])
		handleRestartProvider(cfg, configPath)
	case "stop-provider":
		stopProviderCmd.Parse(os.Args[2:])
		handleStopProvider(cfg, configPath)
	case "setup":
		setupCmd.Parse(os.Args[2:])
		handleSetup(configPath)
	default:
		fmt.Printf("unknown command: %s\n\n", os.Args[1])
		printHelp()
		os.Exit(1)
	}
}

type doctorCheck struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Detail  string `json:"detail,omitempty"`
	Remedy  string `json:"remedy,omitempty"`
	Warning bool   `json:"warning,omitempty"`
}

type doctorReport struct {
	GeneratedAt string        `json:"generated_at"`
	Version     string        `json:"version"`
	Checks      []doctorCheck `json:"checks"`
	OK          bool          `json:"ok"`
}

func handleDoctor(cfg *config.Config, jsonMode bool) {
	report := doctorReport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Version:     version.Version,
	}
	addCheck := func(c doctorCheck) {
		report.Checks = append(report.Checks, c)
	}

	if err := config.Validate(cfg); err != nil {
		addCheck(doctorCheck{
			Name:   "config.validate",
			OK:     false,
			Detail: err.Error(),
			Remedy: "Run `bcvpn config validate` and fix invalid fields.",
		})
	} else {
		addCheck(doctorCheck{Name: "config.validate", OK: true})
	}

	resolved, supported, detail := crypto.KeyStorageStatus(cfg.Security.KeyStorageMode)
	addCheck(doctorCheck{
		Name:   "security.keystore",
		OK:     supported,
		Detail: fmt.Sprintf("requested=%s resolved=%s detail=%s", cfg.Security.KeyStorageMode, resolved, detail),
		Remedy: "Set `security.key_storage_mode=file` or install backend prerequisites for chosen mode.",
	})

	if err := tunnel.EnsureElevatedPrivileges(); err != nil {
		addCheck(doctorCheck{
			Name:   "networking.privileges",
			OK:     false,
			Detail: err.Error(),
			Remedy: "Run with elevated privileges (sudo/admin) for networking operations.",
		})
	} else {
		addCheck(doctorCheck{Name: "networking.privileges", OK: true})
	}

	for _, tool := range requiredNetworkingTools(runtime.GOOS) {
		if _, err := exec.LookPath(tool); err != nil {
			addCheck(doctorCheck{
				Name:   "tool." + tool,
				OK:     false,
				Detail: "not found in PATH",
				Remedy: "Install required platform networking utility.",
			})
		} else {
			addCheck(doctorCheck{Name: "tool." + tool, OK: true})
		}
	}

	if strings.TrimSpace(cfg.Security.MetricsAuthToken) == "" && (strings.TrimSpace(cfg.Provider.MetricsListenAddr) != "" || strings.TrimSpace(cfg.Client.MetricsListenAddr) != "") {
		addCheck(doctorCheck{
			Name:    "security.metrics_auth",
			OK:      true,
			Warning: true,
			Detail:  "metrics endpoint configured without auth token",
			Remedy:  "Set `security.metrics_auth_token` or bind metrics endpoint to loopback only.",
		})
	} else {
		addCheck(doctorCheck{Name: "security.metrics_auth", OK: true})
	}

	report.OK = true
	for _, c := range report.Checks {
		if !c.OK {
			report.OK = false
			break
		}
	}

	if jsonMode {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			log.Fatalf("failed to encode doctor JSON: %v", err)
		}
		return
	}

	fmt.Printf("BlockchainVPN Doctor (%s)\n", report.GeneratedAt)
	fmt.Printf("Version: %s\n", report.Version)
	for _, c := range report.Checks {
		state := "OK"
		if !c.OK {
			state = "FAIL"
		} else if c.Warning {
			state = "WARN"
		}
		fmt.Printf("- [%s] %s", state, c.Name)
		if c.Detail != "" {
			fmt.Printf(": %s", c.Detail)
		}
		fmt.Println()
		if c.Remedy != "" && (!c.OK || c.Warning) {
			fmt.Printf("  remedy: %s\n", c.Remedy)
		}
	}
	if report.OK {
		fmt.Println("Doctor result: healthy")
	} else {
		fmt.Println("Doctor result: issues detected")
	}
}

func handleEvents(limit int, jsonMode bool) {
	events := tunnel.GetRecentEvents(limit)
	if jsonMode {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{"events": events})
		return
	}
	if len(events) == 0 {
		fmt.Println("No runtime events recorded yet.")
		return
	}
	for _, ev := range events {
		fmt.Printf("%s [%s] %s: %s\n", ev.Time, ev.Role, ev.Type, ev.Detail)
	}
}

func exportDiagnosticsBundle(cfg *config.Config, configPath, outPath string) error {
	if strings.TrimSpace(outPath) == "" {
		dir, err := config.AppConfigDir()
		if err != nil {
			return err
		}
		outPath = filepath.Join(dir, fmt.Sprintf("diagnostics-%s.json", time.Now().UTC().Format("20060102-150405")))
	}

	redacted := *cfg
	redacted.RPC.Pass = ""
	redacted.Security.MetricsAuthToken = ""

	payload := map[string]any{
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"version":      version.String(),
		"config_path":  configPath,
		"config":       redacted,
		"events":       tunnel.GetRecentEvents(200),
		"runtime":      tunnel.GetRuntimeMetricsSnapshot(),
	}
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		return err
	}
	return util.WriteFileAtomic(outPath, out.Bytes(), 0o644)
}

func requiredNetworkingTools(goos string) []string {
	switch goos {
	case "linux":
		return []string{"ip", "iptables"}
	case "darwin":
		return []string{"ifconfig", "route", "networksetup"}
	case "windows":
		return []string{"netsh", "route", "powershell"}
	default:
		return nil
	}
}

// New handlers per OPERATIONS.md

func generateAddress(cfg *config.Config) {
	client := connectRPCWithConfig(cfg)
	defer client.Shutdown()

	addressType := cfg.Client.AddressType
	if strings.ToLower(addressType) == "auto" || addressType == "" {
		detected, err := blockchain.DetectAddressType(client)
		if err != nil {
			addressType = "p2pkh"
		} else {
			addressType = detected
		}
	}

	addr, err := client.GetNewAddress(addressType)
	if err != nil {
		log.Fatalf("Failed to generate new address: %v", err)
	}
	fmt.Println(addr.String())
}

func handleGenerateSendAddress(cfg *config.Config) {
	generateAddress(cfg)
}

func handleGenerateReceiveAddress(cfg *config.Config) {
	generateAddress(cfg)
}

func handleSetup(configPath string) {
	cfg := &config.Config{}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("BlockchainVPN Interactive Setup")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()
	fmt.Println("This wizard will help you configure BlockchainVPN for first use.")
	fmt.Println("Press Enter to accept the default value shown in brackets.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("--- RPC Configuration ---")
	fmt.Println()

	defaultRPCHost := "localhost:25173"
	fmt.Printf("RPC Host:port [%s]: ", defaultRPCHost)
	rpcHost, _ := reader.ReadString('\n')
	rpcHost = strings.TrimSpace(rpcHost)
	if rpcHost == "" {
		rpcHost = defaultRPCHost
	}
	cfg.RPC.Host = rpcHost

	fmt.Printf("RPC Username [rpcuser]: ")
	rpcUser, _ := reader.ReadString('\n')
	rpcUser = strings.TrimSpace(rpcUser)
	if rpcUser == "" {
		rpcUser = "rpcuser"
	}
	cfg.RPC.User = rpcUser

	fmt.Printf("RPC Password (leave empty to auto-generate): ")
	rpcPass, _ := reader.ReadString('\n')
	rpcPass = strings.TrimSpace(rpcPass)
	if rpcPass == "" {
		genPass, err := config.GenerateRandomRPCPassword()
		if err != nil {
			log.Fatalf("Failed to generate RPC password: %v", err)
		}
		rpcPass = genPass
		fmt.Printf("Auto-generated password: %s\n", rpcPass)
	}
	cfg.RPC.Pass = rpcPass

	fmt.Printf("Network [mainnet]: ")
	network, _ := reader.ReadString('\n')
	network = strings.TrimSpace(strings.ToLower(network))
	if network == "" {
		network = "mainnet"
	}
	cfg.RPC.Network = network

	fmt.Println()
	fmt.Println("--- Logging Configuration ---")
	fmt.Println()
	fmt.Printf("Log Format [text] (text/json): ")
	logFormat, _ := reader.ReadString('\n')
	logFormat = strings.TrimSpace(strings.ToLower(logFormat))
	if logFormat == "" {
		logFormat = "text"
	}
	cfg.Logging.Format = logFormat

	fmt.Printf("Log Level [info] (debug/info/warn/error): ")
	logLevel, _ := reader.ReadString('\n')
	logLevel = strings.TrimSpace(strings.ToLower(logLevel))
	if logLevel == "" {
		logLevel = "info"
	}
	cfg.Logging.Level = logLevel

	fmt.Println()
	fmt.Println("--- Security Configuration ---")
	fmt.Println()
	fmt.Printf("Key Storage Mode [file] (file/keychain/libsecret/dpapi/auto): ")
	keyMode, _ := reader.ReadString('\n')
	keyMode = strings.TrimSpace(strings.ToLower(keyMode))
	if keyMode == "" {
		keyMode = "file"
	}
	cfg.Security.KeyStorageMode = keyMode

	fmt.Printf("TLS Min Version [1.3] (1.2/1.3): ")
	tlsVer, _ := reader.ReadString('\n')
	tlsVer = strings.TrimSpace(strings.ToLower(tlsVer))
	if tlsVer == "" {
		tlsVer = "1.3"
	}
	cfg.Security.TLSMinVersion = tlsVer

	fmt.Printf("TLS Profile [modern] (modern/compat): ")
	tlsProfile, _ := reader.ReadString('\n')
	tlsProfile = strings.TrimSpace(strings.ToLower(tlsProfile))
	if tlsProfile == "" {
		tlsProfile = "modern"
	}
	cfg.Security.TLSProfile = tlsProfile

	fmt.Println()
	fmt.Println("--- Mode Selection ---")
	fmt.Println()
	fmt.Println("Do you want to configure as provider, client, or both?")
	fmt.Println("  1) Client only")
	fmt.Println("  2) Provider only")
	fmt.Println("  3) Both provider and client")
	fmt.Println()
	fmt.Printf("Select [3]: ")

	modeSel, _ := reader.ReadString('\n')
	modeSel = strings.TrimSpace(modeSel)
	if modeSel == "" {
		modeSel = "3"
	}

	isClient := modeSel == "1" || modeSel == "3"
	isProvider := modeSel == "2" || modeSel == "3"

	if isClient {
		fmt.Println()
		fmt.Println("--- Client Configuration ---")
		fmt.Println()
		defaultClientInterface := "bcvpn1"
		fmt.Printf("Interface Name [%s]: ", defaultClientInterface)
		clientInterface, _ := reader.ReadString('\n')
		clientInterface = strings.TrimSpace(clientInterface)
		if clientInterface == "" {
			clientInterface = defaultClientInterface
		}
		cfg.Client.InterfaceName = clientInterface

		defaultClientTunIP := "10.10.0.2"
		fmt.Printf("TUN IP [%s]: ", defaultClientTunIP)
		clientTunIP, _ := reader.ReadString('\n')
		clientTunIP = strings.TrimSpace(clientTunIP)
		if clientTunIP == "" {
			clientTunIP = defaultClientTunIP
		}
		cfg.Client.TunIP = clientTunIP

		defaultClientTunSubnet := "24"
		fmt.Printf("TUN Subnet Prefix [%s]: ", defaultClientTunSubnet)
		clientTunSubnet, _ := reader.ReadString('\n')
		clientTunSubnet = strings.TrimSpace(clientTunSubnet)
		if clientTunSubnet == "" {
			clientTunSubnet = defaultClientTunSubnet
		}
		cfg.Client.TunSubnet = clientTunSubnet

		fmt.Printf("Enable Kill Switch [false] (true/false): ")
		killSwitch, _ := reader.ReadString('\n')
		killSwitch = strings.TrimSpace(strings.ToLower(killSwitch))
		cfg.Client.EnableKillSwitch = killSwitch == "true"

		fmt.Printf("Max Parallel Tunnels [1]: ")
		maxTunnels, _ := reader.ReadString('\n')
		maxTunnels = strings.TrimSpace(maxTunnels)
		if maxTunnels == "" {
			maxTunnels = "1"
		}
		fmt.Sscanf(maxTunnels, "%d", &cfg.Client.MaxParallelTunnels)
	}

	if isProvider {
		fmt.Println()
		fmt.Println("--- Provider Configuration ---")
		fmt.Println()
		defaultProviderInterface := "bcvpn0"
		fmt.Printf("Interface Name [%s]: ", defaultProviderInterface)
		providerInterface, _ := reader.ReadString('\n')
		providerInterface = strings.TrimSpace(providerInterface)
		if providerInterface == "" {
			providerInterface = defaultProviderInterface
		}
		cfg.Provider.InterfaceName = providerInterface

		defaultProviderListenPort := "51820"
		fmt.Printf("Listen Port [%s]: ", defaultProviderListenPort)
		providerPort, _ := reader.ReadString('\n')
		providerPort = strings.TrimSpace(providerPort)
		if providerPort == "" {
			providerPort = defaultProviderListenPort
		}
		fmt.Sscanf(providerPort, "%d", &cfg.Provider.ListenPort)

		fmt.Printf("Announce IP (leave empty for auto-detect): ")
		providerIP, _ := reader.ReadString('\n')
		providerIP = strings.TrimSpace(providerIP)
		cfg.Provider.AnnounceIP = providerIP

		fmt.Printf("Price in satoshis per session [1000]: ")
		providerPrice, _ := reader.ReadString('\n')
		providerPrice = strings.TrimSpace(providerPrice)
		if providerPrice == "" {
			providerPrice = "1000"
		}
		fmt.Sscanf(providerPrice, "%d", &cfg.Provider.Price)

		fmt.Printf("Max Consumers (0=unlimited) [0]: ")
		maxConsumers, _ := reader.ReadString('\n')
		maxConsumers = strings.TrimSpace(maxConsumers)
		if maxConsumers == "" {
			maxConsumers = "0"
		}
		fmt.Sscanf(maxConsumers, "%d", &cfg.Provider.MaxConsumers)

		defaultProviderTunIP := "10.0.0.1"
		fmt.Printf("TUN IP [%s]: ", defaultProviderTunIP)
		providerTunIP, _ := reader.ReadString('\n')
		providerTunIP = strings.TrimSpace(providerTunIP)
		if providerTunIP == "" {
			providerTunIP = defaultProviderTunIP
		}
		cfg.Provider.TunIP = providerTunIP

		defaultProviderTunSubnet := "24"
		fmt.Printf("TUN Subnet Prefix [%s]: ", defaultProviderTunSubnet)
		providerTunSubnet, _ := reader.ReadString('\n')
		providerTunSubnet = strings.TrimSpace(providerTunSubnet)
		if providerTunSubnet == "" {
			providerTunSubnet = defaultProviderTunSubnet
		}
		cfg.Provider.TunSubnet = providerTunSubnet

		fmt.Printf("Enable NAT Traversal [true] (true/false): ")
		enableNAT, _ := reader.ReadString('\n')
		enableNAT = strings.TrimSpace(strings.ToLower(enableNAT))
		cfg.Provider.EnableNAT = enableNAT != "false"

		fmt.Printf("Pricing Method [session] (session/time/data): ")
		pricingMethod, _ := reader.ReadString('\n')
		pricingMethod = strings.TrimSpace(strings.ToLower(pricingMethod))
		if pricingMethod == "" {
			pricingMethod = "session"
		}
		cfg.Provider.PricingMethod = pricingMethod

		fmt.Printf("Bandwidth Limit (e.g., 10mbit, leave empty for unlimited) [0]: ")
		bandwidthLimit, _ := reader.ReadString('\n')
		bandwidthLimit = strings.TrimSpace(bandwidthLimit)
		if bandwidthLimit == "" {
			bandwidthLimit = "0"
		}
		cfg.Provider.BandwidthLimit = bandwidthLimit

		keyPath, _ := config.DefaultProviderKeyPath()
		cfg.Provider.PrivateKeyFile = keyPath

		cfg.Provider.CertLifetimeHours = 720
		cfg.Provider.CertRotateBeforeHours = 24
		cfg.Provider.HealthCheckEnabled = true
		cfg.Provider.HealthCheckInterval = "30s"
		cfg.Provider.BandwidthMonitorInterval = "30s"
		cfg.Provider.AnnouncementInterval = "24h"
		cfg.Provider.ShutdownTimeout = "10s"
		cfg.Provider.HeartbeatInterval = "5m"
		cfg.Provider.MetricsListenAddr = "127.0.0.1:9090"
	}

	cfg.Client.TunIP = "10.10.0.2"
	cfg.Client.TunSubnet = "24"
	cfg.Client.DNSServers = []string{"1.1.1.1", "8.8.8.8"}
	cfg.Provider.TunIP = "10.0.0.1"
	cfg.Provider.TunSubnet = "24"
	cfg.Provider.DNSServers = []string{"1.1.1.1", "8.8.8.8"}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("Validating configuration...")
	fmt.Println(strings.Repeat("=", 60))

	if err := config.Validate(cfg); err != nil {
		fmt.Printf("\nValidation failed:\n%v\n\n", err)
		fmt.Print("Save anyway? (y/N): ")
		save, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(save)) != "y" {
			fmt.Println("Setup cancelled.")
			return
		}
	}

	fmt.Println()
	fmt.Println("Saving configuration...")

	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cfg); err != nil {
		log.Fatalf("Failed to encode config: %v", err)
	}
	if err := util.WriteFileAtomic(configPath, out.Bytes(), 0o644); err != nil {
		log.Fatalf("Failed to write config: %v", err)
	}

	fmt.Println()
	fmt.Printf("Configuration saved to: %s\n", configPath)
	fmt.Println()
	fmt.Println("Setup complete!")
	fmt.Println()
	fmt.Println("Next steps:")
	if isProvider {
		fmt.Println("  1. Run 'bcvpn generate-provider-key' to create your provider identity")
		fmt.Println("  2. Run 'bcvpn start-provider' to start offering VPN services")
	}
	if isClient {
		fmt.Println("  1. Run 'bcvpn scan' to discover available VPN providers")
		fmt.Println("  2. Run 'bcvpn connect' to connect to a provider")
	}
	fmt.Println()
	fmt.Println("For more help, run 'bcvpn help' or 'bcvpn doctor' to verify your setup.")
}

func handleGenerateTLSKeypair() {
	key, err := btcec.NewPrivateKey()
	if err != nil {
		log.Fatalf("Failed to generate TLS keypair: %v", err)
	}
	privHex := hex.EncodeToString(key.Serialize())
	pubHex := hex.EncodeToString(key.PubKey().SerializeCompressed())
	fmt.Printf("Private Key (hex): %s\n", privHex)
	fmt.Printf("Public Key (hex): %s\n", pubHex)
}

func handleGenerateProviderKey(cfg *config.Config, dryRun bool) {
	keyPath := cfg.Provider.PrivateKeyFile
	if dryRun {
		fmt.Printf("[Dry Run] Would generate provider key at: %s\n", keyPath)
		fmt.Println("[Dry Run] Would prompt for password and generate a new encrypted key.")
		fmt.Println("[Dry Run] Would output the public key.")
		return
	}
	if _, err := os.Stat(keyPath); err == nil {
		fmt.Print("Provider key already exists. Overwrite? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		resp, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(resp)) != "y" {
			fmt.Println("Aborted.")
			os.Exit(0)
		}
	}
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter password for new provider key: ")
	pass1, _ := reader.ReadString('\n')
	fmt.Print("Confirm password: ")
	pass2, _ := reader.ReadString('\n')
	if strings.TrimSpace(pass1) != strings.TrimSpace(pass2) {
		log.Fatal("Passwords do not match")
	}
	password := []byte(strings.TrimSpace(pass1))
	if len(password) == 0 {
		log.Fatal("Password cannot be empty")
	}
	key, err := crypto.GenerateAndEncryptKey(keyPath, password)
	if err != nil {
		log.Fatalf("Failed to generate provider key: %v", err)
	}
	log.Printf("Provider key generated and saved to %s", keyPath)
	log.Printf("Public key: %s", hex.EncodeToString(key.PubKey().SerializeCompressed()))
}

func handleFavorite(cfg *config.Config, configPath string) {
	if len(os.Args) < 3 {
		log.Fatal("Usage: bcvpn favorite [add|remove] <pubkey> [comment]")
	}
	subcmd := strings.ToLower(os.Args[2])
	if subcmd != "add" && subcmd != "remove" {
		log.Fatalf("Unknown favorite command: %s (expected add/remove)", subcmd)
	}
	if len(os.Args) < 4 {
		log.Fatalf("Missing pubkey for %s", subcmd)
	}
	pubkey := strings.TrimSpace(os.Args[3])
	// comment ignored

	if _, err := hex.DecodeString(pubkey); err != nil {
		log.Fatalf("Invalid pubkey hex: %v", err)
	}

	favs := cfg.Client.FavoriteProviders
	switch subcmd {
	case "add":
		for _, existing := range favs {
			if existing == pubkey {
				log.Fatalf("Provider already in favorites")
			}
		}
		cfg.Client.FavoriteProviders = append(favs, pubkey)
	case "remove":
		newFavs := []string{}
		found := false
		for _, existing := range favs {
			if existing == pubkey {
				found = true
				continue
			}
			newFavs = append(newFavs, existing)
		}
		if !found {
			log.Fatalf("Provider not in favorites")
		}
		cfg.Client.FavoriteProviders = newFavs
	}

	if err := saveConfigFile(configPath, cfg); err != nil {
		log.Fatalf("Failed to save config: %v", err)
	}
	log.Printf("Favorites updated. Total: %d", len(cfg.Client.FavoriteProviders))
}

type ratingEntry struct {
	ProviderPubkey string `json:"provider_pubkey"`
	Score          int    `json:"score"`
	Comment        string `json:"comment,omitempty"`
	Timestamp      string `json:"timestamp"`
}

// sessionInfo stores information about the active/last VPN session for rating purposes.
type sessionInfo struct {
	ProviderPubkey   string `json:"provider_pubkey"`
	ProviderEndpoint string `json:"provider_endpoint"` // IP:Port
	ConnectedAt      string `json:"connected_at"`
	DisconnectedAt   string `json:"disconnected_at,omitempty"`
	Rated            bool   `json:"rated"`
}

func saveSessionInfo(pubkey string, endpoint string) error {
	dir, err := config.AppConfigDir()
	if err != nil {
		return err
	}
	info := sessionInfo{
		ProviderPubkey:   pubkey,
		ProviderEndpoint: endpoint,
		ConnectedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "session.json"), data, 0o644)
}

func loadSessionInfo() (*sessionInfo, error) {
	dir, err := config.AppConfigDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "session.json"))
	if err != nil {
		return nil, err
	}
	var info sessionInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func clearSessionInfo() error {
	dir, err := config.AppConfigDir()
	if err != nil {
		return err
	}
	return os.Remove(filepath.Join(dir, "session.json"))
}

func handleRate(cfg *config.Config, configPath string, broadcastOnChain bool) {
	if len(os.Args) < 4 {
		log.Fatal("Usage: bcvpn rate <provider-pubkey> <rating> [comment]")
	}
	pubkey := strings.TrimSpace(os.Args[2])
	ratingStr := strings.TrimSpace(os.Args[3])
	rating, err := strconv.Atoi(ratingStr)
	if err != nil || rating < 1 || rating > 5 {
		log.Fatal("Rating must be an integer between 1 and 5")
	}
	comment := ""
	if len(os.Args) > 4 {
		comment = strings.Join(os.Args[4:], " ")
	}

	if _, err := hex.DecodeString(pubkey); err != nil {
		log.Fatalf("Invalid pubkey hex: %v", err)
	}

	dir, err := config.AppConfigDir()
	if err != nil {
		log.Fatalf("Failed to get config dir: %v", err)
	}
	ratingsPath := filepath.Join(dir, "ratings.json")
	ratings := []ratingEntry{}
	if data, err := os.ReadFile(ratingsPath); err == nil {
		if err := json.Unmarshal(data, &ratings); err != nil {
			ratings = []ratingEntry{}
		}
	} else if !os.IsNotExist(err) {
		log.Fatalf("Failed to read ratings: %v", err)
	}

	// Update or add
	found := false
	for i, r := range ratings {
		if r.ProviderPubkey == pubkey {
			ratings[i].Score = rating
			ratings[i].Comment = comment
			ratings[i].Timestamp = time.Now().UTC().Format(time.RFC3339)
			found = true
			break
		}
	}
	if !found {
		ratings = append(ratings, ratingEntry{
			ProviderPubkey: pubkey,
			Score:          rating,
			Comment:        comment,
			Timestamp:      time.Now().UTC().Format(time.RFC3339),
		})
	}

	data, err := json.MarshalIndent(ratings, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal ratings: %v", err)
	}
	if err := os.WriteFile(ratingsPath, data, 0o644); err != nil {
		log.Fatalf("Failed to write ratings: %v", err)
	}
	fmt.Printf("Rating for %s: %d/5 saved.\n", pubkey, rating)

	// Broadcast on-chain if requested and RPC is configured
	if broadcastOnChain && cfg != nil && strings.TrimSpace(cfg.RPC.Host) != "" {
		// TODO: Implement full blockchain rating broadcast
		// Requires: RPC connection, client key management, signing infrastructure
		fmt.Println("Note: Blockchain rating broadcast requires additional integration (TODO)")
	}
}

// promptForRating asks the user to rate the last provider they connected to.
func promptForRating(cfg *config.Config) {
	dir, err := config.AppConfigDir()
	if err != nil {
		log.Printf("Warning: could not get config dir: %v", err)
		return
	}

	session, err := loadSessionInfo()
	if err != nil {
		if os.IsNotExist(err) {
			return // No session info, skip rating
		}
		log.Printf("Warning: could not load session info: %v", err)
		return
	}

	if session.Rated {
		return // Already rated
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println()
	fmt.Printf("How would you rate your connection to provider %s? (1-5 stars, or press Enter to skip)\n", session.ProviderPubkey[:16]+"...")
	fmt.Print("Rating (1-5): ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		fmt.Println("Rating skipped.")
		return
	}

	rating, err := strconv.Atoi(input)
	if err != nil || rating < 1 || rating > 5 {
		fmt.Println("Invalid rating. Rating skipped.")
		return
	}

	// Save rating locally
	ratingsPath := filepath.Join(dir, "ratings.json")
	ratings := []ratingEntry{}
	if data, err := os.ReadFile(ratingsPath); err == nil {
		json.Unmarshal(data, &ratings)
	} else if !os.IsNotExist(err) {
		log.Printf("Warning: could not read ratings file: %v", err)
	}

	ratings = append(ratings, ratingEntry{
		ProviderPubkey: session.ProviderPubkey,
		Score:          rating,
		Comment:        "",
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	})

	data, _ := json.MarshalIndent(ratings, "", "  ")
	os.WriteFile(ratingsPath, data, 0o644)
	fmt.Printf("Rating saved: %d/5 stars for provider %s\n", rating, session.ProviderPubkey[:16]+"...")

	// Broadcast on blockchain if RPC is configured
	if cfg != nil && strings.TrimSpace(cfg.RPC.Host) != "" {
		fmt.Println("Broadcasting rating to blockchain...")
		if err := broadcastRatingOnChain(cfg, session.ProviderPubkey, uint8(rating)); err != nil {
			fmt.Printf("Warning: could not broadcast rating: %v\n", err)
			fmt.Println("Rating saved locally but not on blockchain.")
		} else {
			fmt.Println("Rating broadcast to blockchain!")
		}
	} else {
		fmt.Println("Note: RPC not configured. Rating saved locally only.")
		fmt.Println("Configure RPC in config to broadcast ratings on blockchain.")
	}

	// Mark session as rated
	session.Rated = true
	session.DisconnectedAt = time.Now().UTC().Format(time.RFC3339)
	sessionData, _ := json.MarshalIndent(session, "", "  ")
	os.WriteFile(filepath.Join(dir, "session.json"), sessionData, 0o644)
}

func broadcastRatingOnChain(cfg *config.Config, providerPubkeyHex string, rating uint8) error {
	providerPubKeyBytes, err := hex.DecodeString(providerPubkeyHex)
	if err != nil {
		return fmt.Errorf("invalid provider pubkey: %w", err)
	}
	providerPubKey, err := btcec.ParsePubKey(providerPubKeyBytes)
	if err != nil {
		return fmt.Errorf("invalid provider pubkey: %w", err)
	}

	client := connectRPCWithConfig(cfg)
	defer client.Shutdown()

	addressType := cfg.Client.AddressType
	if strings.ToLower(addressType) == "auto" || addressType == "" {
		detected, err := blockchain.DetectAddressType(client)
		if err != nil {
			addressType = "p2pkh"
		} else {
			addressType = detected
		}
	}

	clientKey, err := getOrCreateClientKey(cfg)
	if err != nil {
		return fmt.Errorf("failed to get client key: %w", err)
	}

	feeCfg := blockchain.NewFeeConfig(cfg.RPC.MinRelayFee, cfg.RPC.DefaultFeeKb)

	return blockchain.AnnounceRating(client, providerPubKey, clientKey, rating, "bcvpn-client", 6, "conservative", addressType, feeCfg)
}

func getOrCreateClientKey(cfg *config.Config) (*btcec.PrivateKey, error) {
	dir, err := config.AppConfigDir()
	if err != nil {
		return nil, err
	}
	keyPath := filepath.Join(dir, "client.key")

	// Try to load existing key
	if _, err := os.Stat(keyPath); err == nil {
		// Key exists, prompt for password or use env
		password := os.Getenv("BCVPN_KEY_PASSWORD")
		if password == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter password to decrypt client key: ")
			pass, _ := reader.ReadString('\n')
			password = strings.TrimSpace(pass)
		}
		return crypto.LoadAndDecryptKey(keyPath, []byte(password))
	}

	// Generate new key
	fmt.Println("No client key found. Generating new key...")
	key, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter password to encrypt new client key: ")
	pass1, _ := reader.ReadString('\n')
	fmt.Print("Confirm password: ")
	pass2, _ := reader.ReadString('\n')
	if strings.TrimSpace(pass1) != strings.TrimSpace(pass2) {
		return nil, fmt.Errorf("passwords do not match")
	}
	password := strings.TrimSpace(pass1)

	privKeyBytes := key.Serialize()
	encrypted, err := crypto.Encrypt(privKeyBytes, []byte(password))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt key: %w", err)
	}
	if err := os.WriteFile(keyPath, encrypted, 0o600); err != nil {
		return nil, fmt.Errorf("failed to save key: %w", err)
	}
	fmt.Println("Client key generated and saved.")
	return key, nil
}

func handleStopProvider(cfg *config.Config, configPath string) {
	pidPath, err := config.ResolveProviderPIDFilePath(cfg, configPath)
	if err != nil {
		log.Fatalf("Failed to resolve PID file path: %v", err)
	}

	pid, err := readPIDFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No provider is currently running.")
			fmt.Println()
			fmt.Println("To start a provider, run: bcvpn start-provider")
			os.Exit(1)
		}
		log.Fatalf("Failed to read PID file: %v", err)
	}

	if !isProcessRunning(pid) {
		fmt.Printf("Provider process (PID %d) is not running. The PID file may be stale.\n", pid)
		fmt.Println("To start a provider, run: bcvpn start-provider")
		os.Exit(1)
	}

	log.Printf("Stopping provider (PID %d)...", pid)
	if err := stopProviderProcess(pid, true); err != nil {
		log.Fatalf("Failed to stop provider: %v", err)
	}

	// Remove PID file
	os.Remove(pidPath)
	log.Println("Provider stopped successfully.")
}

func handleRestartProvider(cfg *config.Config, configPath string) {
	// Stop the running provider
	handleStopProvider(cfg, configPath)

	// For now, we'll just instruct the user to run start-provider manually.
	// A more seamless approach would be to re-execute start-provider after stopping,
	// but that would block and make this command block as well. That's acceptable but
	// requires more careful handling of signals and cleanup.
	log.Println("Provider stopped. To start it again, run: bcvpn start-provider")
}

func handleVersion(jsonMode bool) {
	if !jsonMode {
		fmt.Println(version.String())
		return
	}
	payload := map[string]string{
		"version": version.Version,
		"commit":  version.GitCommit,
		"built":   version.BuildDate,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		log.Fatalf("failed to encode version JSON: %v", err)
	}
}

func handleAbout(jsonMode bool) {
	about := `BlockchainVPN - Decentralized VPN Marketplace

Version: ` + version.String() + `
Built: ` + version.BuildDate + `

A peer-to-peer VPN marketplace built on the OrdexCoin blockchain.

## Support Development

This project is done in my spare time and will be available for free.
Your donations help me finish and polish it to a working, stable state.

Donation Addresses:
  BTC:  bc1qkmzc6d49fl0edyeynezwlrfqv486nmk6p5pmta
  ETH:  bc1qkmzc6d49fl0edyeynezwlrfqv486nmk6p5pmta
  LTC:  bc1qkmzc6d49fl0edyeynezwlrfqv486nmk6p5pmta
  SOL:  HB2o6q6vsW5796U5y7NxNqA7vYZW1vuQjpAHDo7FAMG8
  XRP:  rUW7Q64vR4PwDM3F27etd6ipxK8MtuxsFs
  OXC:  oxc1q3psft0hvlslddyp8ktr3s737req7q8hrl0rkly
  OXG:  oxg1q34apjkn2yc6rsvuua98432ctqdrjh9hdkhpx0t

For more information, visit the project at:
https://github.com/anomalyco/blockchain-vpn
`

	if jsonMode {
		payload := map[string]interface{}{
			"name":        "BlockchainVPN",
			"version":     version.Version,
			"commit":      version.GitCommit,
			"built":       version.BuildDate,
			"description": "Decentralized VPN marketplace on OrdexCoin",
			"donations": map[string]string{
				"BTC": "bc1qkmzc6d49fl0edyeynezwlrfqv486nmk6p5pmta",
				"ETH": "bc1qkmzc6d49fl0edyeynezwlrfqv486nmk6p5pmta",
				"LTC": "bc1qkmzc6d49fl0edyeynezwlrfqv486nmk6p5pmta",
				"SOL": "HB2o6q6vsW5796U5y7NxNqA7vYZW1vuQjpAHDo7FAMG8",
				"XRP": "rUW7Q64vR4PwDM3F27etd6ipxK8MtuxsFs",
				"OXC": "oxc1q3psft0hvlslddyp8ktr3s737req7q8hrl0rkly",
				"OXG": "oxg1q34apjkn2yc6rsvuua98432ctqdrjh9hdkhpx0t",
			},
			"url": "https://github.com/anomalyco/blockchain-vpn",
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(payload); err != nil {
			log.Fatalf("failed to encode about JSON: %v", err)
		}
		return
	}
	fmt.Print(about)
}

// connectRPCWithConfig creates an RPC client using the provided configuration.
// It supports cookie-based authentication by reading the .cookie file if present.
// It also handles server warmup by checking getrpcinfo before first use.
func connectRPCWithConfig(cfg *config.Config) *rpcclient.Client {
	// Determine RPC host from config or network defaults
	rpcHost := cfg.RPC.Host
	if strings.TrimSpace(rpcHost) == "" {
		rpcHost = getDefaultRPCHost(cfg.RPC.Network)
	}

	// Determine credentials: prefer cookie file if available, fall back to User/Pass
	user, pass := cfg.RPC.User, cfg.RPC.Pass
	if cookieUser, cookiePass, err := readRPCCookie(cfg); err == nil && cookieUser != "" {
		log.Printf("Using RPC cookie authentication (user: %s)", cookieUser)
		user, pass = cookieUser, cookiePass
	} else if err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: failed to read RPC cookie: %v (using configured credentials)", err)
	}

	connCfg := &rpcclient.ConnConfig{
		Host:         rpcHost,
		User:         user,
		Pass:         pass,
		HTTPPostMode: true,
		DisableTLS:   !cfg.RPC.EnableTLS,
	}
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatalf("Failed to connect to RPC server at %s: %v\nHint: Ensure ordexcoind is running and the RPC credentials are correct.", connCfg.Host, err)
	}
	if client == nil {
		log.Fatalf("Failed to connect to RPC server at %s: unexpected nil client", connCfg.Host)
	}

	// Check server warmup status if needed
	if err := waitForServerReady(context.Background(), client); err != nil {
		log.Fatalf("RPC server not ready at %s: %v\nHint: The node may still be warming up. Wait a moment and try again, or check ordexcoind logs.", connCfg.Host, err)
	}

	return client
}

// getDefaultRPCHost returns the default RPC host:port for the given network.
func getDefaultRPCHost(network string) string {
	network = strings.ToLower(strings.TrimSpace(network))
	switch network {
	case "testnet", "testnet3":
		return "localhost:35173"
	case "signet":
		return "localhost:325173"
	case "regtest", "regression":
		return "localhost:18443"
	default: // mainnet, main, or unknown
		return "localhost:25173" // Ordexcoin default mainnet RPC port
	}
}

// readRPCCookie attempts to read the RPC cookie file and extract credentials.
// It returns username and password (or empty string if cookie not found/invalid).
func readRPCCookie(cfg *config.Config) (string, string, error) {
	cookiePath := strings.TrimSpace(cfg.RPC.CookieFile)
	if cookiePath == "" {
		home, _ := os.UserHomeDir()
		cookieDir := strings.TrimSpace(cfg.RPC.CookieDir)
		if cookieDir == "" {
			cookieDir = ".ordexcoin"
		}
		cookieDirRegTx := strings.TrimSpace(cfg.RPC.CookieDirRegTx)
		cookieDirTest3 := strings.TrimSpace(cfg.RPC.CookieDirTest3)
		cookieDirSignet := strings.TrimSpace(cfg.RPC.CookieDirSignet)

		network := strings.ToLower(strings.TrimSpace(cfg.RPC.Network))
		var subDir string
		switch network {
		case "regtest", "regression":
			subDir = cookieDirRegTx
		case "testnet", "testnet3", "testnet4":
			subDir = cookieDirTest3
		case "signet":
			subDir = cookieDirSignet
		}

		commonPaths := []string{
			filepath.Join(home, cookieDir, ".cookie"),
		}
		if subDir != "" {
			commonPaths = append(commonPaths, filepath.Join(home, cookieDir, subDir, ".cookie"))
		}
		commonPaths = append(commonPaths,
			"/var/lib/"+strings.Split(cookieDir, "/")[0]+"/.cookie",
			"C:\\ProgramData\\OrdExcoin\\.cookie",
		)
		for _, p := range commonPaths {
			if _, err := os.Stat(p); err == nil {
				cookiePath = p
				break
			}
		}
		if cookiePath == "" {
			return "", "", os.ErrNotExist
		}
	}

	data, err := os.ReadFile(cookiePath)
	if err != nil {
		return "", "", err
	}
	// Cookie format: username:hex(sessionid)
	line := strings.TrimSpace(string(data))
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid cookie format")
	}
	username := parts[0]
	// Password is empty for cookie auth; the session ID is not used as password in standard Bitcoin Core RPC
	password := ""
	return username, password, nil
}

// waitForServerReady polls the RPC server until it's out of warmup or timeout.
func waitForServerReady(ctx context.Context, client *rpcclient.Client) error {
	// Try getrpcinfo to check warmup status
	timeout := 30 * time.Second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	deadline := time.Now().Add(timeout)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for RPC server to be ready")
			}
			// Use client.RawRequest to call getrpcinfo
			_, err := client.RawRequest("getrpcinfo", nil)
			if err == nil {
				// Server responded, assume it's ready
				return nil
			}
			// Check if error indicates warmup or not ready
			if strings.Contains(err.Error(), "warmup") || strings.Contains(err.Error(), "not ready") {
				continue // keep waiting
			}
			// Connection errors: keep trying
			if strings.Contains(err.Error(), "connection refused") ||
				strings.Contains(err.Error(), "no route") ||
				strings.Contains(err.Error(), "dial tcp") {
				continue
			}
			// Unknown error; assume not ready yet and keep trying
		}
	}
}

func readPassword(prompt string) ([]byte, error) {
	fmt.Print(prompt)
	pass, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	return pass, err
}

func getProviderKey(cfg *config.Config, passwordEnv string) (*btcec.PrivateKey, error) {
	keyPath := cfg.Provider.PrivateKeyFile
	resolvedMode, err := crypto.ResolveKeyStorageMode(cfg.Security.KeyStorageMode)
	if err != nil {
		return nil, err
	}
	if resolvedMode != "file" {
		key, err := crypto.LoadOrCreateProviderKey(keyPath, nil, cfg.Security.KeyStorageMode, cfg.Security.KeyStorageService)
		if err != nil {
			return nil, err
		}
		log.Printf("Provider key loaded via secure storage backend (%s).", resolvedMode)
		return key, nil
	}

	passwordFromEnv := []byte{}
	if name := strings.TrimSpace(passwordEnv); name != "" {
		value := strings.TrimSpace(os.Getenv(name))
		if value == "" {
			return nil, fmt.Errorf("env var %s is empty or not set", name)
		}
		passwordFromEnv = []byte(value)
	}
	if _, err := os.Stat(keyPath); err == nil {
		password := passwordFromEnv
		if len(password) == 0 {
			pass, err := readPassword("Enter password to decrypt provider key: ")
			if err != nil {
				return nil, fmt.Errorf("failed to read password: %w", err)
			}
			password = []byte(strings.TrimSpace(string(pass)))
		}
		key, err := crypto.LoadAndDecryptKey(keyPath, password)
		if err != nil {
			return nil, fmt.Errorf("failed to load and decrypt provider key (wrong password?): %w", err)
		}
		log.Println("Provider key successfully decrypted.")
		return key, nil
	}

	fmt.Println("No provider key found. Let's create a new encrypted key.")
	password := passwordFromEnv
	if len(password) == 0 {
		pass1, err := readPassword("Enter new password for provider key: ")
		if err != nil {
			return nil, fmt.Errorf("failed to read password: %w", err)
		}
		pass2, err := readPassword("Confirm new password: ")
		if err != nil {
			return nil, fmt.Errorf("failed to read password: %w", err)
		}
		if strings.TrimSpace(string(pass1)) != strings.TrimSpace(string(pass2)) {
			return nil, fmt.Errorf("passwords do not match")
		}
		password = []byte(strings.TrimSpace(string(pass1)))
	}
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

func rotateProviderKey(cfg *config.Config, keyPath, oldPasswordEnv, newPasswordEnv string) error {
	resolvedMode, err := crypto.ResolveKeyStorageMode(cfg.Security.KeyStorageMode)
	if err != nil {
		return err
	}
	if resolvedMode != "file" {
		if err := crypto.RotateProviderKey(keyPath, nil, nil, cfg.Security.KeyStorageMode, cfg.Security.KeyStorageService); err != nil {
			return err
		}
		log.Printf("Provider key rotated in secure storage backend (%s).", resolvedMode)
		return nil
	}

	if _, err := os.Stat(keyPath); err != nil {
		return fmt.Errorf("provider key file not found at %s", keyPath)
	}

	reader := bufio.NewReader(os.Stdin)
	oldPassword := []byte(strings.TrimSpace(os.Getenv(strings.TrimSpace(oldPasswordEnv))))
	if len(oldPassword) == 0 {
		fmt.Print("Enter current password to decrypt provider key: ")
		oldPass, _ := reader.ReadString('\n')
		oldPassword = []byte(strings.TrimSpace(oldPass))
	}
	if len(oldPassword) == 0 {
		return fmt.Errorf("old password cannot be empty")
	}

	newPassword := strings.TrimSpace(os.Getenv(strings.TrimSpace(newPasswordEnv)))
	if newPassword == "" {
		fmt.Print("Enter new password for rotated provider key: ")
		pass1, _ := reader.ReadString('\n')
		fmt.Print("Confirm new password: ")
		pass2, _ := reader.ReadString('\n')
		newPassword = strings.TrimSpace(pass1)
		if newPassword != strings.TrimSpace(pass2) {
			return fmt.Errorf("new passwords do not match")
		}
	}
	if strings.TrimSpace(newPassword) == "" {
		return fmt.Errorf("new password cannot be empty")
	}

	if err := crypto.RotateProviderKey(keyPath, oldPassword, []byte(newPassword), cfg.Security.KeyStorageMode, cfg.Security.KeyStorageService); err != nil {
		return err
	}

	absKey := keyPath
	if a, err := filepath.Abs(keyPath); err == nil {
		absKey = a
	}
	log.Printf("Old provider key backed up to timestamped file near %s", absKey)
	return nil
}

func resolveConfigPath(defaultPath string) string {
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath
	}

	if _, err := os.Stat("config.json"); err == nil {
		if err := migrateLegacyLocalFiles(defaultPath); err != nil {
			log.Printf("Warning: failed to migrate local config files to app config dir: %v", err)
			return "config.json"
		}
		return defaultPath
	}

	home, _ := os.UserHomeDir()
	legacyPaths := []string{
		filepath.Join(home, ".ordexcoin", "blockchain-vpn", "config.json"),
		filepath.Join(home, ".ordexcoin", "BlockchainVPN", "config.json"),
		filepath.Join(home, "BlockchainVPN", "config.json"),
	}
	for _, legacyPath := range legacyPaths {
		if _, err := os.Stat(legacyPath); err == nil {
			log.Printf("Note: Found config at legacy location %q, consider moving to %q", legacyPath, defaultPath)
			return legacyPath
		}
	}

	return defaultPath
}

func migrateLegacyLocalFiles(defaultConfigPath string) error {
	if err := copyFile("config.json", defaultConfigPath); err != nil {
		return err
	}
	if err := os.Remove("config.json"); err != nil {
		log.Printf("Warning: migrated config copied but old local file could not be removed: %v", err)
	}

	defaultKeyPath, err := config.ResolveDefaultProviderKeyPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat("provider.key"); err == nil {
		if _, err := os.Stat(defaultKeyPath); os.IsNotExist(err) {
			if err := copyFile("provider.key", defaultKeyPath); err != nil {
				return err
			}
			if err := os.Remove("provider.key"); err != nil {
				log.Printf("Warning: migrated provider key copied but old local file could not be removed: %v", err)
			}
		}
	}

	log.Printf("Migrated local config files to %s", filepath.Dir(defaultConfigPath))
	return nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
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

func buildProviderEndpoint(providerCfg *config.ProviderConfig, announceIP net.IP, announcePort int, providerKey *btcec.PrivateKey, measuredBandwidthKB uint32) *protocol.VPNEndpoint {
	var bandwidthKB uint32
	if providerCfg.BandwidthAutoTest && measuredBandwidthKB > 0 {
		bandwidthKB = measuredBandwidthKB
	} else {
		bandwidthKB = parseBandwidthLimitToKbps(providerCfg.BandwidthLimit)
	}
	maxConsumers := uint16(0)
	if providerCfg.MaxConsumers > 0 && providerCfg.MaxConsumers <= 65535 {
		maxConsumers = uint16(providerCfg.MaxConsumers)
	}

	// Determine country code - use config or auto-detect
	countryCode := strings.ToUpper(strings.TrimSpace(providerCfg.Country))
	if countryCode == "" {
		if loc, err := geoip.AutoLocate(); err == nil {
			countryCode = loc.CountryCode
			log.Printf("Auto-detected country: %s", countryCode)
		} else {
			log.Printf("Warning: Could not auto-detect country: %v", err)
			countryCode = "ZZ" // Unknown
		}
	}

	// Determine pricing method and units
	var pricingMethod uint8 = protocol.PricingMethodSession
	timeUnitSecs := uint32(0)
	dataUnitBytes := uint32(0)

	switch strings.ToLower(strings.TrimSpace(providerCfg.PricingMethod)) {
	case "time":
		pricingMethod = protocol.PricingMethodTime
		// Parse time unit (minute, hour) to seconds
		switch strings.ToLower(strings.TrimSpace(providerCfg.BillingTimeUnit)) {
		case "minute":
			timeUnitSecs = 60
		case "hour":
			timeUnitSecs = 3600
		default:
			timeUnitSecs = 60 // default to per-minute
		}
	case "data":
		pricingMethod = protocol.PricingMethodData
		// Parse data unit (MB, GB) to bytes
		switch strings.ToLower(strings.TrimSpace(providerCfg.BillingDataUnit)) {
		case "mb":
			dataUnitBytes = 1_000_000
		case "gb":
			dataUnitBytes = 1_000_000_000
		default:
			dataUnitBytes = 1_000_000 // default to per-MB
		}
	default: // session
		pricingMethod = protocol.PricingMethodSession
	}

	return &protocol.VPNEndpoint{
		IP:                    announceIP,
		Port:                  uint16(announcePort),
		Price:                 providerCfg.Price,
		PublicKey:             providerKey.PubKey(),
		AdvertisedBandwidthKB: bandwidthKB,
		MaxConsumers:          maxConsumers,
		CountryCode:           countryCode,
		AvailabilityFlags:     protocol.AvailabilityFlagAvailable,
		PricingMethod:         pricingMethod,
		TimeUnitSecs:          timeUnitSecs,
		DataUnitBytes:         dataUnitBytes,
		SessionTimeoutSecs:    uint32(providerCfg.MaxSessionDurationSecs),
	}
}

func parseBandwidthLimitToKbps(v string) uint32 {
	s := strings.ToLower(strings.TrimSpace(v))
	if s == "" || s == "0" || s == "0mbit" || s == "unlimited" {
		return 0
	}
	mult := float64(1)
	switch {
	case strings.HasSuffix(s, "gbit"):
		s = strings.TrimSuffix(s, "gbit")
		mult = 1000 * 1000
	case strings.HasSuffix(s, "mbit"):
		s = strings.TrimSuffix(s, "mbit")
		mult = 1000
	case strings.HasSuffix(s, "kbit"):
		s = strings.TrimSuffix(s, "kbit")
		mult = 1
	default:
		return 0
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || n <= 0 {
		return 0
	}
	kbps := uint32(n * mult)
	return kbps
}

func interactiveConnect(ctx context.Context, client *rpcclient.Client, feeCfg blockchain.FeeConfig, addressType string, endpoints []*geoip.EnrichedVPNEndpoint, clientCfg *config.ClientConfig, secCfg *config.SecurityConfig, dryRun bool, autoReconnect bool, autoReconnectMaxAttempts int, autoReconnectInterval string, autoReconnectMaxInterval string) {
	// Write client PID file for management (stop/disconnect)
	dir, err := config.AppConfigDir()
	if err != nil {
		log.Printf("Warning: failed to get app config dir: %v", err)
	} else {
		pidPath := filepath.Join(dir, "client.pid")
		if err := writePIDFile(pidPath); err != nil {
			log.Printf("Warning: failed to write client PID file: %v", err)
		} else {
			log.Printf("Client PID file written to %s (PID: %d)", pidPath, os.Getpid())
			defer os.Remove(pidPath)
		}
	}

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
	providerAddr, err := blockchain.GetProviderPaymentAddress(client, selectedEndpoint.TxID)
	if err != nil {
		log.Fatalf("Could not get provider payment address: %v", err)
	}
	fmt.Printf("Provider's payment address: %s\n", providerAddr)
	fmt.Printf("Payment required: %d satoshis\n", selectedEndpoint.Endpoint.Price)

	// Initialize spending manager if limits are enabled
	var spendingMgr *tunnel.SpendingManager
	if clientCfg.SpendingLimitEnabled || clientCfg.AutoRechargeEnabled {
		spendingMgr = tunnel.NewSpendingManager(clientCfg, client, providerAddr, localKey, selectedEndpoint.Endpoint.PublicKey, addressType, feeCfg)
		// Check if payment would exceed limits
		paymentAmount := selectedEndpoint.Endpoint.Price
		if err := spendingMgr.RecordPayment(paymentAmount); err != nil {
			log.Fatalf("Spending limit would be exceeded: %v", err)
		}
	}

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
		verifiedAmount, err := blockchain.GetPaymentVerification(selectedEndpoint.Endpoint.Price, selectedEndpoint.Endpoint.Price)
		if err != nil {
			log.Fatalf("Payment verification failed: %v", err)
		}
		fmt.Printf("Verified payment amount: %d sats (advertised: %d sats)\n", verifiedAmount, selectedEndpoint.Endpoint.Price)

		if err := tunnel.EnsureElevatedPrivileges(); err != nil {
			log.Fatalf("Cannot proceed: automatic networking privileges are required before payment: %v", err)
		}
		fmt.Printf("Sending payment of %d sats to provider...\n", selectedEndpoint.Endpoint.Price)
		txHash, err := blockchain.SendPayment(client, providerAddr, selectedEndpoint.Endpoint.Price, localKey.PubKey(), addressType, feeCfg)
		if err != nil {
			log.Fatalf("Failed to send payment: %v", err)
		}
		fmt.Printf("Payment sent: %s\n", txHash)

		// Wait for payment to be confirmed (provider requires confirmations before authorizing)
		fmt.Printf("Waiting for payment confirmation...\n")
		waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		if _, err := blockchain.WaitForConfirmations(waitCtx, client, txHash, 1, 10*time.Second); err != nil {
			waitCancel()
			log.Fatalf("Payment not confirmed in time: %v", err)
		}
		waitCancel()
		fmt.Printf("Payment confirmed!\n")

		// Record successful payment in spending manager (already did pre-check, now finalize)
		spendingMgr.AddCredits(0) // trigger log if needed

		// Start spending manager for auto-recharge and limit monitoring
		if spendingMgr != nil {
			spendingMgr.Start(ctx)
			spendingMgr.SetSessionStart() // Capture spending baseline for session limits
			defer spendingMgr.Stop()
		}
	}

	peerPubKey := selectedEndpoint.Endpoint.PublicKey
	endpointAddr := fmt.Sprintf("%s:%d", selectedEndpoint.Endpoint.IP.String(), selectedEndpoint.Endpoint.Port)
	fmt.Printf("\nConnecting to %s...\n", endpointAddr)

	// Save session info for rating after disconnect
	if !dryRun {
		pubkeyHex := hex.EncodeToString(peerPubKey.SerializeCompressed())
		if err := saveSessionInfo(pubkeyHex, endpointAddr); err != nil {
			log.Printf("Warning: failed to save session info: %v", err)
		}
	}

	if dryRun {
		fmt.Printf("[Dry Run] Simulation: Would create TUN interface %s and connect to %s.\n", clientCfg.InterfaceName, endpointAddr)
	} else {
		mgr := tunnel.NewMultiTunnelManager()
		expected := tunnel.ClientSecurityExpectations{
			ExpectedCountry:     selectedEndpoint.Country,
			ExpectedBandwidthKB: selectedEndpoint.AdvertisedBandwidthKB,
			ThroughputProbePort: selectedEndpoint.Endpoint.ThroughputProbePort,
		}
		pricingParams := tunnel.NewPricingParamsFromEndpoint(selectedEndpoint.Endpoint)

		if autoReconnect {
			clientCfg.AutoReconnectEnabled = true
			clientCfg.AutoReconnectMaxAttempts = autoReconnectMaxAttempts
			clientCfg.AutoReconnectInterval = autoReconnectInterval
			clientCfg.AutoReconnectMaxInterval = autoReconnectMaxInterval

			if err := mgr.AddWithReconnect(
				"session-interactive",
				clientCfg.InterfaceName,
				clientCfg,
				secCfg,
				localKey,
				peerPubKey,
				endpointAddr,
				expected,
				spendingMgr,
				pricingParams,
			); err != nil {
				log.Fatalf("Failed to add tunnel with reconnect: %v", err)
			}
			fmt.Println("Auto-reconnect enabled. Will attempt to reconnect on disconnect.")
		} else {
			if err := mgr.Add(
				"session-interactive",
				clientCfg.InterfaceName,
				clientCfg,
				secCfg,
				localKey,
				peerPubKey,
				endpointAddr,
				expected,
				spendingMgr,
				pricingParams,
			); err != nil {
				log.Fatalf("Failed to add tunnel: %v", err)
			}
		}

		// Wait indefinitely until cancelled or the tunnel errors out (handled by mgr)
		fmt.Println("Press Ctrl+C to disconnect.")
		<-ctx.Done()
		fmt.Println("Shutting down tunnel...")

		// Run cleanup in goroutine with timeout to ensure process exits
		cleanupDone := make(chan struct{})
		go func() {
			mgr.CancelAll()
			close(cleanupDone)
		}()
		select {
		case <-cleanupDone:
			fmt.Println("Tunnel shutdown complete.")
		case <-time.After(10 * time.Second):
			log.Printf("Warning: tunnel cleanup timed out after 10s, forcing exit")
		}
	}
}

func detectChain(genesisHash *chainhash.Hash, network string) *chaincfg.Params {
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
		// Unknown chain - try to use configured network or fall back to mainnet
		chain := strings.ToLower(strings.TrimSpace(network))
		if chain == "" || chain == "auto" {
			chain = "mainnet"
		}
		switch chain {
		case "testnet", "testnet3", "testnet4":
			log.Printf("Warning: unknown genesis hash %s, using Testnet3 params (configured network: %s)", genesisHash.String(), network)
			return &chaincfg.TestNet3Params
		case "regtest", "regression":
			log.Printf("Warning: unknown genesis hash %s, using Regtest params (configured network: %s)", genesisHash.String(), network)
			return &chaincfg.RegressionNetParams
		case "simnet":
			log.Printf("Warning: unknown genesis hash %s, using Simnet params (configured network: %s)", genesisHash.String(), network)
			return &chaincfg.SimNetParams
		default:
			log.Printf("Warning: unknown genesis hash %s, using Mainnet params (configured network: %s)", genesisHash.String(), network)
			return &chaincfg.MainNetParams
		}
	}
}

// paramsFromNetwork returns chaincfg params for a given network name.
func paramsFromNetwork(network string) *chaincfg.Params {
	switch strings.ToLower(strings.TrimSpace(network)) {
	case "testnet", "testnet3", "testnet4":
		return &chaincfg.TestNet3Params
	case "regtest", "regression":
		return &chaincfg.RegressionNetParams
	case "simnet":
		return &chaincfg.SimNetParams
	default:
		return &chaincfg.MainNetParams
	}
}

func sortEndpoints(endpoints []*geoip.EnrichedVPNEndpoint, sortBy string) {
	switch strings.ToLower(sortBy) {
	case "price":
		sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].Endpoint.Price < endpoints[j].Endpoint.Price })
		fmt.Println("Sorted by price (lowest first).")
	case "country":
		sort.Slice(endpoints, func(i, j int) bool { return effectiveCountry(endpoints[i]) < effectiveCountry(endpoints[j]) })
		fmt.Println("Sorted by country.")
	case "bw":
		fallthrough
	case "bandwidth":
		sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].AdvertisedBandwidthKB > endpoints[j].AdvertisedBandwidthKB })
		fmt.Println("Sorted by bandwidth (highest first).")
	case "capacity":
		sort.Slice(endpoints, func(i, j int) bool {
			return effectiveCapacitySlots(endpoints[i]) > effectiveCapacitySlots(endpoints[j])
		})
		fmt.Println("Sorted by capacity (highest first).")
	case "score":
		sort.Slice(endpoints, func(i, j int) bool { return computeProviderScore(endpoints[i]) > computeProviderScore(endpoints[j]) })
		fmt.Println("Sorted by score (highest first).")
	case "latency":
		fallthrough
	default:
		sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].Latency < endpoints[j].Latency })
		fmt.Println("Sorted by latency (lowest first).")
	}
}

func filterEndpoints(endpoints []*geoip.EnrichedVPNEndpoint, country string, maxPrice uint64, minBandwidthKB uint32, maxLatency time.Duration, minSlots int, pricingMethod string, minScore float64) []*geoip.EnrichedVPNEndpoint {
	var filtered []*geoip.EnrichedVPNEndpoint
	for _, ep := range endpoints {
		if country != "" && !strings.EqualFold(country, effectiveCountry(ep)) {
			continue
		}
		if maxPrice > 0 && ep.Endpoint.Price > maxPrice {
			continue
		}
		if pricingMethod != "" {
			epMethod := getPricingMethodString(ep.Endpoint)
			if !strings.EqualFold(pricingMethod, epMethod) {
				continue
			}
		}
		if minBandwidthKB > 0 && ep.AdvertisedBandwidthKB < minBandwidthKB {
			continue
		}
		if maxLatency > 0 && ep.Latency > maxLatency {
			continue
		}
		if minSlots > 0 && effectiveCapacitySlots(ep) < minSlots {
			continue
		}
		if minScore > 0 && computeProviderScore(ep) < minScore {
			continue
		}
		filtered = append(filtered, ep)
	}
	return filtered
}

// getPricingMethodString returns the pricing method as a string from the endpoint.
func getPricingMethodString(ep *protocol.VPNEndpoint) string {
	switch ep.PricingMethod {
	case protocol.PricingMethodTime:
		return "time"
	case protocol.PricingMethodData:
		return "data"
	default:
		return "session"
	}
}

func effectiveCountry(ep *geoip.EnrichedVPNEndpoint) string {
	if ep == nil {
		return "N/A"
	}
	if v := strings.ToUpper(strings.TrimSpace(ep.DeclaredCountry)); v != "" {
		return v
	}
	if v := strings.ToUpper(strings.TrimSpace(ep.Country)); v != "" {
		return v
	}
	return "N/A"
}

func effectiveCapacitySlots(ep *geoip.EnrichedVPNEndpoint) int {
	if ep == nil {
		return 0
	}
	if ep.MaxConsumers == 0 {
		return 1 << 30 // Treat 0 as unlimited for filtering/sorting semantics.
	}
	return int(ep.MaxConsumers)
}

func displayCapacity(ep *geoip.EnrichedVPNEndpoint) string {
	if ep == nil || ep.MaxConsumers == 0 {
		return "unlimited"
	}
	return fmt.Sprintf("%d", ep.MaxConsumers)
}

func computeProviderScore(ep *geoip.EnrichedVPNEndpoint) float64 {
	if ep == nil || ep.Endpoint == nil {
		return 0
	}
	latencyMS := ep.Latency.Milliseconds()
	if latencyMS <= 0 {
		latencyMS = 1
	}
	price := float64(ep.Endpoint.Price)
	if price <= 0 {
		price = 1
	}
	bandwidth := float64(ep.AdvertisedBandwidthKB)
	capacity := float64(effectiveCapacitySlots(ep))
	if capacity > 1e6 {
		capacity = 1000 // clamp unlimited sentinel for scoring
	}
	countryBoost := 1.0
	if strings.TrimSpace(ep.DeclaredCountry) != "" {
		countryBoost = 1.05
	}

	repBoost := 1.0
	if ep.ReputationScore != 0 {
		repBoost = 0.1 + (float64(ep.ReputationScore) / 100.0 * 1.9)
	}

	return countryBoost * repBoost * ((bandwidth / 1000.0) + capacity) / (price * float64(latencyMS))
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

func handleHistory(cfg *config.Config, sinceLast bool, from, to string, jsonMode, tableMode bool) {
	if sinceLast {
		handleHistorySinceLast(cfg)
		return
	}

	if from != "" || to != "" {
		handleHistoryDateRange(cfg, from, to, jsonMode, tableMode)
		return
	}

	if jsonMode {
		handleHistoryJSON()
		return
	}

	handleFullHistory()
}

func handleHistoryDateRange(cfg *config.Config, from, to string, jsonMode, tableMode bool) {
	var fromTime, toTime time.Time
	var err error

	if from != "" {
		fromTime, err = time.Parse(time.RFC3339, from)
		if err != nil {
			log.Fatalf("Invalid --from date format. Use RFC3339 format (e.g., 2024-01-01T00:00:00Z): %v", err)
		}
	}
	if to != "" {
		toTime, err = time.Parse(time.RFC3339, to)
		if err != nil {
			log.Fatalf("Invalid --to date format. Use RFC3339 format (e.g., 2024-12-31T23:59:59Z): %v", err)
		}
	}

	records, err := history.LoadHistory()
	if err != nil {
		log.Fatalf("Failed to load history: %v", err)
	}

	var filtered []history.PaymentRecord
	for _, r := range records {
		if from != "" && r.Timestamp.Before(fromTime) {
			continue
		}
		if to != "" && r.Timestamp.After(toTime) {
			continue
		}
		filtered = append(filtered, r)
	}

	if jsonMode {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(filtered)
		return
	}

	if len(filtered) == 0 {
		fmt.Println("No payment history found in the specified date range.")
		return
	}

	fmt.Printf("%-25s %-15s %-40s %s\n", "Timestamp", "Amount (sats)", "Provider", "TxID")
	fmt.Println(strings.Repeat("-", 120))
	for _, r := range filtered {
		fmt.Printf("%-25s %-15d %-40s %s\n", r.Timestamp.Format("2006-01-02 15:04:05"), r.Amount, r.Provider, r.TxID)
	}
}

func handleHistoryJSON() {
	records, err := history.LoadHistory()
	if err != nil {
		log.Fatalf("Failed to load history: %v", err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(records)
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

	client := connectRPCWithConfig(cfg)
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

func handleConfigSubcommand(configPath string, cfg *config.Config, args []string, jsonMode bool) {
	if len(args) == 0 {
		fmt.Println("Usage:")
		fmt.Println("  bcvpn config get [key]")
		fmt.Println("  bcvpn config set <key> <value>")
		fmt.Println("  bcvpn config validate")
		fmt.Println("  bcvpn config export <path>")
		fmt.Println("  bcvpn config import <path> [--validate]")
		return
	}

	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "get":
		if len(args) < 2 {
			printConfigValue("", cfg, jsonMode)
			return
		}
		key := strings.TrimSpace(args[1])
		v, err := getConfigField(cfg, key)
		if err != nil {
			log.Fatalf("config get failed: %v", err)
		}
		printConfigValue(key, v, jsonMode)

	case "set":
		if len(args) < 3 {
			log.Fatal("Usage: bcvpn config set <key> <value>")
		}
		key := strings.TrimSpace(args[1])
		value := strings.Join(args[2:], " ")
		if err := setConfigField(cfg, key, value); err != nil {
			log.Fatalf("config set failed: %v", err)
		}
		if err := config.Validate(cfg); err != nil {
			log.Fatalf("config became invalid after set: %v", err)
		}
		if err := saveConfigFile(configPath, cfg); err != nil {
			log.Fatalf("failed to write config: %v", err)
		}
		log.Printf("Updated %s", key)

	case "validate":
		if err := config.Validate(cfg); err != nil {
			log.Fatalf("Config invalid: %v", err)
		}
		if jsonMode {
			printConfigValue("valid", true, true)
			return
		}
		fmt.Println("Config is valid.")
	case "export":
		if len(args) < 2 {
			log.Fatal("Usage: bcvpn config export <path>")
		}
		dst := strings.TrimSpace(args[1])
		if dst == "" {
			log.Fatal("config export path must not be empty")
		}
		if err := saveConfigFile(dst, cfg); err != nil {
			log.Fatalf("config export failed: %v", err)
		}
		log.Printf("Exported config to %s", dst)
	case "import":
		if len(args) < 2 {
			log.Fatal("Usage: bcvpn config import <path> [--validate]")
		}
		src := strings.TrimSpace(args[1])
		imported, err := config.LoadConfig(src)
		if err != nil {
			log.Fatalf("config import failed: %v", err)
		}
		validateImported := true
		for _, a := range args[2:] {
			if strings.TrimSpace(a) == "--validate=false" {
				validateImported = false
			}
		}
		if validateImported {
			if err := config.Validate(imported); err != nil {
				log.Fatalf("imported config is invalid: %v", err)
			}
		}
		*cfg = *imported
		if err := config.ResolveProviderKeyPath(cfg, configPath); err != nil {
			log.Fatalf("imported config provider key resolution failed: %v", err)
		}
		if err := saveConfigFile(configPath, cfg); err != nil {
			log.Fatalf("failed to write imported config: %v", err)
		}
		log.Printf("Imported config from %s", src)

	default:
		log.Fatalf("unknown config subcommand %q (expected: get, set, validate, export, import)", args[0])
	}
}

func printConfigValue(key string, v any, jsonMode bool) {
	if jsonMode {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		payload := map[string]any{"value": v}
		if key != "" {
			payload["key"] = key
		}
		if err := enc.Encode(payload); err != nil {
			log.Fatalf("failed to encode JSON output: %v", err)
		}
		return
	}

	if key == "" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(v); err != nil {
			log.Fatalf("failed to print config: %v", err)
		}
		return
	}
	fmt.Printf("%s=%v\n", key, v)
}

func getConfigField(cfg *config.Config, key string) (any, error) {
	return config.GetConfigField(cfg, key)
}

func setConfigField(cfg *config.Config, key string, value string) error {
	return config.SetConfigField(cfg, key, value)
}

func saveConfigFile(path string, cfg *config.Config) error {
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		return err
	}
	return util.WriteFileAtomic(path, out.Bytes(), 0o644)
}

// writePIDFile writes the current process ID to the specified path.
func writePIDFile(path string) error {
	pid := os.Getpid()
	data := []byte(strconv.Itoa(pid))
	return util.WriteFileAtomic(path, data, 0o644)
}

// readPIDFile reads the PID from the specified path.
func readPIDFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}
	return pid, nil
}

func applyConfigDefaults(cfg *config.Config) {
	if cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.Logging.Format) == "" {
		cfg.Logging.Format = "text"
	}
	if strings.TrimSpace(cfg.Logging.Level) == "" {
		cfg.Logging.Level = "info"
	}
	if strings.TrimSpace(cfg.Security.KeyStorageMode) == "" {
		cfg.Security.KeyStorageMode = "file"
	}
	if strings.TrimSpace(cfg.Security.KeyStorageService) == "" {
		cfg.Security.KeyStorageService = "BlockchainVPN"
	}
	if strings.TrimSpace(cfg.Security.TLSMinVersion) == "" {
		cfg.Security.TLSMinVersion = "1.3"
	}
	if strings.TrimSpace(cfg.Security.TLSProfile) == "" {
		cfg.Security.TLSProfile = "modern"
	}
	if strings.TrimSpace(cfg.Security.MetricsAuthToken) == "" {
		cfg.Security.MetricsAuthToken = ""
	}
}

type fileStatus struct {
	Path      string `json:"path"`
	Exists    bool   `json:"exists"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
	ModTime   string `json:"mod_time,omitempty"`
}

type statusOutput struct {
	GeneratedAt  string `json:"generated_at"`
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	AppConfigDir string `json:"app_config_dir"`
	ConfigPath   string `json:"config_path"`
	Files        struct {
		Config      fileStatus `json:"config"`
		ProviderKey fileStatus `json:"provider_key"`
		History     fileStatus `json:"history"`
	} `json:"files"`
	RPC struct {
		Host           string `json:"host"`
		User           string `json:"user"`
		PassConfigured bool   `json:"pass_configured"`
	} `json:"rpc"`
	Logging struct {
		Format string `json:"format"`
		Level  string `json:"level"`
	} `json:"logging"`
	Security struct {
		KeyStorageMode      string   `json:"key_storage_mode"`
		KeyStorageResolved  string   `json:"key_storage_resolved"`
		KeyStorageSupported bool     `json:"key_storage_supported"`
		KeyStorageDetail    string   `json:"key_storage_detail"`
		KeyStorageService   string   `json:"key_storage_service"`
		RevocationCacheFile string   `json:"revocation_cache_file"`
		TLSMinVersion       string   `json:"tls_min_version"`
		TLSProfile          string   `json:"tls_profile"`
		TLSCipherProfile    []string `json:"tls_cipher_profile"`
		MetricsAuthEnabled  bool     `json:"metrics_auth_enabled"`
	} `json:"security"`
	Provider struct {
		InterfaceName        string `json:"interface_name"`
		ListenPort           int    `json:"listen_port"`
		AnnounceIP           string `json:"announce_ip"`
		Country              string `json:"country"`
		Price                uint64 `json:"price_sats_per_session"`
		MaxConsumers         int    `json:"max_consumers"`
		EnableNAT            bool   `json:"enable_nat"`
		EnableEgressNAT      bool   `json:"enable_egress_nat"`
		NATOutboundInterface string `json:"nat_outbound_interface"`
		IsolationMode        string `json:"isolation_mode"`
		HealthCheckEnabled   bool   `json:"health_check_enabled"`
		HealthCheckInterval  string `json:"health_check_interval"`
		BandwidthLimit       string `json:"bandwidth_limit"`
		TunIP                string `json:"tun_ip"`
		TunSubnet            string `json:"tun_subnet"`
		MetricsListenAddr    string `json:"metrics_listen_addr"`
	} `json:"provider"`
	Client struct {
		InterfaceName              string `json:"interface_name"`
		TunIP                      string `json:"tun_ip"`
		TunSubnet                  string `json:"tun_subnet"`
		EnableKillSwitch           bool   `json:"enable_kill_switch"`
		MetricsListenAddr          string `json:"metrics_listen_addr"`
		StrictVerification         bool   `json:"strict_verification"`
		VerifyThroughputAfterSetup bool   `json:"verify_throughput_after_connect"`
	} `json:"client"`
	History struct {
		RecordCount int    `json:"record_count"`
		LastPayment string `json:"last_payment,omitempty"`
		LoadError   string `json:"load_error,omitempty"`
	} `json:"history"`
	Networking struct {
		PrivilegesOK    bool   `json:"privileges_ok"`
		PrivilegesError string `json:"privileges_error,omitempty"`
	} `json:"networking"`
	Warnings []string `json:"warnings,omitempty"`
}

func handleStatus(cfg *config.Config, configPath string, jsonMode bool) {
	status := statusOutput{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		ConfigPath:  configPath,
	}

	appConfigDir, err := config.AppConfigDir()
	if err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("could not resolve app config dir: %v", err))
	} else {
		status.AppConfigDir = appConfigDir
	}

	status.Files.Config = inspectFile(configPath)
	status.Files.ProviderKey = inspectFile(cfg.Provider.PrivateKeyFile)
	if appConfigDir != "" {
		status.Files.History = inspectFile(filepath.Join(appConfigDir, "history.json"))
	}

	status.RPC.Host = cfg.RPC.Host
	status.RPC.User = cfg.RPC.User
	status.RPC.PassConfigured = strings.TrimSpace(cfg.RPC.Pass) != ""
	status.Logging.Format = cfg.Logging.Format
	status.Logging.Level = cfg.Logging.Level
	status.Security.KeyStorageMode = cfg.Security.KeyStorageMode
	resolvedStoreMode, storeOK, storeDetail := crypto.KeyStorageStatus(cfg.Security.KeyStorageMode)
	status.Security.KeyStorageResolved = resolvedStoreMode
	status.Security.KeyStorageSupported = storeOK
	status.Security.KeyStorageDetail = storeDetail
	status.Security.KeyStorageService = cfg.Security.KeyStorageService
	status.Security.RevocationCacheFile = cfg.Security.RevocationCacheFile
	status.Security.TLSMinVersion = cfg.Security.TLSMinVersion
	status.Security.TLSProfile = cfg.Security.TLSProfile
	status.Security.MetricsAuthEnabled = strings.TrimSpace(cfg.Security.MetricsAuthToken) != ""
	if tlsPolicy, err := tunnel.ResolveTLSPolicy(cfg.Security.TLSMinVersion, cfg.Security.TLSProfile, cfg.Security.TlsCustomCipherSuites); err == nil {
		status.Security.TLSMinVersion = tlsPolicy.MinVersionLabel
		status.Security.TLSProfile = tlsPolicy.Profile
		status.Security.TLSCipherProfile = tlsPolicy.CipherNames
	} else {
		status.Warnings = append(status.Warnings, fmt.Sprintf("invalid TLS policy config: %v", err))
	}

	status.Provider.InterfaceName = cfg.Provider.InterfaceName
	status.Provider.ListenPort = cfg.Provider.ListenPort
	status.Provider.AnnounceIP = cfg.Provider.AnnounceIP
	status.Provider.Country = cfg.Provider.Country
	status.Provider.Price = cfg.Provider.Price
	status.Provider.MaxConsumers = cfg.Provider.MaxConsumers
	status.Provider.EnableNAT = cfg.Provider.EnableNAT
	status.Provider.EnableEgressNAT = cfg.Provider.EnableEgressNAT
	status.Provider.NATOutboundInterface = cfg.Provider.NATOutboundInterface
	status.Provider.IsolationMode = cfg.Provider.IsolationMode
	status.Provider.HealthCheckEnabled = cfg.Provider.HealthCheckEnabled
	status.Provider.HealthCheckInterval = cfg.Provider.HealthCheckInterval
	status.Provider.BandwidthLimit = cfg.Provider.BandwidthLimit
	status.Provider.TunIP = cfg.Provider.TunIP
	status.Provider.TunSubnet = cfg.Provider.TunSubnet
	status.Provider.MetricsListenAddr = cfg.Provider.MetricsListenAddr

	status.Client.InterfaceName = cfg.Client.InterfaceName
	status.Client.TunIP = cfg.Client.TunIP
	status.Client.TunSubnet = cfg.Client.TunSubnet
	status.Client.EnableKillSwitch = cfg.Client.EnableKillSwitch
	status.Client.MetricsListenAddr = cfg.Client.MetricsListenAddr
	status.Client.StrictVerification = cfg.Client.StrictVerification
	status.Client.VerifyThroughputAfterSetup = cfg.Client.VerifyThroughputAfterSetup

	records, err := history.LoadHistory()
	if err != nil {
		if !os.IsNotExist(err) {
			status.History.LoadError = err.Error()
			status.Warnings = append(status.Warnings, fmt.Sprintf("failed to load history: %v", err))
		}
	} else {
		status.History.RecordCount = len(records)
		if len(records) > 0 {
			sort.Slice(records, func(i, j int) bool { return records[i].Timestamp.After(records[j].Timestamp) })
			status.History.LastPayment = records[0].Timestamp.UTC().Format(time.RFC3339)
		}
	}

	if !status.Files.ProviderKey.Exists && strings.EqualFold(status.Security.KeyStorageMode, "file") {
		status.Warnings = append(status.Warnings, "provider key file does not exist; provider mode will generate a new encrypted key on first start")
	}
	if !status.Security.KeyStorageSupported {
		status.Warnings = append(status.Warnings, "selected key storage mode is not supported on this platform/runtime")
	}
	if strings.TrimSpace(cfg.Provider.AnnounceIP) == "" {
		status.Warnings = append(status.Warnings, "provider announce_ip is empty; public IP will be auto-detected at runtime")
	}
	if cfg.Provider.EnableEgressNAT && strings.TrimSpace(cfg.Provider.NATOutboundInterface) == "" {
		status.Warnings = append(status.Warnings, "provider egress NAT is enabled but nat_outbound_interface is empty")
	}
	if !status.Security.MetricsAuthEnabled && (strings.TrimSpace(cfg.Provider.MetricsListenAddr) != "" || strings.TrimSpace(cfg.Client.MetricsListenAddr) != "") {
		status.Warnings = append(status.Warnings, "metrics endpoint is enabled without auth token; bind to loopback or set security.metrics_auth_token")
	}
	if err := tunnel.EnsureElevatedPrivileges(); err != nil {
		status.Networking.PrivilegesError = err.Error()
		status.Warnings = append(status.Warnings, "automatic networking changes require elevated privileges")
	} else {
		status.Networking.PrivilegesOK = true
	}

	if jsonMode {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(status); err != nil {
			log.Fatalf("Failed to encode status JSON: %v", err)
		}
		return
	}

	fmt.Println("BlockchainVPN Status")
	fmt.Println(strings.Repeat("=", 20))
	fmt.Printf("Generated At: %s\n", status.GeneratedAt)
	fmt.Printf("Platform: %s/%s\n", status.OS, status.Arch)
	fmt.Printf("App Config Dir: %s\n", status.AppConfigDir)
	fmt.Printf("Config Path: %s\n", status.ConfigPath)
	fmt.Println()

	fmt.Println("Files")
	fmt.Println(strings.Repeat("-", 20))
	fmt.Printf("Config: %s\n", formatFileStatus(status.Files.Config))
	fmt.Printf("Provider Key: %s\n", formatFileStatus(status.Files.ProviderKey))
	fmt.Printf("History: %s\n", formatFileStatus(status.Files.History))
	fmt.Println()

	fmt.Println("RPC")
	fmt.Println(strings.Repeat("-", 20))
	fmt.Printf("Host: %s\n", status.RPC.Host)
	fmt.Printf("User: %s\n", status.RPC.User)
	fmt.Printf("Password Configured: %t\n", status.RPC.PassConfigured)
	fmt.Println()
	fmt.Println("Logging")
	fmt.Println(strings.Repeat("-", 20))
	fmt.Printf("Format: %s\n", status.Logging.Format)
	fmt.Printf("Level: %s\n", status.Logging.Level)
	fmt.Println()
	fmt.Println("Security")
	fmt.Println(strings.Repeat("-", 20))
	fmt.Printf("Key Storage Mode: %s (resolved=%s, supported=%t)\n", status.Security.KeyStorageMode, status.Security.KeyStorageResolved, status.Security.KeyStorageSupported)
	fmt.Printf("Key Storage Detail: %s\n", status.Security.KeyStorageDetail)
	fmt.Printf("Key Storage Service: %s\n", status.Security.KeyStorageService)
	fmt.Printf("Revocation Cache File: %s\n", status.Security.RevocationCacheFile)
	fmt.Printf("TLS Min Version: %s\n", status.Security.TLSMinVersion)
	fmt.Printf("TLS Profile: %s\n", status.Security.TLSProfile)
	fmt.Printf("TLS Cipher Profile: %s\n", strings.Join(status.Security.TLSCipherProfile, ", "))
	fmt.Printf("Metrics Auth Enabled: %t\n", status.Security.MetricsAuthEnabled)
	fmt.Println()

	fmt.Println("Provider")
	fmt.Println(strings.Repeat("-", 20))
	fmt.Printf("Interface: %s\n", status.Provider.InterfaceName)
	fmt.Printf("Listen Port: %d\n", status.Provider.ListenPort)
	fmt.Printf("Announce IP: %s\n", status.Provider.AnnounceIP)
	fmt.Printf("Price: %d sats/session\n", status.Provider.Price)
	fmt.Printf("NAT Traversal: %t\n", status.Provider.EnableNAT)
	fmt.Printf("Egress NAT: %t\n", status.Provider.EnableEgressNAT)
	fmt.Printf("NAT Outbound Interface: %s\n", status.Provider.NATOutboundInterface)
	fmt.Printf("Isolation Mode: %s\n", status.Provider.IsolationMode)
	fmt.Printf("Health Checks: %t (%s)\n", status.Provider.HealthCheckEnabled, status.Provider.HealthCheckInterval)
	fmt.Printf("Bandwidth Limit: %s\n", status.Provider.BandwidthLimit)
	fmt.Printf("TUN: %s/%s\n", status.Provider.TunIP, status.Provider.TunSubnet)
	fmt.Printf("Metrics Listen Addr: %s\n", status.Provider.MetricsListenAddr)
	fmt.Println()

	fmt.Println("Client")
	fmt.Println(strings.Repeat("-", 20))
	fmt.Printf("Interface: %s\n", status.Client.InterfaceName)
	fmt.Printf("TUN: %s/%s\n", status.Client.TunIP, status.Client.TunSubnet)
	fmt.Printf("Kill Switch: %t\n", status.Client.EnableKillSwitch)
	fmt.Printf("Metrics Listen Addr: %s\n", status.Client.MetricsListenAddr)
	fmt.Println()

	fmt.Println("History")
	fmt.Println(strings.Repeat("-", 20))
	fmt.Printf("Records: %d\n", status.History.RecordCount)
	if status.History.LastPayment != "" {
		fmt.Printf("Last Payment: %s\n", status.History.LastPayment)
	}
	if status.History.LoadError != "" {
		fmt.Printf("Load Error: %s\n", status.History.LoadError)
	}
	fmt.Println()

	fmt.Println("Networking")
	fmt.Println(strings.Repeat("-", 20))
	fmt.Printf("Privileges OK: %t\n", status.Networking.PrivilegesOK)
	if status.Networking.PrivilegesError != "" {
		fmt.Printf("Privilege Error: %s\n", status.Networking.PrivilegesError)
	}

	if len(status.Warnings) > 0 {
		fmt.Println()
		fmt.Println("Warnings")
		fmt.Println(strings.Repeat("-", 20))
		for _, warning := range status.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func inspectFile(path string) fileStatus {
	st := fileStatus{Path: path}
	if strings.TrimSpace(path) == "" {
		return st
	}
	info, err := os.Stat(path)
	if err != nil {
		return st
	}
	st.Exists = true
	st.SizeBytes = info.Size()
	st.ModTime = info.ModTime().UTC().Format(time.RFC3339)
	return st
}

func formatFileStatus(st fileStatus) string {
	if !st.Exists {
		return fmt.Sprintf("%s (missing)", st.Path)
	}
	return fmt.Sprintf("%s (size=%d, modified=%s)", st.Path, st.SizeBytes, st.ModTime)
}

func printHelp() {
	fmt.Print(`BlockchainVPN - Decentralized VPN Marketplace

BlockchainVPN is a decentralized VPN marketplace where anyone can:
  1. Become a VPN provider and earn tokens
  2. Connect to providers and pay for bandwidth
  3. Automatic on-chain payment settlement and authorization

Usage: bcvpn <command> [options]

Commands:
  generate-config         Generate a default configuration file
  setup                     Run interactive guided setup
  config                  Manage configuration (see 'bcvpn help config')
  version                 Show version information
  about                   Show about info and donation addresses
  help                    Show this help message

  Infrastructure:
  status [--json]                 Show current status and configuration
  events [--limit N] [--json]     Show recent runtime events
  doctor [--json]                 Run diagnostics and health checks
  diagnostics [--out PATH]        Export diagnostics bundle for troubleshooting
  history [--json] [--table] [--from DATETIME] [--to DATETIME] Show payment transaction history

  VPN Client:
  scan [options]                  Scan for VPN providers and connect interactively
  connect <provider-ip> [options] Connect directly to a provider
  disconnect                      Disconnect active VPN tunnel
  generate-send-address           Generate a new address for sending (via RPC)
  generate-tls-keypair            Generate TLS-compatible keypair for the tunnel
  favorite [add|remove] <pubkey> [comment]  Manage favorite providers
  rate <pubkey> <rating> [comment]         Rate a provider

  VPN Provider:
  start-provider [options]        Start as a VPN provider (earn tokens)
  stop-provider                   Stop the running provider service
  restart-provider                Restart the provider service
  generate-provider-key           Generate a new provider private key
  rotate-provider-key             Rotate your provider private key
  rebroadcast                     Re-broadcast your service announcement
  generate-receive-address        Generate a new address for receiving (via RPC)
  generate-tls-keypair            Generate TLS-compatible keypair (shared)

Options:
  -h, --help    Show this help message
  -v, --version Show version information
  -a, --about   Show about info and donation addresses

Subcommand Help:
  bcvpn help <command> for detailed usage of specific commands.
  Detailed help available for: config, scan, start-provider, connect, 
  generate-config, version, about, status, events, doctor, diagnostics, 
  history, generate-send-address, generate-receive-address, generate-tls-keypair, 
  favorite, rate, generate-provider-key, disconnect, stop-provider, 
  restart-provider, rotate-provider-key, rebroadcast, setup.

Examples:
  bcvpn generate-config               # Create default config.json
  bcvpn setup                        # Run interactive setup wizard
  bcvpn scan                         # Find and connect to providers
  bcvpn connect 1.2.3.4 --port 51820 # Connect directly to provider
  bcvpn start-provider               # Start as a provider
  bcvpn status                       # Check current status
  bcvpn doctor                       # Run diagnostics
  bcvpn config set rpc.host localhost # Update RPC host

For more information, visit: https://github.com/anomalyco/blockchain-vpn
`)
}

func printConfigHelp() {
	fmt.Print(`BlockchainVPN Config Management

Usage: bcvpn config <subcommand> [options]

Subcommands:
  validate               Validate the current config file
  export <path>          Export config to a file
  import <path>          Import config from a file

  Use 'bcvpn config get' and 'bcvpn config set' to view/modify values.
  Run 'bcvpn help config' for details on config subcommands.

Examples:
  bcvpn config get                     # Get entire config as JSON
  bcvpn config get rpc.host            # Get RPC host
  bcvpn config set rpc.host localhost  # Set RPC host
  bcvpn config validate                # Validate config file
  bcvpn config export backup.json      # Export config to file

For more information, visit: https://github.com/anomalyco/blockchain-vpn
`)
}

func printScanHelp() {
	fmt.Print(`BlockchainVPN Provider Scan

Usage: bcvpn scan [options]

The scan command discovers available VPN providers on the network,
enriches them with latency and geolocation data, and allows you to
select one to connect to.

Filter Options:
  --country <code>          Filter by country (e.g., US, DE, GB)
  --max-price <sats>        Maximum price in sats per session
  --pricing-method <method> Filter by pricing: session, time, or data
  --min-bandwidth-kbps <kbps> Minimum advertised bandwidth in Kbps
  --max-latency-ms <ms>     Maximum latency in milliseconds
  --min-available-slots <n> Minimum available consumer slots
  --min-score <float>       Minimum provider score (0-100)
  --sort <method>           Sort by:
    country       - Alphabetical by country
    price         - Lowest price first
    bw            - Highest bandwidth first (bandwidth)
    latency       - Lowest latency first (default)
    capacity      - Most available slots first
    score         - Best overall score (recommended)

Options:
  --startblock <height>  Block height to start scanning from (0 = full scan)
  --limit <N>            Limit number of results (max 100)
  --rescan               Force a fresh scan, ignoring cached results
  --dry-run              Simulate connection without spending funds

Examples:
  bcvpn scan                           # Find providers with default sorting
  bcvpn scan --sort=price              # Find cheapest providers
  bcvpn scan --country=US --sort=score # Best US providers by score
  bcvpn scan --max-price=1000          # Providers under 1000 sats
  bcvpn scan --min-bandwidth-kbps=25000 # High-speed providers

For more information, visit: https://github.com/anomalyco/blockchain-vpn
`)
}

func printStartProviderHelp() {
	fmt.Print(`BlockchainVPN Provider Mode

Usage: bcvpn start-provider [options]

Start as a VPN provider to sell bandwidth to other users. The provider
mode announces your service on the blockchain, monitors for incoming
payments, and authorizes clients based on valid payments.

Requirements:
  - Running ordexcoind node with RPC enabled
  - Wallet funded with tokens for announcement fees
  - Elevated privileges (sudo/admin) for network configuration

Provider Management Commands:
  bcvpn start-provider         Start as a VPN provider
  bcvpn rebroadcast            Re-broadcast your service announcement
  bcvpn generate-provider-key  Generate a new provider private key
  bcvpn rotate-provider-key    Rotate your provider private key
  bcvpn restart-provider       Restart the provider service
  bcvpn stop-provider          Stop the provider service

Key Options:
  --key-password-env <var>  Environment variable containing key password

Configuration:
  Provider settings are read from config.json. Key fields:
    - provider.listen_port      Port for incoming connections (default: 51820)
    - provider.price            Price in sats per session
    - provider.max_consumers    Maximum simultaneous clients (0=unlimited)
    - provider.bandwidth_limit  e.g., "100mbit", "0" for auto-test
    - provider.enable_nat       Enable NAT traversal (UPnP/NAT-PMP)

Payment Models:
  Configure pricing method in config.json:
    - provider.pricing_method    session, time, or data
    - provider.billing_time_unit  minute or hour (for time pricing)
    - provider.billing_data_unit   MB or GB (for data pricing)

Examples:
  bcvpn start-provider                         # Start provider
  bcvpn start-provider --key-password-env PASS  # Non-interactive with env var
  bcvpn generate-provider-key                  # Create a new provider key
  bcvpn rebroadcast                            # Re-announce your service
  bcvpn stop-provider                          # Stop the provider service

For more information, visit: https://github.com/anomalyco/blockchain-vpn
`)
}

func printConnectHelp() {
	fmt.Print(`BlockchainVPN Direct Connect

Usage: bcvpn connect <provider-ip> [options]

Connect directly to a known provider without scanning. This is useful when
you have a provider's IP address or want to bypass the interactive scan.

Arguments:
  <provider-ip>            Provider's IP address or hostname

Connection Options:
  --port <port>           Provider port (default: 51820)
  --pubkey <hex>         Provider's public key (hex)
  --price <sats>         Expected price in sats (for verification)

DNS and Routing Options:
  --dns <servers>         Custom DNS servers (default: 1.1.1.1, 8.8.8.8)
  --no-auto-dns          Don't configure DNS automatically
  --no-auto-route        Don't configure routing automatically
  --full-dns             Route all DNS through VPN (default)
  --split-tunnel <nets>  Only route specific networks through VPN

Payment and Session Options:
  --spending-limit <sats>  Maximum total spending in sats
  --max-session-spending <sats>  Maximum per-session spending
  --auto-reconnect        Automatically reconnect on disconnect

Client Options:
  --interface <name>     TUN interface name (default: bcvpn1)
  --tun-ip <ip>         Client TUN IP (default: 10.10.0.2)
  --kill-switch          Block traffic if VPN disconnects
  --strict-verification  Enable strict security verification

Examples:
  bcvpn connect 1.2.3.4                        # Connect to provider at 1.2.3.4
  bcvpn connect 1.2.3.4 --port 51821           # Custom port
  bcvpn connect 1.2.3.4 --no-auto-dns          # Skip DNS configuration
  bcvpn connect 1.2.3.4 --kill-switch          # Enable kill switch

The application automatically:
  - Handles payment settlement on-chain
  - Monitors connection and renews authorization
  - Tracks data/time usage and calculates costs
  - Enforces spending limits you've configured

For more information, visit: https://github.com/anomalyco/blockchain-vpn
`)
}

// Detailed help functions for commands

func printGenerateConfigHelp() {
	fmt.Print(`BlockchainVPN Generate Config

Usage: bcvpn generate-config

Generate a default configuration file at the standard location. The config
includes sensible defaults for both client and provider modes. You can edit
the file after generation.

Options:
  None

Example:
  bcvpn generate-config
`)
}

func printSetupHelp() {
	fmt.Print(`BlockchainVPN Setup

Usage: bcvpn setup

Start an interactive guided setup that walks you through configuration.
You'll be asked about RPC, logging, security, and whether to configure
as a provider or client. Sensible defaults are provided.

Options:
  None

Example:
  bcvpn setup
`)
}

func printVersionHelp() {
	fmt.Print(`BlockchainVPN Version

Usage: bcvpn version [--json]

Show version information.

Options:
  --json          Output in JSON format (version, commit, build date)

Example:
  bcvpn version
`)
}

func printAboutHelp() {
	fmt.Print(`BlockchainVPN About

Usage: bcvpn about [--json]

Show about information and donation addresses.

Options:
  --json          Output in JSON format

Example:
  bcvpn about
`)
}

func printStatusHelp() {
	fmt.Print(`BlockchainVPN Status

Usage: bcvpn status [--json]

Show current status and configuration. This displays config values, file
status, RPC connection info, security settings, and network privileges.

Options:
  --json          Output in JSON format

Example:
  bcvpn status
`)
}

func printEventsHelp() {
	fmt.Print(`BlockchainVPN Recent Events

Usage: bcvpn events [--limit N] [--json]

Show recent runtime events from the VPN tunnel. This provides a log of
connection events, errors, and state changes.

Options:
  --limit <N>     Maximum number of events to show (default: 100)
  --json          Output in JSON format

Examples:
  bcvpn events
  bcvpn events --limit 50
`)
}

func printDoctorHelp() {
	fmt.Print(`BlockchainVPN Diagnostics

Usage: bcvpn doctor [--json]

Run diagnostics and health checks. This verifies configuration, network
privileges, required tools, and security settings. It outputs a report with
any issues found.

Options:
  --json          Output results in JSON format

Example:
  bcvpn doctor
`)
}

func printDiagnosticsHelp() {
	fmt.Print(`BlockchainVPN Export Diagnostics

Usage: bcvpn diagnostics [--out PATH]

Export a diagnostics bundle for troubleshooting. The bundle includes config,
recent events, and runtime metrics in a JSON file.

Options:
  --out <PATH>    Output file path (default: app config dir with timestamp)

Example:
  bcvpn diagnostics --out /tmp/diag.json
`)
}

func printHistoryHelp() {
	fmt.Print(`BlockchainVPN Payment History

Usage: bcvpn history [--json] [--table] [--from DATETIME] [--to DATETIME]

Show payment transaction history. By default shows all recorded payments.

Options:
  --json          Output in JSON format
  --table         Show as table (default)
  --from <datetime>  Show transactions from this date/time
  --to <datetime>    Show transactions to this date/time

Example:
  bcvpn history --since-last-payment
`)
}

func printGenerateSendAddressHelp() {
	fmt.Print(`BlockchainVPN Generate Send Address

Usage: bcvpn generate-send-address

Generates a new address for sending funds via RPC. This address can be used
as the source for payments.

This command connects to the RPC daemon and calls GetNewAddress. The new
address is printed to stdout.

No options are available.

Example:
  bcvpn generate-send-address
`)
}

func printGenerateReceiveAddressHelp() {
	fmt.Print(`BlockchainVPN Generate Receive Address

Usage: bcvpn generate-receive-address

Generates a new address for receiving payments via RPC. This address can be
shared with others to receive funds.

This command connects to the RPC daemon and calls GetNewAddress. The new
address is printed to stdout.

No options are available.

Example:
  bcvpn generate-receive-address
`)
}

func printGenerateTLSKeypairHelp() {
	fmt.Print(`BlockchainVPN Generate TLS Keypair

Usage: bcvpn generate-tls-keypair

Generates a TLS-compatible keypair for the VPN tunnel. The private key and
public key are printed in hexadecimal format.

The private key should be kept secret. The public key can be shared with
others if needed for authentication.

No options are available.

Example:
  bcvpn generate-tls-keypair
`)
}

func printFavoriteHelp() {
	fmt.Print(`BlockchainVPN Favorite Providers

Usage: bcvpn favorite [add|remove] <pubkey> [comment]

Manage your list of favorite (trusted) providers. You can add or remove a
provider by its public key (hex). An optional comment can be added to
identify the provider.

Subcommands:
  add <pubkey> [comment]     Add a provider to favorites
  remove <pubkey>            Remove a provider from favorites

Examples:
  bcvpn favorite add 02abc...def "US, fast"
  bcvpn favorite remove 02abc...def
`)
}

func printRateHelp() {
	fmt.Print(`BlockchainVPN Rate Provider

Usage: bcvpn rate <pubkey> <rating> [comment]

Rate a VPN provider. The rating is an integer from 1 to 5. An optional
comment can provide additional feedback. Ratings are stored locally in the
application's config directory.

Arguments:
  <pubkey>     Provider's public key (hex)
  <rating>     Rating value (1-5)

Example:
  bcvpn rate 02abc...def 5 "Excellent service, low latency"
`)
}

func printGenerateProviderKeyHelp() {
	fmt.Print(`BlockchainVPN Generate Provider Key

Usage: bcvpn generate-provider-key [--dry-run]

Generates a new encrypted provider private key. The key is saved to the path
specified in the configuration (provider.private_key_file). You will be
prompted to enter a password for encryption.

Options:
  --dry-run        Show what would be generated without creating files

This key is used for both TLS server authentication and for receiving
payments. Keep it secure.

Example:
  bcvpn generate-provider-key
`)
}

func printDisconnectHelp() {
	fmt.Print(`BlockchainVPN Disconnect Client

Usage: bcvpn disconnect

Disconnect an active VPN client connection. This sends a SIGTERM to the
client process and removes the PID file.

This command works only if the client is running in the same user session
and the PID file exists.

Options:
  None

Example:
  bcvpn disconnect
`)
}

func printStopProviderHelp() {
	fmt.Print(`BlockchainVPN Stop Provider

Usage: bcvpn stop-provider

Stop a running provider service. This sends a SIGTERM to the provider process
and removes the PID file.

This command works only if the provider is running in the same user session
and the PID file exists.

Options:
  None

Example:
  bcvpn stop-provider
`)
}

func printRestartProviderHelp() {
	fmt.Print(`BlockchainVPN Restart Provider

Usage: bcvpn restart-provider

Restart the provider service. This is equivalent to stopping and then starting
the provider. You will need to start the provider manually after stopping.

Options:
  None

Example:
  bcvpn restart-provider
`)
}

func printRotateProviderKeyHelp() {
	fmt.Print(`BlockchainVPN Rotate Provider Key

Usage: bcvpn rotate-provider-key [--key-file <path>] [--old-password-env <var>] [--new-password-env <var>]

Rotate your provider private key. This creates a new encrypted key file and
backs up the old one. You will be prompted for the old and new passwords
unless provided via environment variables.

Options:
  --key-file <path>        Path to provider private key (default: from config)
  --old-password-env <var> Environment variable containing current password
  --new-password-env <var> Environment variable containing new password

Example:
  bcvpn rotate-provider-key
`)
}

func printRebroadcastHelp() {
	fmt.Print(`BlockchainVPN Re-broadcast Service

Usage: bcvpn rebroadcast [--dry-run] [--key-password-env <var>]

Re-broadcast your service announcement on the blockchain. This is useful to
refresh your announcement or update your advertised details.

Options:
  --dry-run               Simulate announcement without making real transaction
  --key-password-env <var> Environment variable containing provider key password

Example:
  bcvpn rebroadcast
`)
}
