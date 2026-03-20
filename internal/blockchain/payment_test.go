package blockchain

import (
	"context"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcjson"
)

func TestDeterministicSelectCoins_InsufficientFunds(t *testing.T) {
	t.Parallel()
	unspent := []btcjson.ListUnspentResult{
		{TxID: "a", Vout: 0, Amount: 0.00001},
	}
	_, _, err := deterministicSelectCoins(unspent, 10000)
	if err == nil {
		t.Error("expected error for insufficient funds")
	}
}

func TestDeterministicSelectCoins_EmptyUnspent(t *testing.T) {
	t.Parallel()
	_, _, err := deterministicSelectCoins([]btcjson.ListUnspentResult{}, 1000)
	if err == nil {
		t.Error("expected error for empty unspent")
	}
}

func TestDeterministicSelectCoins_ExactMatch(t *testing.T) {
	t.Parallel()
	unspent := []btcjson.ListUnspentResult{
		{TxID: "a", Vout: 0, Amount: 0.00003},
	}
	selected, total, err := deterministicSelectCoins(unspent, 3000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selected) != 1 {
		t.Errorf("expected 1 coin, got %d", len(selected))
	}
	if total != 3000 {
		t.Errorf("expected total 3000, got %d", total)
	}
}

func TestDeterministicSelectCoins_SingleOverTarget(t *testing.T) {
	t.Parallel()
	unspent := []btcjson.ListUnspentResult{
		{TxID: "a", Vout: 0, Amount: 0.00001},
		{TxID: "b", Vout: 0, Amount: 0.00003},
		{TxID: "c", Vout: 0, Amount: 0.00002},
	}
	selected, total, err := deterministicSelectCoins(unspent, 2500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selected) != 1 {
		t.Errorf("expected 1 coin, got %d", len(selected))
	}
	if selected[0].TxID != "b" {
		t.Errorf("expected smallest-over-target coin b, got %s", selected[0].TxID)
	}
	if total != 3000 {
		t.Errorf("expected total 3000, got %d", total)
	}
}

func TestDeterministicSelectCoins_AccumulateToTarget(t *testing.T) {
	t.Parallel()
	unspent := []btcjson.ListUnspentResult{
		{TxID: "a", Vout: 0, Amount: 0.00001},
		{TxID: "b", Vout: 0, Amount: 0.00001},
		{TxID: "c", Vout: 0, Amount: 0.00001},
	}
	selected, total, err := deterministicSelectCoins(unspent, 2500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selected) != 3 {
		t.Errorf("expected 3 coins accumulated, got %d", len(selected))
	}
	if total != 3000 {
		t.Errorf("expected total 3000 (smallest-first accumulation), got %d", total)
	}
}

func TestDeterministicSelectCoins_LargestFirst(t *testing.T) {
	t.Parallel()
	unspent := []btcjson.ListUnspentResult{
		{TxID: "small", Vout: 0, Amount: 0.00001},
		{TxID: "medium", Vout: 0, Amount: 0.00005},
		{TxID: "large", Vout: 0, Amount: 0.00010},
	}
	selected, total, err := deterministicSelectCoins(unspent, 12000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 15000 {
		t.Errorf("expected accumulation with largest first (large+medium = 15000 > 12000), got %d", total)
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
	_, _, err := deterministicSelectCoins(unspent, 1000)
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
	_, err := WaitForConfirmations(ctx, nil, "", 1, time.Second)
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
	_, err := WaitForConfirmations(ctx, nil, "", 0, time.Second)
	if err == nil {
		t.Error("expected error for nil client")
	}
	if err.Error() != "client is nil" {
		t.Errorf("expected 'client is nil', got %q", err.Error())
	}
}
