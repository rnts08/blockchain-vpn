package tunnel

import (
	"context"
	"log"
	"sync"
	"time"

	"blockchain-vpn/internal/blockchain"
	"blockchain-vpn/internal/config"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/rpcclient"
)

type CreditManager struct {
	mu sync.Mutex

	balance           uint64
	minBalance        uint64
	rechargeAmount    uint64
	rechargeThreshold uint64
	enabled           bool

	lastRecharge     time.Time
	rechargeInterval time.Duration

	client         *rpcclient.Client
	providerAddr   btcutil.Address
	localKey       *btcec.PrivateKey
	providerPubKey *btcec.PublicKey
	stopped        chan struct{}
}

func NewCreditManager(cfg *config.ClientConfig, client *rpcclient.Client, providerAddr btcutil.Address, localKey *btcec.PrivateKey, providerPubKey *btcec.PublicKey) *CreditManager {
	cm := &CreditManager{
		balance:           0,
		minBalance:        cfg.AutoRechargeMinBalance,
		rechargeAmount:    cfg.AutoRechargeAmount,
		rechargeThreshold: cfg.AutoRechargeThreshold,
		enabled:           cfg.AutoRechargeEnabled,
		client:            client,
		providerAddr:      providerAddr,
		localKey:          localKey,
		providerPubKey:    providerPubKey,
		rechargeInterval:  30 * time.Second,
		stopped:           make(chan struct{}),
	}

	if cm.minBalance == 0 {
		cm.minBalance = 100 // Default minimum balance
	}
	if cm.rechargeAmount == 0 {
		cm.rechargeAmount = 1000 // Default recharge amount
	}
	if cm.rechargeThreshold == 0 {
		cm.rechargeThreshold = 500 // Default threshold
	}

	return cm
}

func (cm *CreditManager) Start(ctx context.Context) {
	if !cm.enabled {
		log.Println("Credit manager: auto-recharge disabled")
		return
	}

	log.Printf("Credit manager: started (threshold: %d sats, recharge: %d sats, min: %d sats)",
		cm.rechargeThreshold, cm.rechargeAmount, cm.minBalance)

	go cm.run(ctx)
}

func (cm *CreditManager) run(ctx context.Context) {
	ticker := time.NewTicker(cm.rechargeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Credit manager: stopped")
			return
		case <-ticker.C:
			cm.checkAndRecharge(ctx)
		}
	}
}

func (cm *CreditManager) checkAndRecharge(ctx context.Context) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.balance >= cm.rechargeThreshold {
		return
	}

	log.Printf("Credit manager: balance %d sats below threshold %d sats, recharging...",
		cm.balance, cm.rechargeThreshold)

	txHash, err := blockchain.SendPayment(cm.client, cm.providerAddr, cm.rechargeAmount, cm.localKey.PubKey())
	if err != nil {
		log.Printf("Credit manager: failed to send payment: %v", err)
		return
	}

	cm.balance += cm.rechargeAmount
	cm.lastRecharge = time.Now()

	log.Printf("Credit manager: recharged %d sats, new balance: %d sats (tx: %s)",
		cm.rechargeAmount, cm.balance, txHash.String())
}

func (cm *CreditManager) AddCredits(amount uint64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.balance += amount
	log.Printf("Credit manager: added %d sats, balance now: %d sats", amount, cm.balance)
}

func (cm *CreditManager) GetBalance() uint64 {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.balance
}

func (cm *CreditManager) UseCredits(amount uint64) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.balance < amount {
		log.Printf("Credit manager: insufficient balance %d sats for %d sats charge", cm.balance, amount)
		return false
	}

	cm.balance -= amount
	log.Printf("Credit manager: used %d sats, balance now: %d sats", amount, cm.balance)
	return true
}

func (cm *CreditManager) Stop() {
	close(cm.stopped)
	log.Println("Credit manager: stopped")
}

func (cm *CreditManager) IsEnabled() bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.enabled
}
