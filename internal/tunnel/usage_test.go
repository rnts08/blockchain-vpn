package tunnel

import (
	"testing"
	"time"

	"blockchain-vpn/internal/protocol"
)

func TestUsageMeter_NewUsageMeter(t *testing.T) {
	tests := []struct {
		name         string
		endpoint     *protocol.VPNEndpoint
		price        uint64
		expectMethod uint8
		expectPrice  uint64
		expectTime   uint32
		expectData   uint32
		expectMax    int
	}{
		{
			name: "session pricing default",
			endpoint: &protocol.VPNEndpoint{
				PricingMethod:      0,
				Price:              100,
				TimeUnitSecs:       0,
				DataUnitBytes:      0,
				SessionTimeoutSecs: 0,
			},
			price:        100,
			expectMethod: protocol.PricingMethodSession,
			expectPrice:  100,
		},
		{
			name: "time-based pricing",
			endpoint: &protocol.VPNEndpoint{
				PricingMethod:      protocol.PricingMethodTime,
				Price:              50,
				TimeUnitSecs:       60,
				DataUnitBytes:      0,
				SessionTimeoutSecs: 3600,
			},
			price:        50,
			expectMethod: protocol.PricingMethodTime,
			expectPrice:  50,
			expectTime:   60,
			expectMax:    3600,
		},
		{
			name: "data-based pricing",
			endpoint: &protocol.VPNEndpoint{
				PricingMethod:      protocol.PricingMethodData,
				Price:              25,
				TimeUnitSecs:       0,
				DataUnitBytes:      1000000, // 1MB
				SessionTimeoutSecs: 0,
			},
			price:        25,
			expectMethod: protocol.PricingMethodData,
			expectPrice:  25,
			expectData:   1000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meter := NewUsageMeter(tt.endpoint, tt.price)
			if meter == nil {
				t.Fatalf("NewUsageMeter returned nil")
			}
			if got := meter.pricingMethod; got != tt.expectMethod {
				t.Errorf("pricingMethod = %v, want %v", got, tt.expectMethod)
			}
			if got := meter.pricePerUnit; got != tt.expectPrice {
				t.Errorf("pricePerUnit = %v, want %v", got, tt.expectPrice)
			}
			if got := meter.timeUnitSecs; got != tt.expectTime {
				t.Errorf("timeUnitSecs = %v, want %v", got, tt.expectTime)
			}
			if got := meter.dataUnitBytes; got != tt.expectData {
				t.Errorf("dataUnitBytes = %v, want %v", got, tt.expectData)
			}
			if got := meter.maxSessionSecs; got != tt.expectMax {
				t.Errorf("maxSessionSecs = %v, want %v", got, tt.expectMax)
			}

			// startTime should be non-zero and recent (within last second)
			if meter.startTime.IsZero() {
				t.Error("startTime is zero")
			}
			if elapsed := time.Since(meter.startTime); elapsed > time.Second {
				t.Errorf("startTime is too old: %v", elapsed)
			}
		})
	}
}

func TestUsageMeter_AddTraffic(t *testing.T) {
	endpoint := &protocol.VPNEndpoint{
		PricingMethod: protocol.PricingMethodData,
		Price:         10,
		DataUnitBytes: 1000,
	}
	meter := NewUsageMeter(endpoint, 10)

	if total := meter.TotalBytes(); total != 0 {
		t.Fatalf("initial TotalBytes = %d, want 0", total)
	}

	meter.AddTraffic(500, 300)
	if total := meter.TotalBytes(); total != 800 {
		t.Errorf("TotalBytes after first AddTraffic = %d, want 800", total)
	}

	meter.AddTraffic(200, 100)
	if total := meter.TotalBytes(); total != 1100 {
		t.Errorf("TotalBytes after second AddTraffic = %d, want 1100", total)
	}
}

func TestUsageMeter_ElapsedSeconds(t *testing.T) {
	endpoint := &protocol.VPNEndpoint{
		PricingMethod: protocol.PricingMethodTime,
		TimeUnitSecs:  60,
	}
	meter := NewUsageMeter(endpoint, 50)

	elapsed := meter.ElapsedSeconds()
	if elapsed < 0 {
		t.Errorf("ElapsedSeconds = %d, want >= 0", elapsed)
	}

	// Wait a bit
	time.Sleep(50 * time.Millisecond)
	elapsed2 := meter.ElapsedSeconds()
	if elapsed2 < elapsed {
		t.Errorf("ElapsedSeconds decreased from %d to %d", elapsed, elapsed2)
	}
}

func TestUsageMeter_TotalBytes(t *testing.T) {
	endpoint := &protocol.VPNEndpoint{
		PricingMethod: protocol.PricingMethodData,
		DataUnitBytes: 1000,
	}
	meter := NewUsageMeter(endpoint, 25)

	if total := meter.TotalBytes(); total != 0 {
		t.Fatalf("initial TotalBytes = %d, want 0", total)
	}

	meter.AddTraffic(100, 200)
	if total := meter.TotalBytes(); total != 300 {
		t.Errorf("TotalBytes = %d, want 300", total)
	}

	meter.AddTraffic(500, 0)
	if total := meter.TotalBytes(); total != 800 {
		t.Errorf("TotalBytes = %d, want 800", total)
	}
}

func TestUsageMeter_CurrentCost(t *testing.T) {
	tests := []struct {
		name       string
		endpoint   *protocol.VPNEndpoint
		price      uint64
		setup      func(*UsageMeter)
		expectCost uint64
	}{
		{
			name: "session pricing returns fixed price",
			endpoint: &protocol.VPNEndpoint{
				PricingMethod: protocol.PricingMethodSession,
			},
			price:      100,
			expectCost: 100,
		},
		{
			name: "time-based: zero unit returns 0",
			endpoint: &protocol.VPNEndpoint{
				PricingMethod: protocol.PricingMethodTime,
				TimeUnitSecs:  60,
			},
			price:      50,
			expectCost: 0, // no time elapsed yet
		},
		{
			name: "data-based: partial unit",
			endpoint: &protocol.VPNEndpoint{
				PricingMethod: protocol.PricingMethodData,
				DataUnitBytes: 1000,
			},
			price: 25,
			setup: func(m *UsageMeter) {
				m.AddTraffic(500, 500) // 1000 bytes = 1 unit
			},
			expectCost: 25,
		},
		{
			name: "data-based: multiple units",
			endpoint: &protocol.VPNEndpoint{
				PricingMethod: protocol.PricingMethodData,
				DataUnitBytes: 1000,
			},
			price: 25,
			setup: func(m *UsageMeter) {
				m.AddTraffic(2500, 0) // 2500 bytes = 2 units (floor division)
			},
			expectCost: 50,
		},
		{
			name: "zero time unit returns 0",
			endpoint: &protocol.VPNEndpoint{
				PricingMethod: protocol.PricingMethodTime,
				TimeUnitSecs:  0,
			},
			price:      50,
			expectCost: 0,
		},
		{
			name: "zero data unit returns 0",
			endpoint: &protocol.VPNEndpoint{
				PricingMethod: protocol.PricingMethodData,
				DataUnitBytes: 0,
			},
			price: 25,
			setup: func(m *UsageMeter) {
				m.AddTraffic(1000, 1000)
			},
			expectCost: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meter := NewUsageMeter(tt.endpoint, tt.price)
			if tt.setup != nil {
				tt.setup(meter)
			}
			if got := meter.CurrentCost(); got != tt.expectCost {
				t.Errorf("CurrentCost() = %v, want %v", got, tt.expectCost)
			}
		})
	}
}

func TestUsageMeter_ShouldRenewPayment(t *testing.T) {
	// Session pricing should never renew
	meterSession := NewUsageMeter(&protocol.VPNEndpoint{
		PricingMethod: protocol.PricingMethodSession,
		Price:         100,
	}, 100)
	if meterSession.ShouldRenewPayment(200, 80) {
		t.Error("session pricing ShouldRenewPayment returned true")
	}

	// Time-based with no usage should not renew
	meterTime := NewUsageMeter(&protocol.VPNEndpoint{
		PricingMethod: protocol.PricingMethodTime,
		TimeUnitSecs:  60,
		Price:         10,
	}, 10)
	if meterTime.ShouldRenewPayment(0, 0) {
		t.Error("time-based with no elapsed time ShouldRenewPayment returned true")
	}

	// Data-based with no traffic should not renew
	meterData := NewUsageMeter(&protocol.VPNEndpoint{
		PricingMethod: protocol.PricingMethodData,
		DataUnitBytes: 1000,
		Price:         10,
	}, 10)
	if meterData.ShouldRenewPayment(0, 0) {
		t.Error("data-based with no traffic ShouldRenewPayment returned true")
	}
}

func TestUsageMeter_ConcurrentAccess(t *testing.T) {
	endpoint := &protocol.VPNEndpoint{
		PricingMethod: protocol.PricingMethodData,
		DataUnitBytes: 1000,
	}
	meter := NewUsageMeter(endpoint, 25)

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				meter.AddTraffic(uint64(n*10+j), uint64(n*10+j+1))
			}
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic when reading
	total := meter.TotalBytes()
	cost := meter.CurrentCost()
	t.Logf("Total bytes: %d, cost: %d", total, cost)
}
