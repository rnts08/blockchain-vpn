package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"net"
	"os"
	"path/filepath"
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
	"blockchain-vpn/internal/protocol"
	"blockchain-vpn/internal/tunnel"
	"blockchain-vpn/internal/util"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
)

type guiTheme struct{}

func (t *guiTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 12, G: 92, B: 64, A: 255}
	case theme.ColorNameBackground:
		return color.NRGBA{R: 246, G: 245, B: 239, A: 255}
	case theme.ColorNameButton:
		return color.NRGBA{R: 28, G: 128, B: 98, A: 255}
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (t *guiTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *guiTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *guiTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

type guiState struct {
	mu sync.Mutex

	cfgPath string
	cfg     *config.Config

	logs binding.String

	providerRunning bool
	providerCancel  context.CancelFunc

	scanResults []*geoip.EnrichedVPNEndpoint
	selectedIdx int
}

func main() {
	a := app.NewWithID("com.blockchainvpn.gui")
	a.Settings().SetTheme(&guiTheme{})

	w := a.NewWindow("BlockchainVPN")
	w.Resize(fyne.NewSize(1280, 860))

	state, err := initState()
	if err != nil {
		dialog.ShowError(fmt.Errorf("initialization failed: %w", err), w)
		return
	}

	providerTab := buildProviderTab(w, state)
	clientTab := buildClientTab(w, state)
	statusTab := buildStatusTab(state)
	walletTab := buildWalletTab(state)

	tabs := container.NewAppTabs(
		container.NewTabItem("Provider Mode", providerTab),
		container.NewTabItem("Client Mode", clientTab),
		container.NewTabItem("Network Status", statusTab),
		container.NewTabItem("Wallet", walletTab),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	bg := canvas.NewRectangle(color.NRGBA{R: 236, G: 235, B: 226, A: 255})
	content := container.NewMax(bg, tabs)
	w.SetContent(content)

	w.SetCloseIntercept(func() {
		state.stopProvider()
		w.Close()
	})

	w.ShowAndRun()
}

func initState() (*guiState, error) {
	cfgPath, cfg, err := loadConfigWithFallback()
	if err != nil {
		return nil, err
	}
	logs := binding.NewString()
	_ = logs.Set("GUI ready.\n")

	return &guiState{
		cfgPath:     cfgPath,
		cfg:         cfg,
		logs:        logs,
		selectedIdx: -1,
	}, nil
}

func loadConfigWithFallback() (string, *config.Config, error) {
	defaultPath, err := config.DefaultConfigPath()
	if err != nil {
		return "", nil, err
	}
	path := defaultPath
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if _, legacyErr := os.Stat("config.json"); legacyErr == nil {
			path = "config.json"
		} else {
			if err := config.GenerateDefaultConfig(defaultPath); err != nil {
				return "", nil, err
			}
		}
	}
	cfg, err := config.LoadConfig(path)
	if err != nil {
		return "", nil, err
	}
	if err := config.ResolveProviderKeyPath(cfg, path); err != nil {
		return "", nil, err
	}
	return path, cfg, nil
}

func buildProviderTab(w fyne.Window, s *guiState) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Provider Control", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	countryEntry := widget.NewEntry()
	countryEntry.SetText(s.cfg.Provider.Country)
	priceEntry := widget.NewEntry()
	priceEntry.SetText(fmt.Sprintf("%d", s.cfg.Provider.Price))
	bwEntry := widget.NewEntry()
	bwEntry.SetText(s.cfg.Provider.BandwidthLimit)
	allowEntry := widget.NewEntry()
	allowEntry.SetText(s.cfg.Provider.AllowlistFile)
	denyEntry := widget.NewEntry()
	denyEntry.SetText(s.cfg.Provider.DenylistFile)
	lifeEntry := widget.NewEntry()
	lifeEntry.SetText(fmt.Sprintf("%d", s.cfg.Provider.CertLifetimeHours))
	rotateEntry := widget.NewEntry()
	rotateEntry.SetText(fmt.Sprintf("%d", s.cfg.Provider.CertRotateBeforeHours))
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Provider key password")

	statusLabel := widget.NewLabel("Status: stopped")

	saveBtn := widget.NewButton("Save Provider Config", func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		price, err := strconv.ParseUint(strings.TrimSpace(priceEntry.Text), 10, 64)
		if err != nil {
			dialog.ShowError(fmt.Errorf("invalid price: %w", err), w)
			return
		}
		life, err := strconv.Atoi(strings.TrimSpace(lifeEntry.Text))
		if err != nil {
			dialog.ShowError(fmt.Errorf("invalid cert lifetime: %w", err), w)
			return
		}
		rotate, err := strconv.Atoi(strings.TrimSpace(rotateEntry.Text))
		if err != nil {
			dialog.ShowError(fmt.Errorf("invalid cert rotate window: %w", err), w)
			return
		}

		s.cfg.Provider.Country = strings.TrimSpace(countryEntry.Text)
		s.cfg.Provider.Price = price
		s.cfg.Provider.BandwidthLimit = strings.TrimSpace(bwEntry.Text)
		s.cfg.Provider.AllowlistFile = strings.TrimSpace(allowEntry.Text)
		s.cfg.Provider.DenylistFile = strings.TrimSpace(denyEntry.Text)
		s.cfg.Provider.CertLifetimeHours = life
		s.cfg.Provider.CertRotateBeforeHours = rotate
		if err := saveConfig(s.cfgPath, s.cfg); err != nil {
			dialog.ShowError(err, w)
			return
		}
		s.appendLog("Saved provider settings.")
	})

	autoLocateBtn := widget.NewButton("Auto-Locate Country", func() {
		loc, err := geoip.AutoLocate()
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		countryEntry.SetText(strings.ToUpper(loc.CountryCode))
		s.appendLog("Detected provider country: " + strings.ToUpper(loc.CountryCode))
	})

	startBtn := widget.NewButton("Start Provider", func() {
		pass := strings.TrimSpace(passwordEntry.Text)
		if pass == "" {
			dialog.ShowInformation("Password required", "Enter provider key password to start provider.", w)
			return
		}
		if err := s.startProvider(pass); err != nil {
			dialog.ShowError(err, w)
			return
		}
		statusLabel.SetText("Status: running")
	})

	stopBtn := widget.NewButton("Stop Provider", func() {
		s.stopProvider()
		statusLabel.SetText("Status: stopped")
	})

	rotateKeyBtn := widget.NewButton("Rotate Provider Key", func() {
		pass := strings.TrimSpace(passwordEntry.Text)
		if pass == "" {
			dialog.ShowInformation("Password required", "Enter current provider key password.", w)
			return
		}
		if err := rotateProviderKeyGUI(s.cfg.Provider.PrivateKeyFile, pass); err != nil {
			dialog.ShowError(err, w)
			return
		}
		dialog.ShowInformation("Key Rotated", "Provider key rotated. Re-broadcast your service.", w)
		s.appendLog("Provider key rotated.")
	})

	form := widget.NewForm(
		widget.NewFormItem("Country", countryEntry),
		widget.NewFormItem("Price (sats/session)", priceEntry),
		widget.NewFormItem("Bandwidth Limit", bwEntry),
		widget.NewFormItem("Allowlist File", allowEntry),
		widget.NewFormItem("Denylist File", denyEntry),
		widget.NewFormItem("Cert Lifetime Hours", lifeEntry),
		widget.NewFormItem("Rotate Before Hours", rotateEntry),
		widget.NewFormItem("Key Password", passwordEntry),
	)

	controlRow := container.NewGridWithColumns(4, saveBtn, autoLocateBtn, startBtn, stopBtn)
	secRow := container.NewGridWithColumns(2, rotateKeyBtn, statusLabel)

	return container.NewPadded(container.NewVBox(
		title,
		widget.NewCard("Provider Configuration", "Set pricing, policy, and certificate lifecycle controls", form),
		controlRow,
		secRow,
		buildLogPanel(s),
	))
}

func buildClientTab(w fyne.Window, s *guiState) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Client Discovery & Connect", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	sortSelect := widget.NewSelect([]string{"latency", "price", "country"}, nil)
	sortSelect.SetSelected("latency")
	countryEntry := widget.NewEntry()
	countryEntry.SetPlaceHolder("Country filter e.g. US")
	dryRun := widget.NewCheck("Dry run (no payment, no interface changes)", nil)

	results := widget.NewList(
		func() int {
			s.mu.Lock()
			defer s.mu.Unlock()
			return len(s.scanResults)
		},
		func() fyne.CanvasObject { return widget.NewLabel("result") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			s.mu.Lock()
			defer s.mu.Unlock()
			if i >= len(s.scanResults) {
				o.(*widget.Label).SetText("")
				return
			}
			ep := s.scanResults[i]
			o.(*widget.Label).SetText(
				fmt.Sprintf("[%d] %s | %s:%d | %d sats | %s", i, ep.Country, ep.Endpoint.IP, ep.Endpoint.Port, ep.Endpoint.Price, ep.Latency.Round(time.Millisecond)),
			)
		},
	)
	results.OnSelected = func(id widget.ListItemID) {
		s.mu.Lock()
		s.selectedIdx = id
		s.mu.Unlock()
	}

	scanBtn := widget.NewButton("Scan Providers", func() {
		go func() {
			if err := s.scanProviders(sortSelect.Selected, strings.TrimSpace(countryEntry.Text)); err != nil {
				dialog.ShowError(err, w)
				return
			}
			results.Refresh()
		}()
	})

	connectBtn := widget.NewButton("Connect Selected", func() {
		go func() {
			if err := s.connectSelectedProvider(dryRun.Checked); err != nil {
				dialog.ShowError(err, w)
			}
		}()
	})

	filterRow := container.NewGridWithColumns(4,
		widget.NewLabel("Sort:"),
		sortSelect,
		widget.NewLabel("Country:"),
		countryEntry,
	)
	actionRow := container.NewGridWithColumns(3, scanBtn, connectBtn, dryRun)

	return container.NewPadded(container.NewVBox(
		title,
		widget.NewCard("Filters", "Scan and choose the best provider", container.NewVBox(filterRow, actionRow)),
		widget.NewCard("Provider List", "Latency, price, and country-enriched endpoint table", results),
		buildLogPanel(s),
	))
}

func buildStatusTab(s *guiState) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Network Status", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	configPath := widget.NewLabel("Config Path: " + s.cfgPath)
	providerIface := widget.NewLabel("Provider Interface: " + s.cfg.Provider.InterfaceName + " (" + s.cfg.Provider.TunIP + "/" + s.cfg.Provider.TunSubnet + ")")
	clientIface := widget.NewLabel("Client Interface: " + s.cfg.Client.InterfaceName + " (" + s.cfg.Client.TunIP + "/" + s.cfg.Client.TunSubnet + ")")

	return container.NewPadded(container.NewVBox(
		title,
		widget.NewCard("Interfaces", "Current tunnel interface settings", container.NewVBox(configPath, providerIface, clientIface)),
		buildLogPanel(s),
	))
}

func buildWalletTab(s *guiState) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Wallet & History", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	historyBox := widget.NewMultiLineEntry()
	historyBox.Wrapping = fyne.TextWrapWord
	historyBox.Disable()

	refresh := func() {
		records, err := history.LoadHistory()
		if err != nil {
			historyBox.SetText("Failed to load history: " + err.Error())
			return
		}
		if len(records) == 0 {
			historyBox.SetText("No payment history.")
			return
		}
		sort.Slice(records, func(i, j int) bool { return records[i].Timestamp.After(records[j].Timestamp) })
		var b strings.Builder
		for _, r := range records {
			fmt.Fprintf(&b, "%s | %d sats | %s | %s\n", r.Timestamp.Format(time.RFC3339), r.Amount, r.Provider, r.TxID)
		}
		historyBox.SetText(b.String())
	}
	refresh()
	return container.NewPadded(container.NewVBox(
		title,
		widget.NewButton("Reload History", refresh),
		widget.NewCard("Payment History", "Most recent transactions", historyBox),
	))
}

func buildLogPanel(s *guiState) fyne.CanvasObject {
	logEntry := widget.NewMultiLineEntry()
	logEntry.Bind(s.logs)
	logEntry.Disable()
	logEntry.SetMinRowsVisible(8)
	return widget.NewCard("Activity Log", "Runtime events, errors, and actions", logEntry)
}

func (s *guiState) appendLog(line string) {
	current, _ := s.logs.Get()
	ts := time.Now().Format("15:04:05")
	_ = s.logs.Set(current + fmt.Sprintf("[%s] %s\n", ts, line))
}

func (s *guiState) startProvider(password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.providerRunning {
		return fmt.Errorf("provider already running")
	}
	client := connectRPC(s.cfg.RPC.Host, s.cfg.RPC.User, s.cfg.RPC.Pass)
	authManager := auth.NewAuthManager()

	providerKey, err := getOrCreateProviderKey(s.cfg.Provider.PrivateKeyFile, password)
	if err != nil {
		client.Shutdown()
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.providerCancel = cancel
	s.providerRunning = true

	announceIP, announcePort, natCleanup, err := determineAnnounceDetails(ctx, &s.cfg.Provider)
	if err != nil {
		client.Shutdown()
		cancel()
		s.providerRunning = false
		s.providerCancel = nil
		return err
	}
	endpoint := buildProviderEndpoint(s.cfg.Provider.Price, announceIP, announcePort, providerKey)

	go func() {
		defer client.Shutdown()
		defer func() {
			s.mu.Lock()
			s.providerRunning = false
			s.providerCancel = nil
			s.mu.Unlock()
			if natCleanup != nil {
				natCleanup()
			}
		}()
		s.appendLog("Provider started.")

		go func() {
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()
			if err := blockchain.AnnounceService(client, endpoint); err != nil {
				s.appendLog("Announcement failed: " + err.Error())
			}
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := blockchain.AnnounceService(client, endpoint); err != nil {
						s.appendLog("Re-announcement failed: " + err.Error())
					}
				}
			}
		}()

		go blockchain.MonitorPayments(ctx, client, authManager, s.cfg.Provider.Price)
		go blockchain.StartEchoServer(ctx, s.cfg.Provider.ListenPort)
		tunnel.StartProviderServer(ctx, &s.cfg.Provider, providerKey, authManager)
	}()
	return nil
}

func (s *guiState) stopProvider() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.providerCancel != nil {
		s.providerCancel()
		s.providerCancel = nil
	}
	s.providerRunning = false
	s.appendLog("Provider stop requested.")
}

func (s *guiState) scanProviders(sortBy, country string) error {
	client := connectRPC(s.cfg.RPC.Host, s.cfg.RPC.User, s.cfg.RPC.Pass)
	defer client.Shutdown()

	results, _, err := blockchain.ScanForVPNs(client, 0)
	if err != nil {
		return err
	}
	enriched := geoip.EnrichEndpoints(results)
	var filtered []*geoip.EnrichedVPNEndpoint
	for _, ep := range enriched {
		if country == "" || strings.EqualFold(country, ep.Country) {
			filtered = append(filtered, ep)
		}
	}
	switch sortBy {
	case "price":
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].Endpoint.Price < filtered[j].Endpoint.Price })
	case "country":
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].Country < filtered[j].Country })
	default:
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].Latency < filtered[j].Latency })
	}

	s.mu.Lock()
	s.scanResults = filtered
	s.selectedIdx = -1
	s.mu.Unlock()
	s.appendLog(fmt.Sprintf("Scan complete: %d provider(s) found.", len(filtered)))
	return nil
}

func (s *guiState) connectSelectedProvider(dryRun bool) error {
	s.mu.Lock()
	idx := s.selectedIdx
	if idx < 0 || idx >= len(s.scanResults) {
		s.mu.Unlock()
		return fmt.Errorf("no provider selected")
	}
	selected := s.scanResults[idx]
	s.mu.Unlock()

	client := connectRPC(s.cfg.RPC.Host, s.cfg.RPC.User, s.cfg.RPC.Pass)
	defer client.Shutdown()

	genesisHash, err := client.GetBlockHash(0)
	if err != nil {
		return err
	}
	chainParams := detectChain(genesisHash)

	localKey, err := btcec.NewPrivateKey()
	if err != nil {
		return err
	}
	providerAddr, err := blockchain.GetProviderPaymentAddress(client, selected.TxID, chainParams)
	if err != nil {
		return err
	}

	if !dryRun {
		if _, err := blockchain.SendPayment(client, providerAddr, selected.Endpoint.Price, localKey.PubKey()); err != nil {
			return err
		}
	}

	endpointAddr := fmt.Sprintf("%s:%d", selected.Endpoint.IP, selected.Endpoint.Port)
	s.appendLog("Connecting to " + endpointAddr)
	if dryRun {
		s.appendLog("Dry-run connect completed.")
		return nil
	}
	return tunnel.ConnectToProvider(context.Background(), &s.cfg.Client, localKey, selected.Endpoint.PublicKey, endpointAddr)
}

func saveConfig(path string, cfg *config.Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
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
		log.Fatalf("RPC connection failed: %v", err)
	}
	return client
}

func getOrCreateProviderKey(keyPath, password string) (*btcec.PrivateKey, error) {
	if _, err := os.Stat(keyPath); err == nil {
		return crypto.LoadAndDecryptKey(keyPath, []byte(password))
	}
	return crypto.GenerateAndEncryptKey(keyPath, []byte(password))
}

func rotateProviderKeyGUI(keyPath, currentPassword string) error {
	if _, err := os.Stat(keyPath); err != nil {
		return err
	}
	if _, err := crypto.LoadAndDecryptKey(keyPath, []byte(currentPassword)); err != nil {
		return fmt.Errorf("current password invalid: %w", err)
	}
	backup := keyPath + ".bak-" + time.Now().UTC().Format("20060102-150405")
	if err := os.Rename(keyPath, backup); err != nil {
		return err
	}
	if _, err := crypto.GenerateAndEncryptKey(keyPath, []byte(currentPassword)); err != nil {
		_ = os.Rename(backup, keyPath)
		return err
	}
	return nil
}

func determineAnnounceDetails(ctx context.Context, cfg *config.ProviderConfig) (net.IP, int, func(), error) {
	announcePort := cfg.ListenPort
	if cfg.EnableNAT {
		mapping, err := nat.DiscoverAndMapPorts(ctx, cfg.ListenPort, cfg.ListenPort)
		if err == nil {
			return mapping.ExternalIP, mapping.TCPPort, mapping.Cleanup, nil
		}
	}
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
		return nil, 0, nil, err
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

func detectChain(genesisHash *chainhash.Hash) *chaincfg.Params {
	switch *genesisHash {
	case *chaincfg.MainNetParams.GenesisHash:
		return &chaincfg.MainNetParams
	case *chaincfg.TestNet3Params.GenesisHash:
		return &chaincfg.TestNet3Params
	case *chaincfg.RegressionNetParams.GenesisHash:
		return &chaincfg.RegressionNetParams
	case *chaincfg.SimNetParams.GenesisHash:
		return &chaincfg.SimNetParams
	default:
		return &chaincfg.MainNetParams
	}
}
