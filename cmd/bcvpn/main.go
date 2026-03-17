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

func main() {
	// Handle help and version flags early, before requiring config
	if len(os.Args) >= 2 && (os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help") {
		if len(os.Args) >= 3 && os.Args[1] == "help" && os.Args[2] == "config" {
			printConfigHelp()
			os.Exit(0)
		}
		printHelp()
		os.Exit(0)
	}

	if len(os.Args) >= 2 && (os.Args[1] == "-v" || os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Println(version.String())
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
	_ = tunnel.RecoverPendingNetworkState()

	// Subcommands
	scanCmd := flag.NewFlagSet("scan", flag.ExitOnError)
	startProviderCmd := flag.NewFlagSet("start-provider", flag.ExitOnError)
	historyCmd := flag.NewFlagSet("history", flag.ExitOnError)
	rebroadcastCmd := flag.NewFlagSet("rebroadcast", flag.ExitOnError)
	updatePriceCmd := flag.NewFlagSet("update-price", flag.ExitOnError)
	rotateKeyCmd := flag.NewFlagSet("rotate-provider-key", flag.ExitOnError)
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	configCmd := flag.NewFlagSet("config", flag.ExitOnError)
	versionCmd := flag.NewFlagSet("version", flag.ExitOnError)
	aboutCmd := flag.NewFlagSet("about", flag.ExitOnError)
	doctorCmd := flag.NewFlagSet("doctor", flag.ExitOnError)
	eventsCmd := flag.NewFlagSet("events", flag.ExitOnError)
	diagCmd := flag.NewFlagSet("diagnostics", flag.ExitOnError)

	// Scan specific flags
	scanStartBlock := scanCmd.Int64("startblock", 0, "Block height to start scanning from (0 for full scan)")
	scanSortBy := scanCmd.String("sort", "latency", "Sort providers by 'price', 'country', 'latency', 'bandwidth', 'capacity', or 'score'")
	scanCountry := scanCmd.String("country", "", "Filter providers by country code (e.g., US, DE)")
	scanMaxPrice := scanCmd.Uint64("max-price", 0, "Filter providers with price <= this value (units depend on pricing method) (0 disables)")
	scanPricingMethod := scanCmd.String("pricing-method", "", "Filter providers by pricing method: session, time, data")
	scanMinBandwidthKB := scanCmd.Uint("min-bandwidth-kbps", 0, "Filter providers with advertised bandwidth >= this value in Kbps")
	scanMaxLatencyMS := scanCmd.Int("max-latency-ms", 0, "Filter providers with latency <= this value in ms (0 disables)")
	scanMinSlots := scanCmd.Int("min-available-slots", 0, "Filter providers with available slot capacity >= this value (0 disables)")
	scanDryRun := scanCmd.Bool("dry-run", false, "Simulate connection without spending funds or modifying interfaces")
	historySinceLast := historyCmd.Bool("since-last-payment", false, "Show wallet transactions since the last recorded payment")
	startProviderKeyPassEnv := startProviderCmd.String("key-password-env", "", "Env var name containing provider key password (file mode)")
	rebroadcastKeyPassEnv := rebroadcastCmd.String("key-password-env", "", "Env var name containing provider key password (file mode)")
	updatePriceKeyPassEnv := updatePriceCmd.String("key-password-env", "", "Env var name containing provider key password (file mode)")

	// Update-price specific flags
	updatePriceNewPrice := updatePriceCmd.Uint64("price", 0, "The new price in satoshis per session")
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

	switch os.Args[1] {
	case "start-provider":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		if err := tunnel.EnsureElevatedPrivileges(); err != nil {
			log.Printf("WARNING: Provider mode requires automatic networking privileges.")
			log.Printf("Provider functionality will be limited without elevated privileges.")
			log.Printf("Hint: Run with sudo or as Administrator for full functionality.")
			log.Fatalf("Provider mode requires automatic networking privileges: %v", err)
		}

		startProviderCmd.Parse(os.Args[2:])
		client := connectRPCWithConfig(cfg)
		defer client.Shutdown()

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

		endpoint := buildProviderEndpoint(&cfg.Provider, announceIP, announcePort, providerKey)

		var providerWG sync.WaitGroup
		providerWG.Add(5)
		go func() {
			defer providerWG.Done()
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()
			if err := blockchain.AnnounceService(client, endpoint, cfg.Provider.AnnouncementFeeTargetBlocks, cfg.Provider.AnnouncementFeeMode); err != nil {
				log.Printf("Initial service announcement failed: %v", err)
			}
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := blockchain.AnnounceService(client, endpoint, cfg.Provider.AnnouncementFeeTargetBlocks, cfg.Provider.AnnouncementFeeMode); err != nil {
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

			if err := blockchain.AnnounceHeartbeat(client, providerKey.PubKey(), protocol.AvailabilityFlagAvailable); err != nil {
				log.Printf("Initial heartbeat broadcast failed: %v", err)
			}
			for {
				select {
				case <-ctx.Done():
					return
				case <-hbTicker.C:
					if err := blockchain.AnnounceHeartbeat(client, providerKey.PubKey(), protocol.AvailabilityFlagAvailable); err != nil {
						log.Printf("Scheduled heartbeat broadcast failed: %v", err)
					}
				}
			}
		}()
		// Build payment monitor configuration from provider settings
		pmCfg := blockchain.PaymentMonitorCfg{
			Price:          cfg.Provider.Price,
			PricingMethod:  strings.ToLower(strings.TrimSpace(cfg.Provider.PricingMethod)),
			MaxSessionSecs: cfg.Provider.MaxSessionDurationSecs,
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
		rebroadcastCmd.Parse(os.Args[2:])
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client := connectRPCWithConfig(cfg)
		defer client.Shutdown()

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

		endpoint := buildProviderEndpoint(&cfg.Provider, announceIP, announcePort, providerKey)

		log.Println("Re-broadcasting service announcement...")
		if err := blockchain.AnnounceService(client, endpoint, cfg.Provider.AnnouncementFeeTargetBlocks, cfg.Provider.AnnouncementFeeMode); err != nil {
			log.Fatalf("Service announcement failed: %v", err)
		}
		log.Println("Service announcement re-broadcasted successfully.")

	case "update-price":
		updatePriceCmd.Parse(os.Args[2:])
		if *updatePriceNewPrice == 0 {
			log.Fatal("--price must be a positive value")
		}

		client := connectRPCWithConfig(cfg)
		defer client.Shutdown()

		providerKey, err := getProviderKey(cfg, *updatePriceKeyPassEnv)
		if err != nil {
			log.Fatalf("Failed to get provider key: %v", err)
		}

		log.Printf("Broadcasting price update to %d satoshis...", *updatePriceNewPrice)
		if err := blockchain.AnnouncePriceUpdate(client, providerKey.PubKey(), *updatePriceNewPrice, cfg.Provider.AnnouncementFeeTargetBlocks, cfg.Provider.AnnouncementFeeMode); err != nil {
			log.Fatalf("Price update announcement failed: %v", err)
		}
		log.Println("Price update broadcasted successfully.")

	case "scan":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		scanCmd.Parse(os.Args[2:])
		client := connectRPCWithConfig(cfg)
		defer client.Shutdown()

		genesisHash, err := client.GetBlockHash(0)
		if err != nil {
			log.Fatalf("Failed to get genesis block hash from RPC: %v", err)
		}

		chainParams := detectChain(genesisHash)

		cachePath, err := blockchain.DefaultScanCachePath()
		if err != nil {
			log.Printf("Warning: failed to get default scan cache path: %v", err)
		}
		var cache *blockchain.ScanCache
		if cachePath != "" {
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
			fmt.Println("No VPN endpoints found.")
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
		)

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
			fmt.Printf(
				"  [%d] Country: %s, Latency: %v, IP: %s, Port: %d, Price: %d sats/session, Bandwidth: %d Kbps, Capacity: %s, Score: %.2f\n",
				i,
				effectiveCountry(ep),
				ep.Latency.Round(time.Millisecond),
				ep.Endpoint.IP,
				ep.Endpoint.Port,
				ep.Endpoint.Price,
				ep.AdvertisedBandwidthKB,
				displayCapacity(ep),
				computeProviderScore(ep),
			)
		}
		fmt.Println()

		interactiveConnect(ctx, client, chainParams, filteredEndpoints, &cfg.Client, &cfg.Security, *scanDryRun)

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

		if *historySinceLast {
			handleHistorySinceLast(cfg)
		} else {
			handleFullHistory()
		}

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
		log.Fatalf("Error creating new RPC client: %v", err)
	}

	// Check server warmup status if needed
	if err := waitForServerReady(context.Background(), client); err != nil {
		log.Fatalf("RPC server not ready: %v", err)
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
	// Determine cookie path: use configured cookie file or default location
	cookiePath := strings.TrimSpace(cfg.RPC.CookieFile)
	if cookiePath == "" {
		// Try to derive from RPC host's data directory, or use common defaults
		// For simplicity, check standard locations based on OS and network
		home, _ := os.UserHomeDir()

		commonPaths := []string{
			filepath.Join(home, ".ordexcoin", ".cookie"),
			filepath.Join(home, ".ordexcoin", "regtest", ".cookie"),
			filepath.Join(home, ".ordexcoin", "testnet3", ".cookie"),
			filepath.Join(home, ".ordexcoin", "signet", ".cookie"),
			"/var/lib/ordexcoin/.cookie",
			"C:\\ProgramData\\OrdExcoin\\.cookie",
		}
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

	reader := bufio.NewReader(os.Stdin)
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
			fmt.Print("Enter password to decrypt provider key: ")
			pass, _ := reader.ReadString('\n')
			password = []byte(strings.TrimSpace(pass))
		}
		key, err := crypto.LoadAndDecryptKey(keyPath, password)
		if err != nil {
			return nil, fmt.Errorf("failed to load and decrypt key: %w", err)
		}
		log.Println("Provider key successfully decrypted.")
		return key, nil
	}

	fmt.Println("No provider key found. Let's create a new encrypted key.")
	password := passwordFromEnv
	if len(password) == 0 {
		fmt.Print("Enter new password for provider key: ")
		pass1, _ := reader.ReadString('\n')
		fmt.Print("Confirm new password: ")
		pass2, _ := reader.ReadString('\n')
		if strings.TrimSpace(pass1) != strings.TrimSpace(pass2) {
			return nil, fmt.Errorf("passwords do not match")
		}
		password = []byte(strings.TrimSpace(pass1))
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

func buildProviderEndpoint(providerCfg *config.ProviderConfig, announceIP net.IP, announcePort int, providerKey *btcec.PrivateKey) *protocol.VPNEndpoint {
	bandwidthKB := parseBandwidthLimitToKbps(providerCfg.BandwidthLimit)
	maxConsumers := uint16(0)
	if providerCfg.MaxConsumers > 0 && providerCfg.MaxConsumers <= 65535 {
		maxConsumers = uint16(providerCfg.MaxConsumers)
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
		CountryCode:           strings.ToUpper(strings.TrimSpace(providerCfg.Country)),
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

func interactiveConnect(ctx context.Context, client *rpcclient.Client, chainParams *chaincfg.Params, endpoints []*geoip.EnrichedVPNEndpoint, clientCfg *config.ClientConfig, secCfg *config.SecurityConfig, dryRun bool) {
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

	// Initialize spending manager if limits are enabled
	var spendingMgr *tunnel.SpendingManager
	if clientCfg.SpendingLimitEnabled || clientCfg.AutoRechargeEnabled {
		spendingMgr = tunnel.NewSpendingManager(clientCfg, client, providerAddr, localKey, selectedEndpoint.Endpoint.PublicKey)
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
		_, err = blockchain.SendPayment(client, providerAddr, selectedEndpoint.Endpoint.Price, localKey.PubKey())
		if err != nil {
			log.Fatalf("Failed to send payment: %v", err)
		}

		// Record successful payment in spending manager (already did pre-check, now finalize)
		spendingMgr.AddCredits(0) // trigger log if needed
	}

	peerPubKey := selectedEndpoint.Endpoint.PublicKey
	endpointAddr := fmt.Sprintf("%s:%d", selectedEndpoint.Endpoint.IP.String(), selectedEndpoint.Endpoint.Port)
	fmt.Printf("\nConnecting to %s...\n", endpointAddr)

	if dryRun {
		fmt.Printf("[Dry Run] Simulation: Would create TUN interface %s and connect to %s.\n", clientCfg.InterfaceName, endpointAddr)
	} else {
		mgr := tunnel.NewMultiTunnelManager()
		expected := tunnel.ClientSecurityExpectations{
			ExpectedCountry:     selectedEndpoint.Country,
			ExpectedBandwidthKB: selectedEndpoint.AdvertisedBandwidthKB,
			ThroughputProbePort: selectedEndpoint.Endpoint.ThroughputProbePort,
		}
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
		); err != nil {
			log.Fatalf("Failed to add tunnel: %v", err)
		}

		// Wait indefinitely until cancelled or the tunnel errors out (handled by mgr)
		fmt.Println("Press Ctrl+C to disconnect.")
		<-ctx.Done()
		fmt.Println("Shutting down tunnel...")
		mgr.CancelAll()
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
		sort.Slice(endpoints, func(i, j int) bool { return effectiveCountry(endpoints[i]) < effectiveCountry(endpoints[j]) })
		fmt.Println("Sorted by country.")
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

func filterEndpoints(endpoints []*geoip.EnrichedVPNEndpoint, country string, maxPrice uint64, minBandwidthKB uint32, maxLatency time.Duration, minSlots int, pricingMethod string) []*geoip.EnrichedVPNEndpoint {
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

Usage: bcvpn <command> [options]

Commands:
  scan                    Scan for available VPN providers and connect
  start-provider          Start as a VPN provider (requires config.json)
  rebroadcast             Re-broadcast your provider announcement
  update-price            Update your provider service price
  rotate-provider-key     Rotate your provider private key
  history                 Show payment transaction history
  status                  Show current status and configuration
  config                  Manage configuration (see 'bcvpn help config')
  doctor                  Run diagnostics and health checks
  events                  Show recent runtime events
  diagnostics             Export diagnostics bundle for troubleshooting
  version                 Show version information
  about                   Show about info and donation addresses
  generate-config         Generate a default configuration file

Options:
  -h, --help              Show this help message
  -v, --version           Show version information

Examples:
  bcvpn help                           # Show this help message
  bcvpn help config                    # Show config subcommands
  bcvpn generate-config                 # Create default config.json
  bcvpn scan                           # Find available VPN providers
  bcvpn start-provider                 # Start as a VPN provider
  bcvpn status                         # Check current status
  bcvpn config get rpc.host            # Get specific config value
  bcvpn config set rpc.host localhost  # Set specific config value
  bcvpn doctor                         # Run diagnostics

For more information, visit: https://github.com/anomalyco/blockchain-vpn
`)
}

func printConfigHelp() {
	fmt.Print(`BlockchainVPN Config Management

Usage: bcvpn config <subcommand> [options]

Subcommands:
  get <key>              Get a config value (e.g., bcvpn config get rpc.host)
  set <key> <value>      Set a config value (e.g., bcvpn config set rpc.host localhost)
  validate               Validate the current config file
  export <path>          Export config to a file
  import <path>          Import config from a file

Examples:
  bcvpn config get rpc.host            # Get RPC host
  bcvpn config set rpc.host localhost  # Set RPC host
  bcvpn config get                     # Get entire config as JSON
  bcvpn config validate                # Validate config file
  bcvpn config export backup.json      # Export config to file

For more information, visit: https://github.com/anomalyco/blockchain-vpn
`)
}
