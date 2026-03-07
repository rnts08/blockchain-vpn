package blockchain

import (
	"testing"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
)

func TestDeterministicSelectCoins(t *testing.T) {
	tests := []struct {
		name      string
		targetSat int64
		unspent   []btcjson.ListUnspentResult
		wantCount int
		wantTotal int64
	}{
		{
			name:      "exact single match",
			targetSat: 2000,
			unspent: []btcjson.ListUnspentResult{
				{TxID: "c", Vout: 0, Amount: 0.00003},
				{TxID: "a", Vout: 0, Amount: 0.00002},
				{TxID: "b", Vout: 0, Amount: 0.00001},
			},
			wantCount: 1,
			wantTotal: 2000,
		},
		{
			name:      "single greater preferred over accumulation",
			targetSat: 2500,
			unspent: []btcjson.ListUnspentResult{
				{TxID: "a", Vout: 0, Amount: 0.00001},
				{TxID: "b", Vout: 0, Amount: 0.00002},
				{TxID: "c", Vout: 0, Amount: 0.00004},
			},
			wantCount: 1,
			wantTotal: 4000,
		},
		{
			name:      "accumulate when no single covers target",
			targetSat: 5000,
			unspent: []btcjson.ListUnspentResult{
				{TxID: "a", Vout: 0, Amount: 0.00001},
				{TxID: "b", Vout: 0, Amount: 0.00002},
				{TxID: "c", Vout: 0, Amount: 0.00003},
			},
			wantCount: 2,
			wantTotal: 5000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := btcutil.Amount(tt.targetSat)
			selected, total, err := deterministicSelectCoins(tt.unspent, target)
			if err != nil {
				t.Fatalf("deterministicSelectCoins returned error: %v", err)
			}
			if len(selected) != tt.wantCount {
				t.Fatalf("selected count mismatch: got %d want %d", len(selected), tt.wantCount)
			}
			if int64(total) != tt.wantTotal {
				t.Fatalf("selected total mismatch: got %d want %d", total, tt.wantTotal)
			}
		})
	}
}
