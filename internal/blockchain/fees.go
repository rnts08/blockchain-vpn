package blockchain

import (
	"context"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/rpcclient"
)

func clampFeeTarget(target int64) int64 {
	if target <= 0 {
		return 6
	}
	if target > 1008 {
		return 1008
	}
	return target
}

// estimateDynamicFeePerKbWithMode fetches a feerate from the node using the
// given confirmation target (in blocks) and estimation mode.
func estimateDynamicFeePerKbWithMode(ctx context.Context, client *rpcclient.Client, targetBlocks int64, mode btcjson.EstimateSmartFeeMode) (btcutil.Amount, error) {
	targetBlocks = clampFeeTarget(targetBlocks)
	feeRate, err := withRetry(ctx, "EstimateSmartFee", 4, 500*time.Millisecond, func() (*btcjson.EstimateSmartFeeResult, error) {
		return client.EstimateSmartFee(targetBlocks, &mode)
	})
	if err == nil && feeRate != nil && feeRate.FeeRate != nil && *feeRate.FeeRate > 0 {
		amount, convErr := btcutil.NewAmount(*feeRate.FeeRate)
		if convErr == nil && amount > 0 {
			return amount, nil
		}
	}

	networkInfo, nErr := withRetry(ctx, "GetNetworkInfo", 4, 500*time.Millisecond, func() (*btcjson.GetNetworkInfoResult, error) {
		return client.GetNetworkInfo()
	})
	if nErr == nil && networkInfo != nil && networkInfo.RelayFee > 0 {
		amount, convErr := btcutil.NewAmount(networkInfo.RelayFee)
		if convErr == nil && amount > 0 {
			return amount, nil
		}
	}

	return 0, fmt.Errorf("could not determine dynamic fee rate from node")
}

// estimateDynamicFeePerKb is the default-mode convenience wrapper (conservative, 6 blocks).
func estimateDynamicFeePerKb(ctx context.Context, client *rpcclient.Client, targetBlocks int64) (btcutil.Amount, error) {
	return estimateDynamicFeePerKbWithMode(ctx, client, targetBlocks, btcjson.EstimateModeConservative)
}

// FeeMode converts a config string to a btcjson EstimateSmartFeeMode.
func FeeMode(mode string) btcjson.EstimateSmartFeeMode {
	switch mode {
	case "economical":
		return btcjson.EstimateModeEconomical
	default:
		return btcjson.EstimateModeConservative
	}
}
