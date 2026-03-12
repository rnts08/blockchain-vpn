package blockchain

import (
	"testing"

	"github.com/btcsuite/btcd/btcjson"
)

func TestFeeMode(t *testing.T) {
	tests := []struct {
		input    string
		expected btcjson.EstimateSmartFeeMode
	}{
		{"conservative", btcjson.EstimateModeConservative},
		{"", btcjson.EstimateModeConservative},
		{"economical", btcjson.EstimateModeEconomical},
		{"random", btcjson.EstimateModeConservative},
	}

	for _, tt := range tests {
		result := FeeMode(tt.input)
		if result != tt.expected {
			t.Errorf("FeeMode(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}
