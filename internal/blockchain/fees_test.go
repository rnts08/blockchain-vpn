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

func TestClampFeeTarget(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected int64
	}{
		{"zero defaults to 6", 0, 6},
		{"negative defaults to 6", -1, 6},
		{"large negative defaults to 6", -1000, 6},
		{"valid 1 stays 1", 1, 1},
		{"valid 6 stays 6", 6, 6},
		{"valid 100 stays 100", 100, 100},
		{"valid 1008 stays 1008", 1008, 1008},
		{"over 1008 clamped to 1008", 2000, 1008},
		{"very large clamped to 1008", 1000000, 1008},
	}

	for _, tt := range tests {
		result := clampFeeTarget(tt.input)
		if result != tt.expected {
			t.Errorf("clampFeeTarget(%d) = %d, want %d (%s)", tt.input, result, tt.expected, tt.name)
		}
	}
}
