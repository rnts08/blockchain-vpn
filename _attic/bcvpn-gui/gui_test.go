package main

import (
	"testing"

	"blockchain-vpn/internal/config"

	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestGUICreation(t *testing.T) {
	myApp := test.NewApp()
	state := &guiState{
		cfg:            &config.Config{DemoMode: true},
		logs:           binding.NewString(),
		providerStatus: binding.NewString(),
		isScanning:     binding.NewBool(),
		isConnecting:   binding.NewBool(),
		autoScroll:     binding.NewBool(),
		logSearch:      binding.NewString(),
		fullLogs:       binding.NewString(),
		clientStatus:   binding.NewString(),
		metricsContent: binding.NewString(),
		eventsContent:  binding.NewString(),
		walletBalance:  binding.NewString(),
		countryEntry:   widget.NewSelectEntry(nil),
	}
	_ = state.providerStatus.Set("Stopped")
	_ = state.autoScroll.Set(true)
	applyDefaultConfigValues(state.cfg)

	w := myApp.NewWindow("Test")
	tabs := buildMainTabs(w, state)

	if tabs == nil {
		t.Fatal("tabs container is nil")
	}
}

func TestSettingsTab(t *testing.T) {
	myApp := test.NewApp()
	state := &guiState{
		cfg:            &config.Config{DemoMode: true},
		logs:           binding.NewString(),
		providerStatus: binding.NewString(),
		isScanning:     binding.NewBool(),
		isConnecting:   binding.NewBool(),
		autoScroll:     binding.NewBool(),
		logSearch:      binding.NewString(),
		fullLogs:       binding.NewString(),
		clientStatus:   binding.NewString(),
		metricsContent: binding.NewString(),
		eventsContent:  binding.NewString(),
		walletBalance:  binding.NewString(),
		countryEntry:   widget.NewSelectEntry(nil),
	}
	_ = state.providerStatus.Set("Stopped")
	_ = state.autoScroll.Set(true)
	applyDefaultConfigValues(state.cfg)

	w := myApp.NewWindow("Settings Test")
	settings := buildSettingsTab(w, state)

	if settings == nil {
		t.Fatal("settings tab is nil")
	}
}

func TestWalletTab(t *testing.T) {
	state := &guiState{
		cfg:            &config.Config{DemoMode: true},
		logs:           binding.NewString(),
		providerStatus: binding.NewString(),
		isScanning:     binding.NewBool(),
		isConnecting:   binding.NewBool(),
		autoScroll:     binding.NewBool(),
		logSearch:      binding.NewString(),
		fullLogs:       binding.NewString(),
		clientStatus:   binding.NewString(),
		metricsContent: binding.NewString(),
		eventsContent:  binding.NewString(),
		walletBalance:  binding.NewString(),
		countryEntry:   widget.NewSelectEntry(nil),
	}
	_ = state.providerStatus.Set("Stopped")
	_ = state.autoScroll.Set(true)
	wallet := buildWalletTab(state)
	if wallet == nil {
		t.Fatal("wallet tab is nil")
	}
}
