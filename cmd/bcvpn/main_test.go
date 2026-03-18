package main

import (
	"net"
	"testing"
	"time"

	"blockchain-vpn/internal/blockchain"
	"blockchain-vpn/internal/geoip"
	"blockchain-vpn/internal/protocol"
)

func TestFilterEndpoints(t *testing.T) {
	endpoints := []*geoip.EnrichedVPNEndpoint{
		mkEndpoint("US", 1000, 50000, 10, 20*time.Millisecond),
		mkEndpoint("DE", 3000, 10000, 2, 120*time.Millisecond),
	}
	filtered := filterEndpoints(endpoints, "US", 1500, 20000, 50*time.Millisecond, 5, "", 0)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered endpoint, got %d", len(filtered))
	}
	if effectiveCountry(filtered[0]) != "US" {
		t.Fatalf("unexpected country: %s", effectiveCountry(filtered[0]))
	}
}

func TestSortEndpointsBandwidth(t *testing.T) {
	endpoints := []*geoip.EnrichedVPNEndpoint{
		mkEndpoint("US", 1000, 10000, 10, 20*time.Millisecond),
		mkEndpoint("US", 1000, 50000, 10, 20*time.Millisecond),
	}
	sortEndpoints(endpoints, "bandwidth")
	if endpoints[0].AdvertisedBandwidthKB != 50000 {
		t.Fatalf("expected highest bandwidth first")
	}
}

func TestComputeProviderScorePrefersBetterProvider(t *testing.T) {
	slowExpensive := mkEndpoint("US", 5000, 10000, 2, 200*time.Millisecond)
	fastCheap := mkEndpoint("US", 1000, 50000, 20, 20*time.Millisecond)
	if computeProviderScore(fastCheap) <= computeProviderScore(slowExpensive) {
		t.Fatalf("expected fast/cheap endpoint to score higher")
	}
}

func mkEndpoint(country string, price uint64, bw uint32, cap uint16, latency time.Duration) *geoip.EnrichedVPNEndpoint {
	return &geoip.EnrichedVPNEndpoint{
		ProviderAnnouncement: &blockchain.ProviderAnnouncement{
			Endpoint: &protocol.VPNEndpoint{
				IP:                 net.ParseIP("198.51.100.1"),
				Port:               51820,
				Price:              price,
				PricingMethod:      protocol.PricingMethodSession,
				TimeUnitSecs:       60,
				DataUnitBytes:      1_000_000,
				SessionTimeoutSecs: 0,
			},
			DeclaredCountry:       country,
			AdvertisedBandwidthKB: bw,
			MaxConsumers:          cap,
		},
		Country: country,
		Latency: latency,
	}
}
