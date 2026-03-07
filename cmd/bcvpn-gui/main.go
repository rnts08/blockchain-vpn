package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
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

	cfgPath  string
	cfg      *config.Config
	firstRun bool

	logs binding.String

	providerRunning bool
	providerCancel  context.CancelFunc
	providerDone    chan struct{}

	scanResults []*geoip.EnrichedVPNEndpoint
	selectedIdx int
}

const setupMarkerFile = "setup-complete"

func buildMainTabs(w fyne.Window, state *guiState) fyne.CanvasObject {
	providerTab := buildProviderTab(w, state)
	clientTab := buildClientTab(w, state)
	statusTab := buildStatusTab(state)
	settingsTab := buildSettingsTab(w, state)
	walletTab := buildWalletTab(state)

	tabs := container.NewAppTabs(
		container.NewTabItem("Provider Mode", providerTab),
		container.NewTabItem("Client Mode", clientTab),
		container.NewTabItem("Network Status", statusTab),
		container.NewTabItem("Settings", settingsTab),
		container.NewTabItem("Wallet", walletTab),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	bg := canvas.NewRectangle(color.NRGBA{R: 236, G: 235, B: 226, A: 255})
	return container.NewMax(bg, tabs)
}

func main() {
	a := app.NewWithID("com.blockchainvpn.gui")
	a.Settings().SetTheme(&guiTheme{})

	w := a.NewWindow(fmt.Sprintf("BlockchainVPN %s", version.Version))
	w.Resize(fyne.NewSize(1280, 860))

	state, err := initState()
	if err != nil {
		dialog.ShowError(fmt.Errorf("initialization failed: %w", err), w)
		return
	}
	logFormat := strings.TrimSpace(state.cfg.Logging.Format)
	logLevel := strings.TrimSpace(state.cfg.Logging.Level)
	if env := strings.TrimSpace(os.Getenv("BCVPN_LOG_FORMAT")); env != "" {
		logFormat = env
	}
	if env := strings.TrimSpace(os.Getenv("BCVPN_LOG_LEVEL")); env != "" {
		logLevel = env
	}
	obs.ConfigureLogging(logFormat, logLevel, "bcvpn-gui")
	_ = tunnel.RecoverPendingNetworkState()

	if state.firstRun {
		w.SetContent(buildSetupWizard(w, state))
	} else {
		w.SetContent(buildMainTabs(w, state))
	}

	w.SetCloseIntercept(func() {
		state.stopProvider()
		w.Close()
	})

	w.ShowAndRun()
}

func initState() (*guiState, error) {
	cfgPath, cfg, generatedDefault, err := loadConfigWithFallback()
	if err != nil {
		return nil, err
	}
	firstRun := generatedDefault
	if setupDone, err := hasCompletedSetup(); err == nil {
		firstRun = firstRun || !setupDone
	}
	logs := binding.NewString()
	_ = logs.Set("GUI ready.\n")

	return &guiState{
		cfgPath:     cfgPath,
		cfg:         cfg,
		firstRun:    firstRun,
		logs:        logs,
		selectedIdx: -1,
	}, nil
}

func loadConfigWithFallback() (string, *config.Config, bool, error) {
	defaultPath, err := config.DefaultConfigPath()
	if err != nil {
		return "", nil, false, err
	}
	path := defaultPath
	generatedDefault := false
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if _, legacyErr := os.Stat("config.json"); legacyErr == nil {
			path = "config.json"
		} else {
			if err := config.GenerateDefaultConfig(defaultPath); err != nil {
				return "", nil, false, err
			}
			generatedDefault = true
		}
	}
	cfg, err := config.LoadConfig(path)
	if err != nil {
		return "", nil, false, err
	}
	if err := config.ResolveProviderKeyPath(cfg, path); err != nil {
		return "", nil, false, err
	}
	return path, cfg, generatedDefault, nil
}

func hasCompletedSetup() (bool, error) {
	dir, err := config.AppConfigDir()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(filepath.Join(dir, setupMarkerFile))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func markSetupCompleted() error {
	dir, err := config.AppConfigDir()
	if err != nil {
		return err
	}
	p := filepath.Join(dir, setupMarkerFile)
	return util.WriteFileAtomic(p, []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0o644)
}

func buildSetupWizard(w fyne.Window, s *guiState) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("First-Run Setup Wizard", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	desc := widget.NewLabel("Complete these checks to enable click-and-run provider/client operation.")
	elevHint := widget.NewLabel("Elevation: " + elevationHint())
	status := widget.NewMultiLineEntry()
	status.Disable()
	status.SetMinRowsVisible(8)
	status.SetText("Welcome to BlockchainVPN setup.\n")

	configDone := false
	rpcDone := false
	keyDone := false
	privDone := false

	appendStatus := func(line string) {
		status.SetText(status.Text + line + "\n")
	}

	configBtn := widget.NewButton("1) Ensure Config", func() {
		if _, err := os.Stat(s.cfgPath); os.IsNotExist(err) {
			if err := config.GenerateDefaultConfig(s.cfgPath); err != nil {
				dialog.ShowError(err, w)
				return
			}
		}
		cfg, err := config.LoadConfig(s.cfgPath)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		if err := config.ResolveProviderKeyPath(cfg, s.cfgPath); err != nil {
			dialog.ShowError(err, w)
			return
		}
		s.mu.Lock()
		s.cfg = cfg
		s.mu.Unlock()
		configDone = true
		appendStatus("[OK] Config ready: " + s.cfgPath)
	})

	rpcBtn := widget.NewButton("2) Check RPC Connectivity", func() {
		s.mu.Lock()
		cfg := s.cfg
		s.mu.Unlock()
		if err := checkRPCConnectivity(cfg); err != nil {
			dialog.ShowError(err, w)
			appendStatus("[FAIL] RPC connectivity failed: " + err.Error())
			return
		}
		rpcDone = true
		appendStatus("[OK] RPC connectivity check passed")
	})

	keyPassword := widget.NewPasswordEntry()
	keyPassword.SetPlaceHolder("Provider key password")
	keyBtn := widget.NewButton("3) Create/Unlock Provider Key", func() {
		pass := strings.TrimSpace(keyPassword.Text)
		s.mu.Lock()
		cfgCopy := s.cfg
		keyPath := cfgCopy.Provider.PrivateKeyFile
		fileMode := requiresPasswordForKeyStorage(cfgCopy.Security.KeyStorageMode)
		s.mu.Unlock()
		if fileMode && pass == "" {
			dialog.ShowInformation("Password required", "Enter provider key password to create/unlock key.", w)
			return
		}
		if _, err := getOrCreateProviderKey(cfgCopy, keyPath, pass); err != nil {
			dialog.ShowError(err, w)
			appendStatus("[FAIL] Provider key setup failed: " + err.Error())
			return
		}
		keyDone = true
		appendStatus("[OK] Provider key ready")
	})

	privBtn := widget.NewButton("4) Check Networking Privileges", func() {
		if err := tunnel.EnsureElevatedPrivileges(); err != nil {
			appendStatus("[FAIL] Privilege check failed: " + err.Error())
			dialog.ShowError(err, w)
			return
		}
		privDone = true
		appendStatus("[OK] Networking privileges confirmed")
	})

	relaunchBtn := widget.NewButton("Relaunch Elevated", func() {
		if err := relaunchElevated(); err != nil {
			dialog.ShowError(err, w)
			appendStatus("[FAIL] Elevation relaunch failed: " + err.Error())
			return
		}
		appendStatus("[INFO] Relaunch command issued; closing current process.")
		w.Close()
	})
	if !canRelaunchElevated() {
		relaunchBtn.Disable()
	}

	finishBtn := widget.NewButton("Finish Setup", func() {
		if !configDone || !rpcDone || !keyDone || !privDone {
			dialog.ShowInformation("Setup Incomplete", "Complete all setup checks before finishing.", w)
			return
		}
		if err := markSetupCompleted(); err != nil {
			dialog.ShowError(err, w)
			return
		}
		s.firstRun = false
		appendStatus("[OK] Setup completed. Loading main UI...")
		w.SetContent(buildMainTabs(w, s))
	})

	buttonRow1 := container.NewGridWithColumns(2, configBtn, rpcBtn)
	buttonRow2 := container.NewGridWithColumns(2, keyBtn, privBtn)
	buttonRow3 := container.NewGridWithColumns(2, relaunchBtn, finishBtn)

	return container.NewPadded(container.NewVBox(
		title,
		desc,
		elevHint,
		widget.NewCard("Step 3 Input", "Provider key password", keyPassword),
		buttonRow1,
		buttonRow2,
		buttonRow3,
		widget.NewCard("Setup Status", "", status),
	))
}

func checkRPCConnectivity(cfg *config.Config) error {
	connCfg := &rpcclient.ConnConfig{
		Host:         cfg.RPC.Host,
		User:         cfg.RPC.User,
		Pass:         cfg.RPC.Pass,
		HTTPPostMode: true,
		DisableTLS:   true,
	}
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		return fmt.Errorf("rpc connect failed: %w", err)
	}
	defer client.Shutdown()
	if _, err := client.GetBlockCount(); err != nil {
		return fmt.Errorf("rpc blockcount check failed: %w", err)
	}
	return nil
}

func buildProviderTab(w fyne.Window, s *guiState) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Provider Control", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	ifaceEntry := widget.NewEntry()
	ifaceEntry.SetText(s.cfg.Provider.InterfaceName)
	listenPortEntry := widget.NewEntry()
	listenPortEntry.SetText(fmt.Sprintf("%d", s.cfg.Provider.ListenPort))
	announceIPEntry := widget.NewEntry()
	announceIPEntry.SetText(s.cfg.Provider.AnnounceIP)
	countryEntry := widget.NewEntry()
	countryEntry.SetText(s.cfg.Provider.Country)
	priceEntry := widget.NewEntry()
	priceEntry.SetText(fmt.Sprintf("%d", s.cfg.Provider.Price))
	maxConsumersEntry := widget.NewEntry()
	maxConsumersEntry.SetText(fmt.Sprintf("%d", s.cfg.Provider.MaxConsumers))
	bwEntry := widget.NewEntry()
	bwEntry.SetText(s.cfg.Provider.BandwidthLimit)
	tunIPEntry := widget.NewEntry()
	tunIPEntry.SetText(s.cfg.Provider.TunIP)
	tunSubnetEntry := widget.NewEntry()
	tunSubnetEntry.SetText(s.cfg.Provider.TunSubnet)
	natEnabled := widget.NewCheck("Enable NAT Traversal (UPnP/NAT-PMP)", nil)
	natEnabled.SetChecked(s.cfg.Provider.EnableNAT)
	egressNATEnabled := widget.NewCheck("Enable Provider Egress NAT", nil)
	egressNATEnabled.SetChecked(s.cfg.Provider.EnableEgressNAT)
	natOutboundEntry := widget.NewEntry()
	natOutboundEntry.SetText(s.cfg.Provider.NATOutboundInterface)
	isolationSelect := widget.NewSelect([]string{"none", "sandbox"}, nil)
	if strings.TrimSpace(s.cfg.Provider.IsolationMode) == "" {
		isolationSelect.SetSelected("none")
	} else {
		isolationSelect.SetSelected(s.cfg.Provider.IsolationMode)
	}
	allowEntry := widget.NewEntry()
	allowEntry.SetText(s.cfg.Provider.AllowlistFile)
	denyEntry := widget.NewEntry()
	denyEntry.SetText(s.cfg.Provider.DenylistFile)
	lifeEntry := widget.NewEntry()
	lifeEntry.SetText(fmt.Sprintf("%d", s.cfg.Provider.CertLifetimeHours))
	rotateEntry := widget.NewEntry()
	rotateEntry.SetText(fmt.Sprintf("%d", s.cfg.Provider.CertRotateBeforeHours))
	healthEnabled := widget.NewCheck("Enable Health Checks", nil)
	healthEnabled.SetChecked(s.cfg.Provider.HealthCheckEnabled)
	healthIntervalEntry := widget.NewEntry()
	healthIntervalEntry.SetText(s.cfg.Provider.HealthCheckInterval)
	metricsAddrEntry := widget.NewEntry()
	metricsAddrEntry.SetText(s.cfg.Provider.MetricsListenAddr)
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Provider key password (file mode only)")

	statusLabel := widget.NewLabel("Status: stopped")

	saveBtn := widget.NewButton("Save Provider Config", func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		price, err := strconv.ParseUint(strings.TrimSpace(priceEntry.Text), 10, 64)
		if err != nil {
			dialog.ShowError(fmt.Errorf("invalid price: %w", err), w)
			return
		}
		maxConsumers, err := strconv.Atoi(strings.TrimSpace(maxConsumersEntry.Text))
		if err != nil || maxConsumers < 0 {
			dialog.ShowError(fmt.Errorf("invalid max consumers: must be a non-negative integer"), w)
			return
		}
		listenPort, err := strconv.Atoi(strings.TrimSpace(listenPortEntry.Text))
		if err != nil || listenPort <= 0 || listenPort > 65535 {
			dialog.ShowError(fmt.Errorf("invalid listen port"), w)
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
		announceIP := strings.TrimSpace(announceIPEntry.Text)
		if announceIP != "" && net.ParseIP(announceIP) == nil {
			dialog.ShowError(fmt.Errorf("invalid announce IP"), w)
			return
		}
		if net.ParseIP(strings.TrimSpace(tunIPEntry.Text)) == nil {
			dialog.ShowError(fmt.Errorf("invalid provider TUN IP"), w)
			return
		}
		if _, err := strconv.Atoi(strings.TrimSpace(tunSubnetEntry.Text)); err != nil {
			dialog.ShowError(fmt.Errorf("invalid provider TUN subnet"), w)
			return
		}
		if _, err := time.ParseDuration(strings.TrimSpace(healthIntervalEntry.Text)); err != nil {
			dialog.ShowError(fmt.Errorf("invalid health check interval: %w", err), w)
			return
		}

		s.cfg.Provider.InterfaceName = strings.TrimSpace(ifaceEntry.Text)
		s.cfg.Provider.ListenPort = listenPort
		s.cfg.Provider.AnnounceIP = announceIP
		s.cfg.Provider.Country = strings.TrimSpace(countryEntry.Text)
		s.cfg.Provider.Price = price
		s.cfg.Provider.MaxConsumers = maxConsumers
		s.cfg.Provider.BandwidthLimit = strings.TrimSpace(bwEntry.Text)
		s.cfg.Provider.TunIP = strings.TrimSpace(tunIPEntry.Text)
		s.cfg.Provider.TunSubnet = strings.TrimSpace(tunSubnetEntry.Text)
		s.cfg.Provider.EnableNAT = natEnabled.Checked
		s.cfg.Provider.EnableEgressNAT = egressNATEnabled.Checked
		s.cfg.Provider.NATOutboundInterface = strings.TrimSpace(natOutboundEntry.Text)
		s.cfg.Provider.IsolationMode = strings.TrimSpace(isolationSelect.Selected)
		s.cfg.Provider.AllowlistFile = strings.TrimSpace(allowEntry.Text)
		s.cfg.Provider.DenylistFile = strings.TrimSpace(denyEntry.Text)
		s.cfg.Provider.CertLifetimeHours = life
		s.cfg.Provider.CertRotateBeforeHours = rotate
		s.cfg.Provider.HealthCheckEnabled = healthEnabled.Checked
		s.cfg.Provider.HealthCheckInterval = strings.TrimSpace(healthIntervalEntry.Text)
		s.cfg.Provider.MetricsListenAddr = strings.TrimSpace(metricsAddrEntry.Text)
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
		if requiresPasswordForKeyStorage(s.cfg.Security.KeyStorageMode) && pass == "" {
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
	rebroadcastBtn := widget.NewButton("Rebroadcast Service", func() {
		pass := strings.TrimSpace(passwordEntry.Text)
		if requiresPasswordForKeyStorage(s.cfg.Security.KeyStorageMode) && pass == "" {
			dialog.ShowInformation("Password required", "Enter provider key password to rebroadcast.", w)
			return
		}
		go func() {
			if err := s.rebroadcastService(pass); err != nil {
				dialog.ShowError(err, w)
				return
			}
			s.appendLog("Service announcement rebroadcasted.")
		}()
	})
	updatePriceBtn := widget.NewButton("Broadcast Price Update", func() {
		pass := strings.TrimSpace(passwordEntry.Text)
		if requiresPasswordForKeyStorage(s.cfg.Security.KeyStorageMode) && pass == "" {
			dialog.ShowInformation("Password required", "Enter provider key password to broadcast price update.", w)
			return
		}
		go func() {
			if err := s.broadcastPriceUpdate(pass); err != nil {
				dialog.ShowError(err, w)
				return
			}
			s.appendLog(fmt.Sprintf("Price update broadcasted: %d sats/session", s.cfg.Provider.Price))
		}()
	})

	rotateKeyBtn := widget.NewButton("Rotate Provider Key", func() {
		pass := strings.TrimSpace(passwordEntry.Text)
		if requiresPasswordForKeyStorage(s.cfg.Security.KeyStorageMode) && pass == "" {
			dialog.ShowInformation("Password required", "Enter current provider key password.", w)
			return
		}
		if err := rotateProviderKeyGUI(s.cfg, s.cfg.Provider.PrivateKeyFile, pass); err != nil {
			dialog.ShowError(err, w)
			return
		}
		dialog.ShowInformation("Key Rotated", "Provider key rotated. Re-broadcast your service.", w)
		s.appendLog("Provider key rotated.")
	})

	form := widget.NewForm(
		widget.NewFormItem("Interface Name", ifaceEntry),
		widget.NewFormItem("Listen Port", listenPortEntry),
		widget.NewFormItem("Announce IP (optional)", announceIPEntry),
		widget.NewFormItem("Country", countryEntry),
		widget.NewFormItem("Price (sats/session)", priceEntry),
		widget.NewFormItem("Max Consumers (0=unlimited)", maxConsumersEntry),
		widget.NewFormItem("Bandwidth Limit", bwEntry),
		widget.NewFormItem("Provider TUN IP", tunIPEntry),
		widget.NewFormItem("Provider TUN Subnet", tunSubnetEntry),
		widget.NewFormItem("NAT Traversal", natEnabled),
		widget.NewFormItem("Provider Egress NAT", egressNATEnabled),
		widget.NewFormItem("NAT Outbound Interface", natOutboundEntry),
		widget.NewFormItem("Isolation Mode", isolationSelect),
		widget.NewFormItem("Allowlist File", allowEntry),
		widget.NewFormItem("Denylist File", denyEntry),
		widget.NewFormItem("Cert Lifetime Hours", lifeEntry),
		widget.NewFormItem("Rotate Before Hours", rotateEntry),
		widget.NewFormItem("Health Checks", healthEnabled),
		widget.NewFormItem("Health Check Interval", healthIntervalEntry),
		widget.NewFormItem("Metrics Listen Addr", metricsAddrEntry),
		widget.NewFormItem("Key Password", passwordEntry),
	)

	controlRow := container.NewGridWithColumns(6, saveBtn, autoLocateBtn, startBtn, stopBtn, rebroadcastBtn, updatePriceBtn)
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
	sortSelect := widget.NewSelect([]string{"latency", "price", "country", "bandwidth", "capacity", "score"}, nil)
	sortSelect.SetSelected("latency")
	countryEntry := widget.NewEntry()
	countryEntry.SetPlaceHolder("Country filter e.g. US")
	maxPriceEntry := widget.NewEntry()
	maxPriceEntry.SetPlaceHolder("Max price sats (optional)")
	minBwEntry := widget.NewEntry()
	minBwEntry.SetPlaceHolder("Min bandwidth Kbps")
	maxLatencyEntry := widget.NewEntry()
	maxLatencyEntry.SetPlaceHolder("Max latency ms")
	minSlotsEntry := widget.NewEntry()
	minSlotsEntry.SetPlaceHolder("Min available slots")
	clientIfaceEntry := widget.NewEntry()
	clientIfaceEntry.SetText(s.cfg.Client.InterfaceName)
	clientTunIPEntry := widget.NewEntry()
	clientTunIPEntry.SetText(s.cfg.Client.TunIP)
	clientTunSubnetEntry := widget.NewEntry()
	clientTunSubnetEntry.SetText(s.cfg.Client.TunSubnet)
	clientMetricsAddrEntry := widget.NewEntry()
	clientMetricsAddrEntry.SetText(s.cfg.Client.MetricsListenAddr)
	clientKillSwitch := widget.NewCheck("Enable Kill Switch", nil)
	clientKillSwitch.SetChecked(s.cfg.Client.EnableKillSwitch)
	clientStrictVerify := widget.NewCheck("Strict Verification", nil)
	clientStrictVerify.SetChecked(s.cfg.Client.StrictVerification)
	clientThroughputVerify := widget.NewCheck("Verify Throughput After Connect", nil)
	clientThroughputVerify.SetChecked(s.cfg.Client.VerifyThroughputAfterSetup)
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
				fmt.Sprintf(
					"[%d] %s | %s:%d | %d sats | %s | %d Kbps | cap=%s | score=%.2f",
					i,
					effectiveCountryGUI(ep),
					ep.Endpoint.IP,
					ep.Endpoint.Port,
					ep.Endpoint.Price,
					ep.Latency.Round(time.Millisecond),
					ep.AdvertisedBandwidthKB,
					displayCapacityGUI(ep),
					computeProviderScoreGUI(ep),
				),
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
			maxPrice, err := parseUint64Optional(maxPriceEntry.Text)
			if err != nil {
				dialog.ShowError(fmt.Errorf("invalid max price: %w", err), w)
				return
			}
			minBW, err := parseUint32Optional(minBwEntry.Text)
			if err != nil {
				dialog.ShowError(fmt.Errorf("invalid min bandwidth: %w", err), w)
				return
			}
			maxLatencyMS, err := parseIntOptional(maxLatencyEntry.Text)
			if err != nil {
				dialog.ShowError(fmt.Errorf("invalid max latency: %w", err), w)
				return
			}
			minSlots, err := parseIntOptional(minSlotsEntry.Text)
			if err != nil {
				dialog.ShowError(fmt.Errorf("invalid min slots: %w", err), w)
				return
			}
			if err := s.scanProviders(sortSelect.Selected, strings.TrimSpace(countryEntry.Text), maxPrice, minBW, time.Duration(maxLatencyMS)*time.Millisecond, minSlots); err != nil {
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
	saveClientBtn := widget.NewButton("Save Client Settings", func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		if net.ParseIP(strings.TrimSpace(clientTunIPEntry.Text)) == nil {
			dialog.ShowError(fmt.Errorf("invalid client TUN IP"), w)
			return
		}
		if _, err := strconv.Atoi(strings.TrimSpace(clientTunSubnetEntry.Text)); err != nil {
			dialog.ShowError(fmt.Errorf("invalid client TUN subnet"), w)
			return
		}

		s.cfg.Client.InterfaceName = strings.TrimSpace(clientIfaceEntry.Text)
		s.cfg.Client.TunIP = strings.TrimSpace(clientTunIPEntry.Text)
		s.cfg.Client.TunSubnet = strings.TrimSpace(clientTunSubnetEntry.Text)
		s.cfg.Client.MetricsListenAddr = strings.TrimSpace(clientMetricsAddrEntry.Text)
		s.cfg.Client.EnableKillSwitch = clientKillSwitch.Checked
		s.cfg.Client.StrictVerification = clientStrictVerify.Checked
		s.cfg.Client.VerifyThroughputAfterSetup = clientThroughputVerify.Checked
		if err := saveConfig(s.cfgPath, s.cfg); err != nil {
			dialog.ShowError(err, w)
			return
		}
		s.appendLog("Saved client settings.")
	})

	filterRow := container.NewGridWithColumns(12,
		widget.NewLabel("Sort:"),
		sortSelect,
		widget.NewLabel("Country:"),
		countryEntry,
		widget.NewLabel("Max Price:"),
		maxPriceEntry,
		widget.NewLabel("Min BW:"),
		minBwEntry,
		widget.NewLabel("Max Latency:"),
		maxLatencyEntry,
		widget.NewLabel("Min Slots:"),
		minSlotsEntry,
	)
	settingsRow := container.NewGridWithColumns(
		8,
		widget.NewLabel("Interface"),
		clientIfaceEntry,
		widget.NewLabel("TUN IP"),
		clientTunIPEntry,
		widget.NewLabel("Subnet"),
		clientTunSubnetEntry,
		widget.NewLabel("Metrics"),
		clientMetricsAddrEntry,
	)
	actionRow := container.NewGridWithColumns(5, scanBtn, connectBtn, saveClientBtn, clientKillSwitch, dryRun)
	securityRow := container.NewGridWithColumns(2, clientStrictVerify, clientThroughputVerify)

	return container.NewPadded(container.NewVBox(
		title,
		widget.NewCard("Filters", "Scan and choose the best provider", container.NewVBox(filterRow, settingsRow, actionRow, securityRow)),
		widget.NewCard("Provider List", "Latency, price, and country-enriched endpoint table", results),
		buildLogPanel(s),
	))
}

func buildStatusTab(s *guiState) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Network Status", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	versionLabel := widget.NewLabel("Version: " + version.String())
	configPath := widget.NewLabel("Config Path: " + s.cfgPath)
	providerIface := widget.NewLabel("Provider Interface: " + s.cfg.Provider.InterfaceName + " (" + s.cfg.Provider.TunIP + "/" + s.cfg.Provider.TunSubnet + ")")
	clientIface := widget.NewLabel("Client Interface: " + s.cfg.Client.InterfaceName + " (" + s.cfg.Client.TunIP + "/" + s.cfg.Client.TunSubnet + ")")
	clientKill := widget.NewLabel(fmt.Sprintf("Client Kill Switch: %t", s.cfg.Client.EnableKillSwitch))
	privilegeStatus := "Privileges: OK"
	if err := tunnel.EnsureElevatedPrivileges(); err != nil {
		privilegeStatus = "Privileges: " + err.Error()
	}
	privLabel := widget.NewLabel(privilegeStatus)
	metricsBox := widget.NewMultiLineEntry()
	metricsBox.Disable()
	metricsBox.SetMinRowsVisible(8)
	refreshMetrics := func() {
		s.mu.Lock()
		cfg := *s.cfg
		s.mu.Unlock()
		var out strings.Builder
		fmt.Fprintf(&out, "Provider Metrics Addr: %s\n", cfg.Provider.MetricsListenAddr)
		fmt.Fprintf(&out, "Client Metrics Addr: %s\n", cfg.Client.MetricsListenAddr)
		fmt.Fprintf(&out, "Metrics Auth: %t\n\n", strings.TrimSpace(cfg.Security.MetricsAuthToken) != "")
		fmt.Fprintf(&out, "Provider Metrics:\n%s\n\n", fetchMetricsSummary(cfg.Provider.MetricsListenAddr, cfg.Security.MetricsAuthToken))
		fmt.Fprintf(&out, "Client Metrics:\n%s", fetchMetricsSummary(cfg.Client.MetricsListenAddr, cfg.Security.MetricsAuthToken))
		metricsBox.SetText(out.String())
	}
	refreshBtn := widget.NewButton("Refresh Metrics", refreshMetrics)
	refreshMetrics()
	doctorBox := widget.NewMultiLineEntry()
	doctorBox.Disable()
	doctorBox.SetMinRowsVisible(8)
	runDoctor := func() {
		s.mu.Lock()
		cfg := *s.cfg
		s.mu.Unlock()
		doctorBox.SetText(runDoctorChecksGUI(&cfg))
	}
	doctorBtn := widget.NewButton("Run Doctor Checks", runDoctor)
	runDoctor()
	eventsBox := widget.NewMultiLineEntry()
	eventsBox.Disable()
	eventsBox.SetMinRowsVisible(8)
	refreshEvents := func() {
		events := tunnel.GetRecentEvents(200)
		if len(events) == 0 {
			eventsBox.SetText("No runtime events yet.")
			return
		}
		var out strings.Builder
		for _, ev := range events {
			fmt.Fprintf(&out, "%s [%s] %s: %s\n", ev.Time, ev.Role, ev.Type, ev.Detail)
		}
		eventsBox.SetText(out.String())
	}
	eventsBtn := widget.NewButton("Refresh Events", refreshEvents)
	refreshEvents()
	exportDiagBtn := widget.NewButton("Export Diagnostics Bundle", func() {
		s.mu.Lock()
		cfgCopy := *s.cfg
		s.mu.Unlock()
		dir, err := config.AppConfigDir()
		if err != nil {
			s.appendLog("Diagnostics export failed: " + err.Error())
			return
		}
		outPath := filepath.Join(dir, fmt.Sprintf("diagnostics-gui-%s.json", time.Now().UTC().Format("20060102-150405")))
		payload := map[string]any{
			"generated_at": time.Now().UTC().Format(time.RFC3339),
			"version":      version.String(),
			"config_path":  s.cfgPath,
			"events":       tunnel.GetRecentEvents(200),
			"runtime":      tunnel.GetRuntimeMetricsSnapshot(),
		}
		cfgCopy.RPC.Pass = ""
		cfgCopy.Security.MetricsAuthToken = ""
		payload["config"] = cfgCopy
		var out bytes.Buffer
		enc := json.NewEncoder(&out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(payload); err != nil {
			s.appendLog("Diagnostics export failed: " + err.Error())
			return
		}
		if err := util.WriteFileAtomic(outPath, out.Bytes(), 0o644); err != nil {
			s.appendLog("Diagnostics export failed: " + err.Error())
			return
		}
		s.appendLog("Diagnostics exported: " + outPath)
	})

	return container.NewPadded(container.NewVBox(
		title,
		versionLabel,
		widget.NewCard("Interfaces", "Current tunnel interface settings", container.NewVBox(configPath, providerIface, clientIface, clientKill, privLabel)),
		widget.NewCard("Runtime Metrics", "Provider/client runtime metrics endpoint snapshots", container.NewVBox(refreshBtn, metricsBox)),
		widget.NewCard("Doctor", "Config/privilege/tool readiness checks", container.NewVBox(doctorBtn, doctorBox)),
		widget.NewCard("Event Timeline", "Recent runtime session and auth events", container.NewVBox(eventsBtn, eventsBox)),
		exportDiagBtn,
		buildLogPanel(s),
	))
}

func fetchMetricsSummary(addr, token string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "(metrics disabled)"
	}
	req, err := http.NewRequest(http.MethodGet, "http://"+addr+"/metrics.json", nil)
	if err != nil {
		return "request error: " + err.Error()
	}
	if tok := strings.TrimSpace(token); tok != "" {
		req.Header.Set("X-BCVPN-Metrics-Token", tok)
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "request failed: " + err.Error()
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("metrics endpoint returned %s", resp.Status)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "decode failed: " + err.Error()
	}
	providerRunning := payload["provider_running"]
	clientConnected := payload["client_connected"]
	sessions := payload["active_sessions"]
	up := payload["total_up_bytes"]
	down := payload["total_down_bytes"]
	errors := payload["error_count"]
	lastErr := payload["last_error"]
	return fmt.Sprintf(
		"provider_running=%v\nclient_connected=%v\nactive_sessions=%v\ntotal_up_bytes=%v\ntotal_down_bytes=%v\nerror_count=%v\nlast_error=%v",
		providerRunning, clientConnected, sessions, up, down, errors, lastErr,
	)
}

func runDoctorChecksGUI(cfg *config.Config) string {
	var out strings.Builder
	fmt.Fprintf(&out, "BlockchainVPN Doctor\n")
	if err := config.Validate(cfg); err != nil {
		fmt.Fprintf(&out, "- [FAIL] config.validate: %v\n", err)
	} else {
		fmt.Fprintf(&out, "- [OK] config.validate\n")
	}
	resolved, ok, detail := crypto.KeyStorageStatus(cfg.Security.KeyStorageMode)
	if ok {
		fmt.Fprintf(&out, "- [OK] security.keystore: requested=%s resolved=%s (%s)\n", cfg.Security.KeyStorageMode, resolved, detail)
	} else {
		fmt.Fprintf(&out, "- [FAIL] security.keystore: requested=%s resolved=%s (%s)\n", cfg.Security.KeyStorageMode, resolved, detail)
	}
	if err := tunnel.EnsureElevatedPrivileges(); err != nil {
		fmt.Fprintf(&out, "- [FAIL] networking.privileges: %v\n", err)
	} else {
		fmt.Fprintf(&out, "- [OK] networking.privileges\n")
	}
	for _, tool := range requiredNetworkingToolsGUI(runtime.GOOS) {
		if _, err := exec.LookPath(tool); err != nil {
			fmt.Fprintf(&out, "- [FAIL] tool.%s: not found\n", tool)
		} else {
			fmt.Fprintf(&out, "- [OK] tool.%s\n", tool)
		}
	}
	if strings.TrimSpace(cfg.Security.MetricsAuthToken) == "" && (strings.TrimSpace(cfg.Provider.MetricsListenAddr) != "" || strings.TrimSpace(cfg.Client.MetricsListenAddr) != "") {
		fmt.Fprintf(&out, "- [WARN] security.metrics_auth: metrics enabled without auth token\n")
	} else {
		fmt.Fprintf(&out, "- [OK] security.metrics_auth\n")
	}
	return out.String()
}

func requiredNetworkingToolsGUI(goos string) []string {
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

func buildSettingsTab(w fyne.Window, s *guiState) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Global Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	hint := widget.NewLabel("Validation hints: host required, ports 1-65535, valid IP/prefix, valid health_check_interval duration (e.g. 30s).")

	rpcHost := widget.NewEntry()
	rpcHost.SetText(s.cfg.RPC.Host)
	rpcUser := widget.NewEntry()
	rpcUser.SetText(s.cfg.RPC.User)
	rpcPass := widget.NewPasswordEntry()
	rpcPass.SetText(s.cfg.RPC.Pass)
	keyStorageMode := widget.NewSelect([]string{"file", "auto", "keychain", "libsecret", "dpapi"}, nil)
	if strings.TrimSpace(s.cfg.Security.KeyStorageMode) == "" {
		keyStorageMode.SetSelected("file")
	} else {
		keyStorageMode.SetSelected(s.cfg.Security.KeyStorageMode)
	}
	keyStorageService := widget.NewEntry()
	keyStorageService.SetText(s.cfg.Security.KeyStorageService)
	revocationFile := widget.NewEntry()
	revocationFile.SetText(s.cfg.Security.RevocationCacheFile)
	metricsAuthToken := widget.NewPasswordEntry()
	metricsAuthToken.SetText(s.cfg.Security.MetricsAuthToken)
	metricsAuthToken.SetPlaceHolder("Optional token required for /metrics.json")
	tlsMinVersion := widget.NewSelect([]string{"1.3", "1.2"}, nil)
	if strings.TrimSpace(s.cfg.Security.TLSMinVersion) == "" {
		tlsMinVersion.SetSelected("1.3")
	} else {
		tlsMinVersion.SetSelected(s.cfg.Security.TLSMinVersion)
	}
	tlsProfile := widget.NewSelect([]string{"modern", "compat"}, nil)
	if strings.TrimSpace(s.cfg.Security.TLSProfile) == "" {
		tlsProfile.SetSelected("modern")
	} else {
		tlsProfile.SetSelected(s.cfg.Security.TLSProfile)
	}
	logFormat := widget.NewSelect([]string{"text", "json"}, nil)
	if strings.TrimSpace(s.cfg.Logging.Format) == "" {
		logFormat.SetSelected("text")
	} else {
		logFormat.SetSelected(s.cfg.Logging.Format)
	}
	logLevel := widget.NewSelect([]string{"debug", "info", "warn", "error"}, nil)
	if strings.TrimSpace(s.cfg.Logging.Level) == "" {
		logLevel.SetSelected("info")
	} else {
		logLevel.SetSelected(s.cfg.Logging.Level)
	}
	statusOut := widget.NewMultiLineEntry()
	statusOut.Disable()
	statusOut.SetMinRowsVisible(6)
	profilePath := widget.NewEntry()
	profilePath.SetPlaceHolder("Profile path for import/export")

	saveBtn := widget.NewButton("Save + Validate", func() {
		s.mu.Lock()
		s.cfg.RPC.Host = strings.TrimSpace(rpcHost.Text)
		s.cfg.RPC.User = strings.TrimSpace(rpcUser.Text)
		s.cfg.RPC.Pass = strings.TrimSpace(rpcPass.Text)
		s.cfg.Security.KeyStorageMode = strings.TrimSpace(keyStorageMode.Selected)
		s.cfg.Security.KeyStorageService = strings.TrimSpace(keyStorageService.Text)
		s.cfg.Security.RevocationCacheFile = strings.TrimSpace(revocationFile.Text)
		s.cfg.Security.TLSMinVersion = strings.TrimSpace(tlsMinVersion.Selected)
		s.cfg.Security.TLSProfile = strings.TrimSpace(tlsProfile.Selected)
		s.cfg.Security.MetricsAuthToken = strings.TrimSpace(metricsAuthToken.Text)
		s.cfg.Logging.Format = strings.TrimSpace(logFormat.Selected)
		s.cfg.Logging.Level = strings.TrimSpace(logLevel.Selected)
		if err := config.Validate(s.cfg); err != nil {
			s.mu.Unlock()
			dialog.ShowError(fmt.Errorf("config validation failed: %w", err), w)
			return
		}
		if err := saveConfig(s.cfgPath, s.cfg); err != nil {
			s.mu.Unlock()
			dialog.ShowError(err, w)
			return
		}
		s.mu.Unlock()
		statusOut.SetText("Config saved and validated.")
		s.appendLog("Settings saved and validated.")
	})

	validateBtn := widget.NewButton("Validate Current Config", func() {
		s.mu.Lock()
		err := config.Validate(s.cfg)
		s.mu.Unlock()
		if err != nil {
			statusOut.SetText("Validation failed:\n" + err.Error())
			return
		}
		statusOut.SetText("Config is valid.")
	})

	defaultsBtn := widget.NewButton("Apply Defaults For Empty Fields", func() {
		s.mu.Lock()
		applyDefaultConfigValues(s.cfg)
		rpcHost.SetText(s.cfg.RPC.Host)
		rpcUser.SetText(s.cfg.RPC.User)
		rpcPass.SetText(s.cfg.RPC.Pass)
		s.mu.Unlock()
		statusOut.SetText("Applied defaults for empty fields. Review and click Save + Validate.")
	})
	exportBtn := widget.NewButton("Export Profile", func() {
		p := strings.TrimSpace(profilePath.Text)
		if p == "" {
			dialog.ShowError(fmt.Errorf("set a profile path first"), w)
			return
		}
		s.mu.Lock()
		cfgCopy := *s.cfg
		s.mu.Unlock()
		if err := saveConfig(p, &cfgCopy); err != nil {
			dialog.ShowError(err, w)
			return
		}
		statusOut.SetText("Profile exported to: " + p)
	})
	importBtn := widget.NewButton("Import Profile", func() {
		p := strings.TrimSpace(profilePath.Text)
		if p == "" {
			dialog.ShowError(fmt.Errorf("set a profile path first"), w)
			return
		}
		cfgImported, err := config.LoadConfig(p)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		if err := config.Validate(cfgImported); err != nil {
			dialog.ShowError(fmt.Errorf("imported config invalid: %w", err), w)
			return
		}
		if err := config.ResolveProviderKeyPath(cfgImported, p); err != nil {
			dialog.ShowError(err, w)
			return
		}
		s.mu.Lock()
		*s.cfg = *cfgImported
		err = saveConfig(s.cfgPath, s.cfg)
		s.mu.Unlock()
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		statusOut.SetText("Profile imported from: " + p)
		s.appendLog("Imported settings profile.")
	})

	form := widget.NewForm(
		widget.NewFormItem("RPC Host", rpcHost),
		widget.NewFormItem("RPC User", rpcUser),
		widget.NewFormItem("RPC Pass", rpcPass),
		widget.NewFormItem("Key Storage Mode", keyStorageMode),
		widget.NewFormItem("Key Storage Service", keyStorageService),
		widget.NewFormItem("Revocation Cache File", revocationFile),
		widget.NewFormItem("TLS Min Version", tlsMinVersion),
		widget.NewFormItem("TLS Profile", tlsProfile),
		widget.NewFormItem("Metrics Auth Token", metricsAuthToken),
		widget.NewFormItem("Log Format", logFormat),
		widget.NewFormItem("Log Level", logLevel),
	)
	buttons := container.NewGridWithColumns(3, saveBtn, validateBtn, defaultsBtn)
	profileRow := container.NewGridWithColumns(3, widget.NewLabel("Profile Path"), profilePath, container.NewGridWithColumns(2, exportBtn, importBtn))

	return container.NewPadded(container.NewVBox(
		title,
		hint,
		widget.NewCard("RPC", "Global daemon connection settings", form),
		profileRow,
		buttons,
		widget.NewCard("Validation Output", "", statusOut),
		buildLogPanel(s),
	))
}

func applyDefaultConfigValues(cfg *config.Config) {
	if strings.TrimSpace(cfg.RPC.Host) == "" {
		cfg.RPC.Host = "localhost:18443"
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
	if strings.TrimSpace(cfg.Provider.InterfaceName) == "" {
		cfg.Provider.InterfaceName = "bcvpn0"
	}
	if cfg.Provider.ListenPort == 0 {
		cfg.Provider.ListenPort = 51820
	}
	if cfg.Provider.Price == 0 {
		cfg.Provider.Price = 1000
	}
	if strings.TrimSpace(cfg.Provider.BandwidthLimit) == "" {
		cfg.Provider.BandwidthLimit = "10mbit"
	}
	if strings.TrimSpace(cfg.Provider.TunIP) == "" {
		cfg.Provider.TunIP = "10.10.0.1"
	}
	if strings.TrimSpace(cfg.Provider.TunSubnet) == "" {
		cfg.Provider.TunSubnet = "24"
	}
	if strings.TrimSpace(cfg.Provider.HealthCheckInterval) == "" {
		cfg.Provider.HealthCheckInterval = "30s"
	}
	if strings.TrimSpace(cfg.Provider.MetricsListenAddr) == "" {
		cfg.Provider.MetricsListenAddr = ""
	}
	if cfg.Provider.CertLifetimeHours == 0 {
		cfg.Provider.CertLifetimeHours = 720
	}
	if cfg.Provider.CertRotateBeforeHours == 0 {
		cfg.Provider.CertRotateBeforeHours = 24
	}
	if strings.TrimSpace(cfg.Client.InterfaceName) == "" {
		cfg.Client.InterfaceName = "bcvpn1"
	}
	if strings.TrimSpace(cfg.Client.TunIP) == "" {
		cfg.Client.TunIP = "10.10.0.2"
	}
	if strings.TrimSpace(cfg.Client.TunSubnet) == "" {
		cfg.Client.TunSubnet = "24"
	}
	if strings.TrimSpace(cfg.Client.MetricsListenAddr) == "" {
		cfg.Client.MetricsListenAddr = ""
	}
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
	if err := tunnel.EnsureElevatedPrivileges(); err != nil {
		return fmt.Errorf("provider networking setup requires elevated privileges: %w", err)
	}
	client := connectRPC(s.cfg.RPC.Host, s.cfg.RPC.User, s.cfg.RPC.Pass)
	authManager := auth.NewAuthManager()

	providerKey, err := getOrCreateProviderKey(s.cfg, s.cfg.Provider.PrivateKeyFile, password)
	if err != nil {
		client.Shutdown()
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.providerCancel = cancel
	s.providerRunning = true
	s.providerDone = make(chan struct{})

	announceIP, announcePort, natCleanup, err := determineAnnounceDetails(ctx, &s.cfg.Provider)
	if err != nil {
		client.Shutdown()
		cancel()
		s.providerRunning = false
		s.providerCancel = nil
		return err
	}
	endpoint := buildProviderEndpoint(&s.cfg.Provider, announceIP, announcePort, providerKey)

	go func() {
		defer client.Shutdown()
		defer close(s.providerDone)
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
		go func() {
			if err := blockchain.StartEchoServer(ctx, s.cfg.Provider.ListenPort); err != nil {
				s.appendLog("Echo server error: " + err.Error())
			}
		}()
		if err := tunnel.StartProviderServer(ctx, &s.cfg.Provider, &s.cfg.Security, providerKey, authManager); err != nil {
			s.appendLog("Provider server error: " + err.Error())
		}
	}()
	go func() {
		hbTicker := time.NewTicker(5 * time.Minute)
		defer hbTicker.Stop()
		if err := blockchain.AnnounceHeartbeat(client, providerKey.PubKey(), protocol.AvailabilityFlagAvailable); err != nil {
			s.appendLog(fmt.Sprintf("Initial heartbeat broadcast failed: %v", err))
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-hbTicker.C:
				if err := blockchain.AnnounceHeartbeat(client, providerKey.PubKey(), protocol.AvailabilityFlagAvailable); err != nil {
					s.appendLog(fmt.Sprintf("Scheduled heartbeat broadcast failed: %v", err))
				}
			}
		}
	}()
	return nil
}

func (s *guiState) stopProvider() {
	s.mu.Lock()
	done := s.providerDone
	defer s.mu.Unlock()
	if s.providerCancel != nil {
		s.providerCancel()
		s.providerCancel = nil
	}
	s.providerRunning = false
	s.appendLog("Provider stop requested.")
	if done != nil {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			s.appendLog("Provider shutdown timeout reached.")
		}
	}
}

func (s *guiState) rebroadcastService(password string) error {
	s.mu.Lock()
	cfg := *s.cfg
	s.mu.Unlock()

	client := connectRPC(cfg.RPC.Host, cfg.RPC.User, cfg.RPC.Pass)
	defer client.Shutdown()

	key, err := getOrCreateProviderKey(&cfg, cfg.Provider.PrivateKeyFile, password)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	announceIP, announcePort, natCleanup, err := determineAnnounceDetails(ctx, &cfg.Provider)
	if err != nil {
		return err
	}
	if natCleanup != nil {
		defer natCleanup()
	}
	endpoint := buildProviderEndpoint(&cfg.Provider, announceIP, announcePort, key)
	return blockchain.AnnounceService(client, endpoint)
}

func (s *guiState) broadcastPriceUpdate(password string) error {
	s.mu.Lock()
	cfg := *s.cfg
	s.mu.Unlock()

	client := connectRPC(cfg.RPC.Host, cfg.RPC.User, cfg.RPC.Pass)
	defer client.Shutdown()

	key, err := getOrCreateProviderKey(&cfg, cfg.Provider.PrivateKeyFile, password)
	if err != nil {
		return err
	}
	return blockchain.AnnouncePriceUpdate(client, key.PubKey(), cfg.Provider.Price)
}

func (s *guiState) scanProviders(sortBy, country string, maxPrice uint64, minBandwidthKB uint32, maxLatency time.Duration, minSlots int) error {
	client := connectRPC(s.cfg.RPC.Host, s.cfg.RPC.User, s.cfg.RPC.Pass)
	defer client.Shutdown()

	results, _, err := blockchain.ScanForVPNs(client, 0)
	if err != nil {
		return err
	}
	enriched := geoip.EnrichEndpoints(results)
	var filtered []*geoip.EnrichedVPNEndpoint
	for _, ep := range enriched {
		if country != "" && !strings.EqualFold(country, effectiveCountryGUI(ep)) {
			continue
		}
		if maxPrice > 0 && ep.Endpoint.Price > maxPrice {
			continue
		}
		if minBandwidthKB > 0 && ep.AdvertisedBandwidthKB < minBandwidthKB {
			continue
		}
		if maxLatency > 0 && ep.Latency > maxLatency {
			continue
		}
		if minSlots > 0 && effectiveCapacitySlotsGUI(ep) < minSlots {
			continue
		}
		filtered = append(filtered, ep)
	}
	switch sortBy {
	case "price":
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].Endpoint.Price < filtered[j].Endpoint.Price })
	case "country":
		sort.Slice(filtered, func(i, j int) bool { return effectiveCountryGUI(filtered[i]) < effectiveCountryGUI(filtered[j]) })
	case "bandwidth":
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].AdvertisedBandwidthKB > filtered[j].AdvertisedBandwidthKB })
	case "capacity":
		sort.Slice(filtered, func(i, j int) bool {
			return effectiveCapacitySlotsGUI(filtered[i]) > effectiveCapacitySlotsGUI(filtered[j])
		})
	case "score":
		sort.Slice(filtered, func(i, j int) bool {
			return computeProviderScoreGUI(filtered[i]) > computeProviderScoreGUI(filtered[j])
		})
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

func parseUint64Optional(v string) (uint64, error) {
	if strings.TrimSpace(v) == "" {
		return 0, nil
	}
	return strconv.ParseUint(strings.TrimSpace(v), 10, 64)
}

func parseUint32Optional(v string) (uint32, error) {
	if strings.TrimSpace(v) == "" {
		return 0, nil
	}
	n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 32)
	return uint32(n), err
}

func parseIntOptional(v string) (int, error) {
	if strings.TrimSpace(v) == "" {
		return 0, nil
	}
	return strconv.Atoi(strings.TrimSpace(v))
}

func effectiveCountryGUI(ep *geoip.EnrichedVPNEndpoint) string {
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

func effectiveCapacitySlotsGUI(ep *geoip.EnrichedVPNEndpoint) int {
	if ep == nil {
		return 0
	}
	if ep.MaxConsumers == 0 {
		return 1 << 30
	}
	return int(ep.MaxConsumers)
}

func displayCapacityGUI(ep *geoip.EnrichedVPNEndpoint) string {
	if ep == nil || ep.MaxConsumers == 0 {
		return "unlimited"
	}
	return fmt.Sprintf("%d", ep.MaxConsumers)
}

func computeProviderScoreGUI(ep *geoip.EnrichedVPNEndpoint) float64 {
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
	capacity := float64(effectiveCapacitySlotsGUI(ep))
	if capacity > 1e6 {
		capacity = 1000
	}
	countryBoost := 1.0
	if strings.TrimSpace(ep.DeclaredCountry) != "" {
		countryBoost = 1.05
	}
	return countryBoost * ((float64(ep.AdvertisedBandwidthKB) / 1000.0) + capacity) / (price * float64(latencyMS))
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
		if err := tunnel.EnsureElevatedPrivileges(); err != nil {
			return fmt.Errorf("cannot proceed with payment until networking privileges are available: %w", err)
		}
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
	return tunnel.ConnectToProvider(
		context.Background(),
		&s.cfg.Client,
		&s.cfg.Security,
		localKey,
		selected.Endpoint.PublicKey,
		endpointAddr,
		tunnel.ClientSecurityExpectations{
			ExpectedCountry:     selected.Country,
			ExpectedBandwidthKB: selected.AdvertisedBandwidthKB,
		},
	)
}

func saveConfig(path string, cfg *config.Config) error {
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		return err
	}
	return util.WriteFileAtomic(path, out.Bytes(), 0o644)
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

func getOrCreateProviderKey(cfg *config.Config, keyPath, password string) (*btcec.PrivateKey, error) {
	return crypto.LoadOrCreateProviderKey(keyPath, []byte(password), cfg.Security.KeyStorageMode, cfg.Security.KeyStorageService)
}

func rotateProviderKeyGUI(cfg *config.Config, keyPath, currentPassword string) error {
	return crypto.RotateProviderKey(
		keyPath,
		[]byte(currentPassword),
		[]byte(currentPassword),
		cfg.Security.KeyStorageMode,
		cfg.Security.KeyStorageService,
	)
}

func requiresPasswordForKeyStorage(mode string) bool {
	resolved, err := crypto.ResolveKeyStorageMode(mode)
	if err != nil {
		return true
	}
	return resolved == "file"
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

func buildProviderEndpoint(providerCfg *config.ProviderConfig, announceIP net.IP, announcePort int, providerKey *btcec.PrivateKey) *protocol.VPNEndpoint {
	bandwidthKB := parseBandwidthLimitToKbps(providerCfg.BandwidthLimit)
	maxConsumers := uint16(0)
	if providerCfg.MaxConsumers > 0 && providerCfg.MaxConsumers <= 65535 {
		maxConsumers = uint16(providerCfg.MaxConsumers)
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
	return uint32(n * mult)
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
