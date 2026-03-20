package blockchain

import (
	"context"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
)

func TestDeterministicSelectCoins_InsufficientFunds(t *testing.T) {
	t.Parallel()
	unspent := []btcjson.ListUnspentResult{
		{TxID: "a", Vout: 0, Amount: 0.00001},
	}
	_, _, err := deterministicSelectCoins(unspent, btcutil.Amount(10000))
	if err == nil {
		t.Error("expected error for insufficient funds")
	}
}

func TestDeterministicSelectCoins_EmptyUnspent(t *testing.T) {
	t.Parallel()
	_, _, err := deterministicSelectCoins([]btcjson.ListUnspentResult{}, btcutil.Amount(1000))
	if err == nil {
		t.Error("expected error for empty unspent")
	}
}

func TestDeterministicSelectCoins_ExactMatch(t *testing.T) {
	t.Parallel()
	unspent := []btcjson.ListUnspentResult{
		{TxID: "a", Vout: 0, Amount: 0.00002000},
	}
	selected, total, err := deterministicSelectCoins(unspent, btcutil.Amount(2000))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selected) != 1 {
		t.Errorf("expected 1 coin, got %d", len(selected))
	}
	if int64(total) != 2000 {
		t.Errorf("expected total 2000, got %d", int64(total))
	}
}

func TestDeterministicSelectCoins_SingleOverTarget(t *testing.T) {
	t.Parallel()
	unspent := []btcjson.ListUnspentResult{
		{TxID: "a", Vout: 0, Amount: 0.00000500},
		{TxID: "b", Vout: 0, Amount: 0.00003000},
		{TxID: "c", Vout: 0, Amount: 0.00001000},
	}
	selected, total, err := deterministicSelectCoins(unspent, btcutil.Amount(2500))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selected) != 1 {
		t.Errorf("expected 1 coin, got %d", len(selected))
	}
	if selected[0].TxID != "b" {
		t.Errorf("expected smallest-over-target coin b, got %s", selected[0].TxID)
	}
	if int64(total) != 3000 {
		t.Errorf("expected total 3000, got %d", int64(total))
	}
}

func TestDeterministicSelectCoins_AccumulateToTarget(t *testing.T) {
	t.Parallel()
	unspent := []btcjson.ListUnspentResult{
		{TxID: "a", Vout: 0, Amount: 0.00000500},
		{TxID: "b", Vout: 0, Amount: 0.00000500},
		{TxID: "c", Vout: 0, Amount: 0.00000500},
	}
	selected, total, err := deterministicSelectCoins(unspent, btcutil.Amount(1200))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selected) != 3 {
		t.Errorf("expected 3 coins accumulated, got %d", len(selected))
	}
	if int64(total) != 1500 {
		t.Errorf("expected total 1500 (smallest-first accumulation), got %d", int64(total))
	}
}

func TestDeterministicSelectCoins_LargestFirst(t *testing.T) {
	t.Parallel()
	unspent := []btcjson.ListUnspentResult{
		{TxID: "small", Vout: 0, Amount: 0.00000100},
		{TxID: "medium", Vout: 0, Amount: 0.00000500},
		{TxID: "large", Vout: 0, Amount: 0.00001000},
	}
	selected, total, err := deterministicSelectCoins(unspent, btcutil.Amount(1200))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if int64(total) != 1500 {
		t.Errorf("expected accumulation with largest first (large+medium = 1500 > 1200), got %d", int64(total))
	}
	if len(selected) != 2 {
		t.Errorf("expected 2 coins selected, got %d", len(selected))
	}
	if selected[0].TxID != "large" {
		t.Errorf("expected largest first, got %s", selected[0].TxID)
	}
	if selected[1].TxID != "medium" {
		t.Errorf("expected medium second, got %s", selected[1].TxID)
	}
}

func TestDeterministicSelectCoins_IdenticalAmounts(t *testing.T) {
	t.Parallel()
	unspent := []btcjson.ListUnspentResult{
		{TxID: "a", Vout: 0, Amount: 0.00001000},
		{TxID: "b", Vout: 0, Amount: 0.00001000},
		{TxID: "c", Vout: 0, Amount: 0.00001000},
	}
	_, _, err := deterministicSelectCoins(unspent, btcutil.Amount(1000))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyPaymentInput_EdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		price  uint64
		amount uint64
		wantOK bool
	}{
		{1000, 999, false},
		{1000, 1000, true},
		{1000, 1001, true},
		{0, 0, true},
		{0, 1, true},
		{100, 0, false},
	}
	for _, tt := range tests {
		err := VerifyPaymentInput(tt.price, tt.amount)
		ok := err == nil
		if ok != tt.wantOK {
			t.Errorf("VerifyPaymentInput(price=%d, amount=%d) ok=%v, wantOK=%v", tt.price, tt.amount, ok, tt.wantOK)
		}
	}
}

func TestGetPaymentVerification(t *testing.T) {
	t.Parallel()
	tests := []struct {
		price  uint64
		amount uint64
		want   uint64
		wantOK bool
	}{
		{1000, 1000, 1000, true},
		{1000, 1500, 1500, true},
		{1000, 999, 0, false},
	}
	for _, tt := range tests {
		got, err := GetPaymentVerification(tt.price, tt.amount)
		ok := err == nil
		if ok != tt.wantOK {
			t.Errorf("GetPaymentVerification(price=%d, amount=%d) ok=%v, wantOK=%v", tt.price, tt.amount, ok, tt.wantOK)
		}
		if ok && got != tt.want {
			t.Errorf("GetPaymentVerification(price=%d, amount=%d) = %d, want %d", tt.price, tt.amount, got, tt.want)
		}
	}
}

func TestWaitForConfirmations_NilClient(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, err := WaitForConfirmations(ctx, nil, nil, 1, time.Second)
	if err == nil {
		t.Error("expected error for nil client")
	}
	if err.Error() != "client is nil" {
		t.Errorf("expected 'client is nil', got %q", err.Error())
	}
}

func TestWaitForConfirmations_ZeroConfirmations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, err := WaitForConfirmations(ctx, nil, nil, 0, time.Second)
	if err == nil {
		t.Error("expected error for nil client")
	}
	if err.Error() != "client is nil" {
		t.Errorf("expected 'client is nil', got %q", err.Error())
	}
}
