package blockchain

import (
	"context"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/rpcclient"
)

const (
	defaultMinFeePerKb uint64 = 1000
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

type FeeConfig struct {
	MinFeeKb     uint64
	DefaultFeeKb uint64
}

func NewFeeConfig(minFee, defaultFee uint64) FeeConfig {
	cfg := FeeConfig{
		MinFeeKb:     defaultMinFeePerKb,
		DefaultFeeKb: defaultMinFeePerKb,
	}
	if minFee > 0 {
		cfg.MinFeeKb = minFee
	}
	if defaultFee > 0 {
		cfg.DefaultFeeKb = defaultFee
	}
	return cfg
}

func estimateDynamicFeePerKbWithMode(ctx context.Context, client *rpcclient.Client, targetBlocks int64, mode btcjson.EstimateSmartFeeMode, cfg FeeConfig) (uint64, error) {
	targetBlocks = clampFeeTarget(targetBlocks)
	feeRate, err := withRetry(ctx, "EstimateSmartFee", 4, 500*time.Millisecond, func() (*btcjson.EstimateSmartFeeResult, error) {
		return client.EstimateSmartFee(targetBlocks, &mode)
	})
	if err == nil && feeRate != nil && feeRate.FeeRate != nil && *feeRate.FeeRate > 0 {
		feeRatePerKb := uint64(*feeRate.FeeRate * 1e5)
		if feeRatePerKb > 0 {
			return feeRatePerKb, nil
		}
	}

	networkInfo, nErr := withRetry(ctx, "GetNetworkInfo", 4, 500*time.Millisecond, func() (*btcjson.GetNetworkInfoResult, error) {
		return client.GetNetworkInfo()
	})
	if nErr == nil && networkInfo != nil && networkInfo.RelayFee > 0 {
		feeRatePerKb := uint64(networkInfo.RelayFee * 1e5)
		if feeRatePerKb > 0 {
			if feeRatePerKb < cfg.MinFeeKb {
				return cfg.DefaultFeeKb, nil
			}
			return feeRatePerKb, nil
		}
	}

	return cfg.DefaultFeeKb, nil
}

func estimateDynamicFeePerKb(ctx context.Context, client *rpcclient.Client, targetBlocks int64, cfg FeeConfig) (uint64, error) {
	return estimateDynamicFeePerKbWithMode(ctx, client, targetBlocks, btcjson.EstimateModeConservative, cfg)
}

func FeeMode(mode string) btcjson.EstimateSmartFeeMode {
	switch mode {
	case "economical":
		return btcjson.EstimateModeEconomical
	default:
		return btcjson.EstimateModeConservative
	}
}
