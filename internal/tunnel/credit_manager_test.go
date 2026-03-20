package tunnel

import (
	"strings"
	"testing"

	"blockchain-vpn/internal/config"
	"github.com/btcsuite/btcd/btcec/v2"
)

func TestSpendingManager_NewSpendingManager(t *testing.T) {
	cfg := &config.ClientConfig{
		AutoRechargeEnabled:    true,
		AutoRechargeMinBalance: 100,
		AutoRechargeThreshold:  500,
		AutoRechargeAmount:     1000,
		SpendingLimitEnabled:   true,
		SpendingLimitSats:      10000,
		SpendingWarningPercent: 80,
		AutoDisconnectOnLimit:  true,
		MaxSessionSpendingSats: 2000,
	}

	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey: %v", err)
	}

	sm := NewSpendingManager(cfg, nil, nil, privKey, privKey.PubKey(), "p2pkh")

	if sm == nil {
		t.Fatal("NewSpendingManager returned nil")
	}
	if !sm.rechargeEnabled {
		t.Error("rechargeEnabled = false, want true")
	}
	if sm.minBalance != 100 {
		t.Errorf("minBalance = %d, want 100", sm.minBalance)
	}
	if sm.rechargeThreshold != 500 {
		t.Errorf("rechargeThreshold = %d, want 500", sm.rechargeThreshold)
	}
	if sm.rechargeAmount != 1000 {
		t.Errorf("rechargeAmount = %d, want 1000", sm.rechargeAmount)
	}
	if !sm.limitEnabled {
		t.Error("limitEnabled = false, want true")
	}
	if sm.totalLimit != 10000 {
		t.Errorf("totalLimit = %d, want 10000", sm.totalLimit)
	}
	if sm.warningPercent != 80 {
		t.Errorf("warningPercent = %d, want 80", sm.warningPercent)
	}
	if !sm.autoDisconnect {
		t.Error("autoDisconnect = false, want true")
	}
	if sm.maxSessionSpend != 2000 {
		t.Errorf("maxSessionSpend = %d, want 2000", sm.maxSessionSpend)
	}
	if sm.balance != 0 {
		t.Errorf("balance = %d, want 0", sm.balance)
	}
	if sm.spentToday != 0 {
		t.Errorf("spentToday = %d, want 0", sm.spentToday)
	}
}

func TestSpendingManager_RecordPayment(t *testing.T) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey: %v", err)
	}

	cfg := &config.ClientConfig{
		AutoRechargeEnabled:    false,
		SpendingLimitEnabled:   true,
		SpendingLimitSats:      1000,
		MaxSessionSpendingSats: 500,
	}

	sm := NewSpendingManager(cfg, nil, nil, privKey, privKey.PubKey(), "p2pkh")
	// Pre-seed balance to avoid balance limit; we're testing spending limits
	sm.balance = 10000

	// First payment should succeed
	if err := sm.RecordPayment(300); err != nil {
		t.Errorf("RecordPayment(300) = %v, want nil", err)
	}
	if sm.spentToday != 300 {
		t.Errorf("spentToday = %d, want 300", sm.spentToday)
	}
	if rem := sm.GetRemainingBudget(); rem != 700 {
		t.Errorf("GetRemainingBudget = %d, want 700", rem)
	}

	// Second payment within limit
	if err := sm.RecordPayment(200); err != nil {
		t.Errorf("RecordPayment(200) = %v, want nil", err)
	}
	if sm.spentToday != 500 {
		t.Errorf("spentToday = %d, want 500", sm.spentToday)
	}

	// Test session spending limit
	t.Run("session spending limit enforced", func(t *testing.T) {
		sm2 := NewSpendingManager(cfg, nil, nil, privKey, privKey.PubKey(), "p2pkh")
		sm2.balance = 10000   // pre-seed
		sm2.SetSessionStart() // captures current spentToday (0)
		// sessionStartSpent = spentToday (0). We try to record 600 > maxSessionSpend=500
		if err := sm2.RecordPayment(600); err == nil {
			t.Error("RecordPayment(600) after SetSessionStart expected error for session limit")
		} else {
			if !strings.Contains(err.Error(), "session spending limit exceeded") {
				t.Errorf("error = %v, missing expected substring", err)
			}
		}
	})

	// Third payment exceeds total limit
	if err := sm.RecordPayment(600); err == nil {
		t.Error("RecordPayment(600) expected error for total limit")
	} else {
		if !strings.Contains(err.Error(), "spending limit exceeded") {
			t.Errorf("error = %v, missing expected substring", err)
		}
	}
}

func TestSpendingManager_ShouldDisconnect(t *testing.T) {
	cfg := &config.ClientConfig{
		SpendingLimitEnabled:  true,
		SpendingLimitSats:     1000,
		AutoDisconnectOnLimit: true,
	}

	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey: %v", err)
	}
	sm := NewSpendingManager(cfg, nil, nil, privKey, privKey.PubKey(), "p2pkh")
	sm.balance = 10000 // pre-seed to allow payments

	// Initially should not disconnect
	if sm.ShouldDisconnect() {
		t.Error("ShouldDisconnect() = true initially, want false")
	}

	// Record spending below limit
	if err := sm.RecordPayment(500); err != nil {
		t.Errorf("RecordPayment(500) = %v", err)
	}
	if sm.ShouldDisconnect() {
		t.Error("ShouldDisconnect() = true after 500, want false")
	}

	// Record spending that reaches limit exactly
	if err := sm.RecordPayment(500); err != nil {
		t.Errorf("RecordPayment(500) = %v", err)
	}
	if !sm.ShouldDisconnect() {
		t.Error("ShouldDisconnect() = false after reaching exactly, want true")
	}

	// If autoDisconnect is false, should not disconnect even at limit
	cfg2 := &config.ClientConfig{
		SpendingLimitEnabled:  true,
		SpendingLimitSats:     1000,
		AutoDisconnectOnLimit: false,
	}
	sm2 := NewSpendingManager(cfg2, nil, nil, privKey, privKey.PubKey(), "p2pkh")
	sm2.balance = 10000
	if err := sm2.RecordPayment(1000); err != nil {
		t.Errorf("RecordPayment(1000) = %v", err)
	}
	if sm2.ShouldDisconnect() {
		t.Error("ShouldDisconnect() = true with autoDisconnect=false, want false")
	}
}

func TestSpendingManager_GetRemainingBudget(t *testing.T) {
	cfg := &config.ClientConfig{
		SpendingLimitEnabled: true,
		SpendingLimitSats:    1000,
	}

	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey: %v", err)
	}
	sm := NewSpendingManager(cfg, nil, nil, privKey, privKey.PubKey(), "p2pkh")
	sm.balance = 10000 // pre-seed

	// Initially full budget
	if got := sm.GetRemainingBudget(); got != 1000 {
		t.Fatalf("GetRemainingBudget() = %d, want 1000", got)
	}

	if err := sm.RecordPayment(300); err != nil {
		t.Errorf("RecordPayment(300) = %v", err)
	}
	if got := sm.GetRemainingBudget(); got != 700 {
		t.Errorf("GetRemainingBudget() = %d, want 700", got)
	}

	if err := sm.RecordPayment(700); err != nil {
		t.Errorf("RecordPayment(700) = %v", err)
	}
	if got := sm.GetRemainingBudget(); got != 0 {
		t.Errorf("GetRemainingBudget() = %d, want 0", got)
	}

	// With limit disabled, should be unlimited (max uint64)
	cfg2 := &config.ClientConfig{SpendingLimitEnabled: false}
	sm2 := NewSpendingManager(cfg2, nil, nil, privKey, privKey.PubKey(), "p2pkh")
	sm2.balance = 10000
	if got := sm2.GetRemainingBudget(); got != ^uint64(0) {
		t.Errorf("GetRemainingBudget (unlimited) = %d, want max uint64", got)
	}
}

func TestSpendingManager_SetSessionStart(t *testing.T) {
	cfg := &config.ClientConfig{}

	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey: %v", err)
	}
	sm := NewSpendingManager(cfg, nil, nil, privKey, privKey.PubKey(), "p2pkh")
	sm.balance = 10000 // pre-seed

	if err := sm.RecordPayment(100); err != nil {
		t.Errorf("RecordPayment(100) = %v", err)
	}
	if err := sm.RecordPayment(100); err != nil {
		t.Errorf("RecordPayment(100) = %v", err)
	} // spentToday = 200

	sm.SetSessionStart()
	if sm.sessionStartSpent != 200 {
		t.Errorf("sessionStartSpent = %d, want 200", sm.sessionStartSpent)
	}
	if sm.warningIssued {
		t.Error("warningIssued = true after SetSessionStart, want false")
	}
}

func TestSpendingManager_AddCredits(t *testing.T) {
	cfg := &config.ClientConfig{}

	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey: %v", err)
	}
	sm := NewSpendingManager(cfg, nil, nil, privKey, privKey.PubKey(), "p2pkh")

	sm.AddCredits(500)
	if sm.balance != 500 {
		t.Errorf("balance = %d, want 500", sm.balance)
	}

	sm.AddCredits(300)
	if sm.balance != 800 {
		t.Errorf("balance = %d, want 800", sm.balance)
	}
}

func TestSpendingManager_Stop(t *testing.T) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey: %v", err)
	}
	sm := NewSpendingManager(&config.ClientConfig{}, nil, nil, privKey, privKey.PubKey(), "p2pkh")

	sm.Stop()
}

func TestSpendingManager_GetBalance(t *testing.T) {
	cfg := &config.ClientConfig{AutoRechargeMinBalance: 100}
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey: %v", err)
	}
	sm := NewSpendingManager(cfg, nil, nil, privKey, privKey.PubKey(), "p2pkh")

	if got := sm.GetBalance(); got != 0 {
		t.Errorf("GetBalance() = %d, want 0", got)
	}

	sm.balance = 500
	if got := sm.GetBalance(); got != 500 {
		t.Errorf("GetBalance() = %d, want 500", got)
	}
}

func TestSpendingManager_PaymentBeforeBalanceCheck(t *testing.T) {
	// Simulate scenario: payment required but balance insufficient
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey: %v", err)
	}

	cfg := &config.ClientConfig{
		AutoRechargeEnabled:  false,
		SpendingLimitEnabled: false, // no spending limit, only balance check
	}
	sm := NewSpendingManager(cfg, nil, nil, privKey, privKey.PubKey(), "p2pkh")
	sm.balance = 50

	// Payment of 100 exceeds balance
	if err := sm.RecordPayment(100); err == nil {
		t.Error("RecordPayment(100) expected error for insufficient balance")
	} else {
		if !strings.Contains(err.Error(), "insufficient balance") {
			t.Errorf("error = %v, missing expected substring", err)
		}
	}

	// Exact balance should succeed
	sm.balance = 100
	if err := sm.RecordPayment(100); err != nil {
		t.Errorf("RecordPayment(100) with exact balance = %v, want nil", err)
	}
	if sm.balance != 0 {
		t.Errorf("balance after payment = %d, want 0", sm.balance)
	}
}
