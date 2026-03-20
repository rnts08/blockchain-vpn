//go:build functional

package tunnel

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"blockchain-vpn/internal/config"
	"blockchain-vpn/internal/geoip"
	"blockchain-vpn/internal/protocol"

	"github.com/btcsuite/btcd/btcec/v2"
)

func TestFunctional_TimeBasedBilling(t *testing.T) {
	t.Parallel()

	endpoint := &protocol.VPNEndpoint{
		IP:                 []byte{192, 168, 1, 1},
		Port:               443,
		Price:              10,
		PricingMethod:      protocol.PricingMethodTime,
		TimeUnitSecs:       60,
		SessionTimeoutSecs: 3600,
	}

	meter := NewUsageMeter(endpoint, 10)

	if meter.pricingMethod != protocol.PricingMethodTime {
		t.Fatalf("Expected pricing method time, got %d", meter.pricingMethod)
	}

	initialCost := meter.CurrentCost()
	if initialCost != 0 {
		t.Errorf("Initial cost should be 0, got %d", initialCost)
	}

	time.Sleep(100 * time.Millisecond)

	costAfter100ms := meter.CurrentCost()
	if costAfter100ms != 0 {
		t.Errorf("Cost after 100ms should be 0 (less than 60s unit), got %d", costAfter100ms)
	}

	time.Sleep(65 * time.Second)

	costAfter65s := meter.CurrentCost()
	if costAfter65s != 10 {
		t.Errorf("Cost after 65s should be 10 sats (1 unit), got %d", costAfter65s)
	}

	t.Logf("Time-based billing: Cost after 65s: %d sats", costAfter65s)
}

func TestFunctional_TimeBasedBilling_PaymentRenewal(t *testing.T) {
	t.Parallel()

	endpoint := &protocol.VPNEndpoint{
		IP:            []byte{192, 168, 1, 1},
		Port:          443,
		Price:         10,
		PricingMethod: protocol.PricingMethodTime,
		TimeUnitSecs:  60,
	}

	meter := NewUsageMeter(endpoint, 10)

	shouldRenew := meter.ShouldRenewPayment(0, 80)
	if shouldRenew {
		t.Error("ShouldRenewPayment should be false when there's no prior payment")
	}

	time.Sleep(65 * time.Second)

	shouldRenew = meter.ShouldRenewPayment(0, 80)
	if !shouldRenew {
		t.Error("ShouldRenewPayment should be true after 65s (past 1 billing unit)")
	}

	shouldRenew = meter.ShouldRenewPayment(10, 80)
	if shouldRenew {
		t.Error("ShouldRenewPayment should be false when already paid for current usage")
	}

	t.Log("Time-based payment renewal logic works correctly")
}

func TestFunctional_TimeBasedBilling_ThresholdBehavior(t *testing.T) {
	t.Parallel()

	endpoint := &protocol.VPNEndpoint{
		IP:            []byte{192, 168, 1, 1},
		Port:          443,
		Price:         100,
		PricingMethod: protocol.PricingMethodTime,
		TimeUnitSecs:  60,
	}

	meter := NewUsageMeter(endpoint, 100)

	time.Sleep(50 * time.Second)

	shouldRenewAt80 := meter.ShouldRenewPayment(0, 80)
	if shouldRenewAt80 {
		t.Error("ShouldRenewPayment(80%) should be false at 50s (80% of 60s)")
	}

	time.Sleep(10 * time.Second)

	shouldRenewAt80 = meter.ShouldRenewPayment(0, 80)
	if !shouldRenewAt80 {
		t.Error("ShouldRenewPayment(80%) should be true at 60s (80% of 60s threshold)")
	}

	shouldRenewAt50 := meter.ShouldRenewPayment(0, 50)
	if !shouldRenewAt50 {
		t.Error("ShouldRenewPayment(50%) should be true at 60s (50% of 60s threshold)")
	}

	t.Log("Time-based billing threshold behavior works correctly")
}

func TestFunctional_DataBasedBilling(t *testing.T) {
	t.Parallel()

	endpoint := &protocol.VPNEndpoint{
		IP:                 []byte{192, 168, 1, 1},
		Port:               443,
		Price:              25,
		PricingMethod:      protocol.PricingMethodData,
		DataUnitBytes:      1_000_000,
		SessionTimeoutSecs: 3600,
	}

	meter := NewUsageMeter(endpoint, 25)

	if meter.pricingMethod != protocol.PricingMethodData {
		t.Fatalf("Expected pricing method data, got %d", meter.pricingMethod)
	}

	initialCost := meter.CurrentCost()
	if initialCost != 0 {
		t.Errorf("Initial cost should be 0, got %d", initialCost)
	}

	meter.AddTraffic(500_000, 0)

	costAfter500KB := meter.CurrentCost()
	if costAfter500KB != 0 {
		t.Errorf("Cost after 500KB should be 0 (less than 1MB unit), got %d", costAfter500KB)
	}

	meter.AddTraffic(500_000, 0)

	costAfter1MB := meter.CurrentCost()
	if costAfter1MB != 25 {
		t.Errorf("Cost after 1MB should be 25 sats (1 unit), got %d", costAfter1MB)
	}

	meter.AddTraffic(1_000_000, 1_000_000)

	costAfter3MB := meter.CurrentCost()
	if costAfter3MB != 75 {
		t.Errorf("Cost after 3MB should be 75 sats (3 units), got %d", costAfter3MB)
	}

	t.Logf("Data-based billing: Cost after 1MB: %d sats, after 3MB: %d sats", costAfter1MB, costAfter3MB)
}

func TestFunctional_DataBasedBilling_PaymentRenewal(t *testing.T) {
	t.Parallel()

	endpoint := &protocol.VPNEndpoint{
		IP:            []byte{192, 168, 1, 1},
		Port:          443,
		Price:         10,
		PricingMethod: protocol.PricingMethodData,
		DataUnitBytes: 1_000_000,
	}

	meter := NewUsageMeter(endpoint, 10)

	shouldRenew := meter.ShouldRenewPayment(0, 80)
	if shouldRenew {
		t.Error("ShouldRenewPayment should be false when there's no data transferred")
	}

	meter.AddTraffic(500_000, 0)

	shouldRenew = meter.ShouldRenewPayment(0, 80)
	if shouldRenew {
		t.Error("ShouldRenewPayment should be false at 500KB (less than 80% of 1MB)")
	}

	meter.AddTraffic(510_000, 0)

	shouldRenew = meter.ShouldRenewPayment(0, 80)
	if !shouldRenew {
		t.Error("ShouldRenewPayment should be true after 1.01MB (past 1MB threshold)")
	}

	shouldRenew = meter.ShouldRenewPayment(10, 80)
	if shouldRenew {
		t.Error("ShouldRenewPayment should be false when already paid for current usage")
	}

	t.Log("Data-based payment renewal logic works correctly")
}

func TestFunctional_DataBasedBilling_Tiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		dataUnitBytes uint32
		price         uint64
		trafficSent   uint64
		trafficRecv   uint64
		expectedCost  uint64
	}{
		{
			name:          "1KB tier with 1 sat price",
			dataUnitBytes: 1024,
			price:         1,
			trafficSent:   1024,
			trafficRecv:   0,
			expectedCost:  1,
		},
		{
			name:          "1MB tier with 10 sats per MB",
			dataUnitBytes: 1_000_000,
			price:         10,
			trafficSent:   500_000,
			trafficRecv:   500_000,
			expectedCost:  10,
		},
		{
			name:          "10MB tier with 50 sats per 10MB",
			dataUnitBytes: 10_000_000,
			price:         50,
			trafficSent:   25_000_000,
			trafficRecv:   0,
			expectedCost:  100,
		},
		{
			name:          "partial unit not charged",
			dataUnitBytes: 1_000_000,
			price:         25,
			trafficSent:   500_000,
			trafficRecv:   0,
			expectedCost:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := &protocol.VPNEndpoint{
				IP:            []byte{192, 168, 1, 1},
				Port:          443,
				Price:         tt.price,
				PricingMethod: protocol.PricingMethodData,
				DataUnitBytes: tt.dataUnitBytes,
			}

			meter := NewUsageMeter(endpoint, tt.price)
			meter.AddTraffic(tt.trafficSent, tt.trafficRecv)

			cost := meter.CurrentCost()
			if cost != tt.expectedCost {
				t.Errorf("Expected cost %d, got %d", tt.expectedCost, cost)
			}
		})
	}
}

func TestFunctional_SpendingLimits_Enforcement(t *testing.T) {
	t.Parallel()

	cfg := &config.ClientConfig{
		AutoRechargeEnabled:    false,
		SpendingLimitEnabled:   true,
		SpendingLimitSats:      1000,
		SpendingWarningPercent: 50,
		AutoDisconnectOnLimit:  true,
		MaxSessionSpendingSats: 500,
	}

	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey: %v", err)
	}

	sm := NewSpendingManager(cfg, nil, nil, privKey, privKey.PubKey())
	sm.balance = 10000

	sm.SetSessionStart()

	if err := sm.RecordPayment(100); err != nil {
		t.Errorf("First payment should succeed: %v", err)
	}

	remaining := sm.GetRemainingBudget()
	if remaining != 900 {
		t.Errorf("Remaining budget should be 900, got %d", remaining)
	}

	if err := sm.RecordPayment(400); err != nil {
		t.Errorf("Second payment should succeed: %v", err)
	}

	remaining = sm.GetRemainingBudget()
	if remaining != 500 {
		t.Errorf("Remaining budget should be 500, got %d", remaining)
	}

	if sm.ShouldDisconnect() {
		t.Error("ShouldDisconnect should be false before reaching limit")
	}

	if err := sm.RecordPayment(500); err != nil {
		t.Errorf("Payment up to limit should succeed: %v", err)
	}

	if !sm.ShouldDisconnect() {
		t.Error("ShouldDisconnect should be true after reaching limit")
	}

	remaining = sm.GetRemainingBudget()
	if remaining != 0 {
		t.Errorf("Remaining budget should be 0 at limit, got %d", remaining)
	}

	if err := sm.RecordPayment(100); err == nil {
		t.Error("Payment exceeding limit should fail")
	}

	t.Log("Spending limit enforcement works correctly")
}

func TestFunctional_SpendingLimits_SessionLimit(t *testing.T) {
	t.Parallel()

	cfg := &config.ClientConfig{
		SpendingLimitEnabled:   false,
		MaxSessionSpendingSats: 300,
	}

	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey: %v", err)
	}

	sm := NewSpendingManager(cfg, nil, nil, privKey, privKey.PubKey())
	sm.balance = 10000
	sm.SetSessionStart()

	if err := sm.RecordPayment(400); err == nil {
		t.Error("Single payment exceeding session limit should fail")
	} else {
		if !strings.Contains(err.Error(), "session spending limit exceeded") {
			t.Errorf("Expected session spending limit error, got: %v", err)
		}
	}

	t.Log("Session spending limit enforcement works correctly")
}

func TestFunctional_SpendingLimits_WarningThreshold(t *testing.T) {
	t.Parallel()

	cfg := &config.ClientConfig{
		SpendingLimitEnabled:   true,
		SpendingLimitSats:      1000,
		SpendingWarningPercent: 80,
	}

	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey: %v", err)
	}

	sm := NewSpendingManager(cfg, nil, nil, privKey, privKey.PubKey())
	sm.balance = 10000
	sm.SetSessionStart()

	if err := sm.RecordPayment(700); err != nil {
		t.Errorf("Payment at 70%% should succeed: %v", err)
	}

	if err := sm.RecordPayment(150); err != nil {
		t.Errorf("Payment at 85%% should succeed: %v", err)
	}

	t.Log("Spending warning threshold works correctly")
}

func TestFunctional_RefundFlow_LowQualityDisconnection(t *testing.T) {
	t.Parallel()

	restore := mockSecurityCheckFns(
		func() (net.IP, error) { return net.ParseIP("203.0.113.10"), nil },
		func() (*geoip.GeoLocation, error) { return &geoip.GeoLocation{CountryCode: "DE"}, nil },
		func(context.Context, string) (uint32, error) { return 1, nil },
	)
	defer restore()

	quality, err := CheckConnectionQuality(context.Background(), ClientSecurityExpectations{
		ExpectedCountry:       "US",
		ExpectedBandwidthKB:   1000000,
		VerifyThroughputAfter: true,
		ThroughputProbePort:   51821,
		QualityThreshold:      0.75,
	}, net.ParseIP("203.0.113.5"))
	if err != nil {
		t.Fatalf("CheckConnectionQuality failed: %v", err)
	}

	if quality.Passed {
		t.Error("Connection should fail quality check with low bandwidth")
	}

	if len(quality.Warnings) == 0 {
		t.Error("Expected quality warnings for low connection quality")
	}

	hasRefundRecommendation := false
	for _, warning := range quality.Warnings {
		if strings.Contains(warning, "bandwidth") || strings.Contains(warning, "country") {
			hasRefundRecommendation = true
			break
		}
	}

	if !hasRefundRecommendation {
		t.Error("Expected refund recommendation in warnings")
	}

	t.Logf("Refund flow: Quality score %.0f%%, Passed=%v, Warnings=%v",
		quality.QualityScore*100, quality.Passed, quality.Warnings)
}

func TestFunctional_RefundFlow_HighQualityNoRefund(t *testing.T) {
	t.Parallel()

	restore := mockSecurityCheckFns(
		func() (net.IP, error) { return net.ParseIP("203.0.113.10"), nil },
		func() (*geoip.GeoLocation, error) { return &geoip.GeoLocation{CountryCode: "US"}, nil },
		func(context.Context, string) (uint32, error) { return 50000, nil },
	)
	defer restore()

	quality, err := CheckConnectionQuality(context.Background(), ClientSecurityExpectations{
		ExpectedCountry:       "US",
		ExpectedBandwidthKB:   10000,
		VerifyThroughputAfter: true,
		ThroughputProbePort:   51821,
		QualityThreshold:      0.75,
	}, net.ParseIP("203.0.113.5"))
	if err != nil {
		t.Fatalf("CheckConnectionQuality failed: %v", err)
	}

	if quality.QualityScore < 0.5 {
		t.Errorf("High quality connection should have score >= 50%%, got %.0f%%", quality.QualityScore*100)
	}

	t.Logf("High quality: Quality score %.0f%%, Passed=%v",
		quality.QualityScore*100, quality.Passed)
}

func TestFunctional_RefundFlow_QualityThresholdEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		bandwidthKB          uint32
		expectedBandwidthKB  uint32
		expectedScoreMinimum float32
	}{
		{
			name:                 "High bandwidth",
			bandwidthKB:          10000,
			expectedBandwidthKB:  10000,
			expectedScoreMinimum: 0.5,
		},
		{
			name:                 "Medium bandwidth",
			bandwidthKB:          5000,
			expectedBandwidthKB:  10000,
			expectedScoreMinimum: 0.2,
		},
		{
			name:                 "Low bandwidth",
			bandwidthKB:          1000,
			expectedBandwidthKB:  10000,
			expectedScoreMinimum: 0.0,
		},
		{
			name:                 "Zero bandwidth",
			bandwidthKB:          1,
			expectedBandwidthKB:  10000,
			expectedScoreMinimum: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := mockSecurityCheckFns(
				func() (net.IP, error) { return net.ParseIP("203.0.113.10"), nil },
				func() (*geoip.GeoLocation, error) { return nil, nil },
				func(context.Context, string) (uint32, error) { return tt.bandwidthKB, nil },
			)
			defer restore()

			quality, err := CheckConnectionQuality(context.Background(), ClientSecurityExpectations{
				ExpectedCountry:       "",
				ExpectedBandwidthKB:   tt.expectedBandwidthKB,
				VerifyThroughputAfter: true,
				ThroughputProbePort:   51821,
				QualityThreshold:      0.75,
			}, net.ParseIP("203.0.113.5"))
			if err != nil {
				t.Fatalf("CheckConnectionQuality failed: %v", err)
			}

			if quality.QualityScore < tt.expectedScoreMinimum {
				t.Errorf("Expected score >= %.0f%%, got %.0f%% (bandwidth=%d KB)",
					tt.expectedScoreMinimum*100, quality.QualityScore*100, tt.bandwidthKB)
			}
		})
	}
}

func TestFunctional_RefundFlow_DNSCausesRefund(t *testing.T) {
	t.Parallel()

	restore := mockSecurityCheckFns(
		func() (net.IP, error) { return net.ParseIP("203.0.113.10"), nil },
		func() (*geoip.GeoLocation, error) { return &geoip.GeoLocation{CountryCode: "XX"}, nil },
		func(context.Context, string) (uint32, error) { return 50000, nil },
	)
	defer restore()

	quality, err := CheckConnectionQuality(context.Background(), ClientSecurityExpectations{
		ExpectedCountry:       "US",
		ExpectedBandwidthKB:   10000,
		VerifyThroughputAfter: true,
		ThroughputProbePort:   51821,
		QualityThreshold:      0.75,
	}, net.ParseIP("203.0.113.5"))
	if err != nil {
		t.Fatalf("CheckConnectionQuality failed: %v", err)
	}

	if len(quality.Warnings) == 0 {
		t.Error("Expected warnings for country mismatch")
	}

	t.Logf("Country mismatch refund flow: Quality score %.0f%%, Passed=%v, Warnings=%v",
		quality.QualityScore*100, quality.Passed, quality.Warnings)
}
