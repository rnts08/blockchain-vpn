package geoip

import (
	"log"
	"net"
	"sync"
	"time"

	"blockchain-vpn/internal/blockchain"
	"blockchain-vpn/internal/protocol"

	"github.com/oschwald/geoip2-golang"
)

// GeoIPDatabasePath is the path to the GeoLite2-Country.mmdb file.
// In a real application, this should be configurable.
const GeoIPDatabasePath = "GeoLite2-Country.mmdb"

// EnrichedVPNEndpoint extends VPNEndpoint with off-chain data like country.
type EnrichedVPNEndpoint struct {
	*blockchain.ProviderAnnouncement
	Country string
	Latency time.Duration
}

// EnrichEndpoints adds GeoIP country data to a list of endpoints.
func EnrichEndpoints(announcements []*blockchain.ProviderAnnouncement) []*EnrichedVPNEndpoint {
	db, err := geoip2.Open(GeoIPDatabasePath) // geoip2-golang is thread-safe
	if err != nil {
		log.Printf("Warning: Could not open GeoIP database at %q. Country data will be unavailable. Download it from MaxMind. Error: %v", GeoIPDatabasePath, err)
		// Return endpoints without country data if DB is missing
		enriched := make([]*EnrichedVPNEndpoint, len(announcements))
		for i, ann := range announcements {
			enriched[i] = &EnrichedVPNEndpoint{ProviderAnnouncement: ann, Country: "N/A"}
		}
		return enriched
	}
	defer db.Close()

	var wg sync.WaitGroup
	enrichedChan := make(chan *EnrichedVPNEndpoint, len(announcements))

	for _, ann := range announcements {
		wg.Add(1)
		go func(announcement *blockchain.ProviderAnnouncement) {
			defer wg.Done()

			// GeoIP Lookup
			record, err := db.Country(announcement.Endpoint.IP)
			country := "N/A"
			if err != nil {
				log.Printf("GeoIP lookup failed for %s: %v", announcement.Endpoint.IP, err)
			} else if record != nil && record.Country.IsoCode != "" {
				country = record.Country.IsoCode
			}

			// Latency Test
			latency := MeasureLatency(announcement.Endpoint)

			enrichedChan <- &EnrichedVPNEndpoint{ProviderAnnouncement: announcement, Country: country, Latency: latency}
		}(ann)
	}

	wg.Wait()
	close(enrichedChan)

	var enrichedEndpoints []*EnrichedVPNEndpoint
	for ep := range enrichedChan {
		enrichedEndpoints = append(enrichedEndpoints, ep)
	}

	return enrichedEndpoints
}

// MeasureLatency sends a small UDP packet to the endpoint and measures RTT.
// Returns a large duration if the endpoint is unreachable.
func MeasureLatency(endpoint *protocol.VPNEndpoint) time.Duration {
	targetAddr := &net.UDPAddr{IP: endpoint.IP, Port: int(endpoint.Port)}
	conn, err := net.DialUDP("udp", nil, targetAddr)
	if err != nil {
		return time.Hour // Unreachable
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(1 * time.Second)) // 1-second timeout for the test

	startTime := time.Now()
	if _, err := conn.Write([]byte("ping")); err != nil {
		return time.Hour // Unreachable
	}

	// Wait for a response to calculate RTT
	buf := make([]byte, 64)
	if _, _, err := conn.ReadFromUDP(buf); err != nil {
		return time.Hour // Timeout or error means unreachable
	}

	return time.Since(startTime)
}
