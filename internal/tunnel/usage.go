package tunnel

import (
	"sync"
	"time"

	"blockchain-vpn/internal/protocol"
)

// UsageMeter tracks time and data usage for a session according to the provider's pricing model.
type UsageMeter struct {
	mu             sync.Mutex
	startTime      time.Time
	bytesSent      uint64
	bytesReceived  uint64
	pricingMethod  uint8
	pricePerUnit   uint64
	timeUnitSecs   uint32
	dataUnitBytes  uint32
	maxSessionSecs int
}

// NewUsageMeter creates a meter for the given endpoint and price.
func NewUsageMeter(endpoint *protocol.VPNEndpoint, price uint64) *UsageMeter {
	// If endpoint has pricing method, use it; otherwise default to session
	method := endpoint.PricingMethod
	if method == 0 {
		method = protocol.PricingMethodSession
	}
	return &UsageMeter{
		startTime:      time.Now(),
		pricingMethod:  method,
		pricePerUnit:   price,
		timeUnitSecs:   endpoint.TimeUnitSecs,
		dataUnitBytes:  endpoint.DataUnitBytes,
		maxSessionSecs: int(endpoint.SessionTimeoutSecs),
	}
}

// AddTraffic increments the byte counters.
func (um *UsageMeter) AddTraffic(sent, received uint64) {
	um.mu.Lock()
	defer um.mu.Unlock()
	um.bytesSent += sent
	um.bytesReceived += received
}

// ElapsedSeconds returns the number of seconds since the session started.
func (um *UsageMeter) ElapsedSeconds() int64 {
	return int64(time.Since(um.startTime).Seconds())
}

// TotalBytes returns the sum of sent and received bytes.
func (um *UsageMeter) TotalBytes() uint64 {
	um.mu.Lock()
	defer um.mu.Unlock()
	return um.bytesSent + um.bytesReceived
}

// CurrentCost calculates the total amount due based on usage so far.
// For session: returns fixed price (only once, but caller should handle that)
// For time: returns (elapsed_seconds / time_unit_seconds) * price_per_unit
// For data: returns (total_bytes / data_unit_bytes) * price_per_unit
func (um *UsageMeter) CurrentCost() uint64 {
	um.mu.Lock()
	defer um.mu.Unlock()

	switch um.pricingMethod {
	case protocol.PricingMethodSession:
		// Session pricing: fixed fee, but we only charge once at start
		return um.pricePerUnit
	case protocol.PricingMethodTime:
		if um.timeUnitSecs == 0 {
			return 0
		}
		elapsedSec := uint64(um.ElapsedSeconds())
		units := elapsedSec / uint64(um.timeUnitSecs)
		return units * um.pricePerUnit
	case protocol.PricingMethodData:
		if um.dataUnitBytes == 0 {
			return 0
		}
		totalBytes := um.bytesSent + um.bytesReceived
		units := totalBytes / uint64(um.dataUnitBytes)
		return units * um.pricePerUnit
	default:
		return 0
	}
}

// ShouldRenewPayment returns true if the current accumulated cost indicates it's time to request another payment.
// This is called by the client to know when to send additional payments (for time/data models).
func (um *UsageMeter) ShouldRenewPayment(lastPaidCost uint64, thresholdPercent uint32) bool {
	current := um.CurrentCost()
	if current <= lastPaidCost {
		return false // no new usage yet
	}
	// thresholdPercent: e.g., 80 means renew when 80% of the next billing unit is reached
	if thresholdPercent == 0 {
		thresholdPercent = 80 // default
	}
	// Determine the billing increment
	var increment uint64
	switch um.pricingMethod {
	case protocol.PricingMethodTime:
		increment = um.pricePerUnit
	case protocol.PricingMethodData:
		increment = um.pricePerUnit
	default:
		return false // session doesn't need renewal
	}
	if increment == 0 {
		return false
	}
	// How much has been used in the current billing cycle?
	usedInCycle := current - lastPaidCost
	threshold := (increment * uint64(thresholdPercent)) / 100
	return usedInCycle >= threshold
}
