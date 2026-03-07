package blockchain

import (
	"context"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/rpcclient"
)

func estimateDynamicFeePerKb(ctx context.Context, client *rpcclient.Client, targetBlocks int64) (btcutil.Amount, error) {
	mode := btcjson.EstimateModeConservative
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
