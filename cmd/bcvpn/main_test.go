package main

import (
	"net"
	"testing"
	"time"

	"blockchain-vpn/internal/blockchain"
	"blockchain-vpn/internal/geoip"
	"blockchain-vpn/internal/protocol"
)

func TestFilterEndpointsEmpty(t *testing.T) {
	t.Parallel()
	filtered := filterEndpoints(nil, "", 0, 0, 0, 0, "", 0)
	if len(filtered) != 0 {
		t.Fatalf("expected 0 filtered endpoints, got %d", len(filtered))
	}
}

func TestFilterEndpointsNoMatch(t *testing.T) {
	t.Parallel()
	endpoints := []*geoip.EnrichedVPNEndpoint{
		mkEndpoint("US", 1000, 50000, 10, 20*time.Millisecond),
	}
	filtered := filterEndpoints(endpoints, "XX", 500, 0, 0, 0, "", 0)
	if len(filtered) != 0 {
		t.Fatalf("expected 0 filtered endpoints, got %d", len(filtered))
	}
}

func TestFilterEndpointsPricingMethod(t *testing.T) {
	t.Parallel()
	endpoints := []*geoip.EnrichedVPNEndpoint{
		mkEndpoint("US", 1000, 50000, 10, 20*time.Millisecond),
	}
	filtered := filterEndpoints(endpoints, "", 0, 0, 0, 0, "session", 0)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered endpoint by pricing method, got %d", len(filtered))
	}
}

func TestSortEndpointsPrice(t *testing.T) {
	t.Parallel()
	endpoints := []*geoip.EnrichedVPNEndpoint{
		mkEndpoint("US", 5000, 10000, 10, 20*time.Millisecond),
		mkEndpoint("US", 1000, 10000, 10, 20*time.Millisecond),
		mkEndpoint("US", 3000, 10000, 10, 20*time.Millisecond),
	}
	sortEndpoints(endpoints, "price")
	if endpoints[0].Endpoint.Price != 1000 || endpoints[2].Endpoint.Price != 5000 {
		t.Fatalf("expected sorted by price ascending")
	}
}

func TestSortEndpointsLatency(t *testing.T) {
	t.Parallel()
	endpoints := []*geoip.EnrichedVPNEndpoint{
		mkEndpoint("US", 1000, 10000, 10, 200*time.Millisecond),
		mkEndpoint("US", 1000, 10000, 10, 20*time.Millisecond),
	}
	sortEndpoints(endpoints, "latency")
	if endpoints[0].Latency > endpoints[1].Latency {
		t.Fatalf("expected sorted by latency ascending")
	}
}

func TestSortEndpointsCapacity(t *testing.T) {
	t.Parallel()
	endpoints := []*geoip.EnrichedVPNEndpoint{
		mkEndpoint("US", 1000, 10000, 2, 20*time.Millisecond),
		mkEndpoint("US", 1000, 10000, 20, 20*time.Millisecond),
	}
	sortEndpoints(endpoints, "capacity")
	if endpoints[0].MaxConsumers < endpoints[1].MaxConsumers {
		t.Fatalf("expected sorted by capacity descending")
	}
}

func TestEffectiveCountryOverride(t *testing.T) {
	t.Parallel()
	endpoints := []*geoip.EnrichedVPNEndpoint{
		{
			ProviderAnnouncement: &blockchain.ProviderAnnouncement{
				DeclaredCountry: "",
				Endpoint:        &protocol.VPNEndpoint{IP: net.ParseIP("1.2.3.4")},
			},
			Country: "XX",
		},
	}
	if effectiveCountry(endpoints[0]) != "XX" {
		t.Fatalf("expected geoip country when declared is empty, got %s", effectiveCountry(endpoints[0]))
	}
}

func TestComputeProviderScoreZeroPrice(t *testing.T) {
	t.Parallel()
	endpoint := mkEndpoint("US", 0, 50000, 10, 20*time.Millisecond)
	score := computeProviderScore(endpoint)
	if score <= 0 {
		t.Fatalf("expected positive score for zero price")
	}
}

func TestComputeProviderScoreZeroLatency(t *testing.T) {
	t.Parallel()
	endpoint := mkEndpoint("US", 1000, 50000, 10, 0)
	score := computeProviderScore(endpoint)
	if score <= 0 {
		t.Fatalf("expected positive score for zero latency")
	}
}

func TestComputeProviderScoreMaxConsumers(t *testing.T) {
	t.Parallel()
	highCap := mkEndpoint("US", 1000, 50000, 100, 20*time.Millisecond)
	lowCap := mkEndpoint("US", 1000, 50000, 1, 20*time.Millisecond)
	if computeProviderScore(highCap) <= computeProviderScore(lowCap) {
		t.Fatalf("expected higher score for higher capacity")
	}
}

func BenchmarkFilterEndpoints(b *testing.B) {
	endpoints := make([]*geoip.EnrichedVPNEndpoint, 100)
	for i := 0; i < 100; i++ {
		endpoints[i] = mkEndpoint("US", uint64(i*100), 50000, 10, time.Duration(i)*time.Millisecond)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filterEndpoints(endpoints, "", 0, 0, 0, 0, "", 0)
	}
}

func BenchmarkSortEndpoints(b *testing.B) {
	endpoints := make([]*geoip.EnrichedVPNEndpoint, 100)
	for i := 0; i < 100; i++ {
		endpoints[i] = mkEndpoint("US", uint64(i*100), uint32(50000-i*100), uint16(10-i/10), time.Duration(i)*time.Millisecond)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sorted := make([]*geoip.EnrichedVPNEndpoint, len(endpoints))
		copy(sorted, endpoints)
		sortEndpoints(sorted, "score")
	}
}

func BenchmarkComputeProviderScore(b *testing.B) {
	endpoint := mkEndpoint("US", 1000, 50000, 10, 20*time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		computeProviderScore(endpoint)
	}
}

func mkEndpointWithReputation(country string, price uint64, bw uint32, cap uint16, latency time.Duration, repScore int) *geoip.EnrichedVPNEndpoint {
	ep := mkEndpoint(country, price, bw, cap, latency)
	ep.ReputationScore = repScore
	return ep
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
