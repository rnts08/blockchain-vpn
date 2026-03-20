package tunnel

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"blockchain-vpn/internal/blockchain"
	"blockchain-vpn/internal/config"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/rpcclient"
)

// SpendingManager tracks client spending, enforces limits, and handles auto-recharge.
type SpendingManager struct {
	mu sync.Mutex

	// Spending limits
	totalLimit        uint64
	spentToday        uint64
	limitEnabled      bool
	warningPercent    uint32
	warningIssued     bool
	autoDisconnect    bool
	maxSessionSpend   uint64
	sessionStartSpent uint64 // snapshot at session start

	// Auto-recharge (prepaid credit)
	rechargeEnabled   bool
	rechargeThreshold uint64
	rechargeAmount    uint64
	minBalance        uint64
	balance           uint64 // current prepaid balance

	lastRecharge     time.Time
	rechargeInterval time.Duration

	client         *rpcclient.Client
	providerAddr   btcutil.Address
	localKey       *btcec.PrivateKey
	providerPubKey *btcec.PublicKey
	addressType    string
	stopped        chan struct{}
}

// NewSpendingManager creates a SpendingManager from client config.
func NewSpendingManager(cfg *config.ClientConfig, client *rpcclient.Client, providerAddr btcutil.Address, localKey *btcec.PrivateKey, providerPubKey *btcec.PublicKey, addressType string) *SpendingManager {
	sm := &SpendingManager{
		balance:           0,
		minBalance:        cfg.AutoRechargeMinBalance,
		rechargeAmount:    cfg.AutoRechargeAmount,
		rechargeThreshold: cfg.AutoRechargeThreshold,
		rechargeEnabled:   cfg.AutoRechargeEnabled,
		client:            client,
		providerAddr:      providerAddr,
		localKey:          localKey,
		providerPubKey:    providerPubKey,
		addressType:       addressType,
		rechargeInterval:  30 * time.Second,
		stopped:           make(chan struct{}),

		// Spending limits
		totalLimit:      cfg.SpendingLimitSats,
		limitEnabled:    cfg.SpendingLimitEnabled,
		warningPercent:  cfg.SpendingWarningPercent,
		autoDisconnect:  cfg.AutoDisconnectOnLimit,
		maxSessionSpend: cfg.MaxSessionSpendingSats,
	}

	if sm.minBalance == 0 {
		sm.minBalance = 100
	}
	if sm.rechargeAmount == 0 {
		sm.rechargeAmount = 1000
	}
	if sm.rechargeThreshold == 0 {
		sm.rechargeThreshold = 500
	}

	return sm
}

// Start begins the auto-recharge background loop if enabled.
func (sm *SpendingManager) Start(ctx context.Context) {
	if !sm.rechargeEnabled {
		log.Println("Spending manager: auto-recharge disabled")
		return
	}

	log.Printf("Spending manager: started (recharge threshold: %d sats, amount: %d sats, min: %d sats, spending limit: %s)",
		sm.rechargeThreshold, sm.rechargeAmount, sm.minBalance,
		func() string {
			if sm.limitEnabled {
				return fmt.Sprintf("%d sats", sm.totalLimit)
			}
			return "unlimited"
		}(),
	)

	go sm.run(ctx)
}

func (sm *SpendingManager) run(ctx context.Context) {
	ticker := time.NewTicker(sm.rechargeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Spending manager: stopped")
			return
		case <-ticker.C:
			sm.checkAndRecharge(ctx)
		}
	}
}

func (sm *SpendingManager) checkAndRecharge(ctx context.Context) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.balance < sm.rechargeThreshold {
		log.Printf("Spending manager: balance %d sats below threshold %d sats, recharging...",
			sm.balance, sm.rechargeThreshold)

		txHash, err := blockchain.SendPayment(sm.client, sm.providerAddr, sm.rechargeAmount, sm.localKey.PubKey(), sm.addressType)
		if err != nil {
			log.Printf("Spending manager: failed to send recharge payment: %v", err)
			return
		}

		sm.balance += sm.rechargeAmount
		sm.lastRecharge = time.Now()

		// Also record this as spending (it's a payment to provider)
		sm.spentToday += sm.rechargeAmount

		log.Printf("Spending manager: recharged %d sats, new balance: %d sats (tx: %s)",
			sm.rechargeAmount, sm.balance, txHash.String())
	}
}

// RecordPayment adds a payment amount to the spending total.
// Returns an error if the payment would exceed limits.
func (sm *SpendingManager) RecordPayment(amount uint64) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check total spending limit
	if sm.limitEnabled && sm.spentToday+amount > sm.totalLimit {
		return fmt.Errorf("spending limit exceeded: %d+%d > %d", sm.spentToday, amount, sm.totalLimit)
	}

	// Check session spending limit
	if sm.maxSessionSpend > 0 && (sm.sessionStartSpent+amount) > sm.maxSessionSpend {
		return fmt.Errorf("session spending limit exceeded: %d+%d > %d", sm.sessionStartSpent, amount, sm.maxSessionSpend)
	}

	// Deduct from balance if using prepaid model
	if sm.balance < amount {
		return fmt.Errorf("insufficient balance: have %d, need %d", sm.balance, amount)
	}
	sm.balance -= amount
	sm.spentToday += amount

	// Check warning threshold
	if !sm.warningIssued && sm.limitEnabled {
		percent := uint32(float64(sm.spentToday) / float64(sm.totalLimit) * 100)
		if percent >= sm.warningPercent {
			log.Printf("WARNING: Spending at %d%% of limit (%d/%d sats)", percent, sm.spentToday, sm.totalLimit)
			sm.warningIssued = true
		}
	}

	log.Printf("Spending manager: recorded payment of %d sats, balance now: %d, spent today: %d",
		amount, sm.balance, sm.spentToday)
	return nil
}

// ShouldDisconnect returns true if the session should be terminated due to spending limits.
func (sm *SpendingManager) ShouldDisconnect() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.autoDisconnect && sm.limitEnabled && sm.spentToday >= sm.totalLimit {
		return true
	}
	return false
}

// GetRemainingBudget returns the remaining spending limit for today.
func (sm *SpendingManager) GetRemainingBudget() uint64 {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if !sm.limitEnabled {
		return ^uint64(0) // unlimited
	}
	if sm.spentToday >= sm.totalLimit {
		return 0
	}
	return sm.totalLimit - sm.spentToday
}

// GetBalance returns the current prepaid balance.
func (sm *SpendingManager) GetBalance() uint64 {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.balance
}

// SetSessionStart captures the current spending to enforce per-session limits.
func (sm *SpendingManager) SetSessionStart() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessionStartSpent = sm.spentToday
	sm.warningIssued = false // reset warning for new session
}

// AddCredits adds to the prepaid balance (e.g., from external top-up).
func (sm *SpendingManager) AddCredits(amount uint64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.balance += amount
	log.Printf("Spending manager: added %d sats, balance now: %d", amount, sm.balance)
}

// Stop gracefully shuts down the spending manager.
func (sm *SpendingManager) Stop() {
	close(sm.stopped)
	log.Println("Spending manager: stopped")
}

// IsEnabled returns true if auto-recharge is enabled.
func (sm *SpendingManager) IsEnabled() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.rechargeEnabled
}
