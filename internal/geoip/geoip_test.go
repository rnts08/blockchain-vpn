package geoip

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"blockchain-vpn/internal/blockchain"
	"blockchain-vpn/internal/protocol"
)

func TestGeoLocationJSON(t *testing.T) {
	data := `{
		"status": "success",
		"country": "United States",
		"countryCode": "US",
		"region": "CA",
		"regionName": "California",
		"city": "San Francisco",
		"zip": "94105",
		"lat": 37.7898,
		"lon": -122.3942,
		"timezone": "America/Los_Angeles",
		"isp": "Test ISP",
		"org": "Test Org",
		"as": "AS12345",
		"query": "1.2.3.4"
	}`

	var loc GeoLocation
	if err := json.Unmarshal([]byte(data), &loc); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if loc.Status != "success" {
		t.Errorf("expected status 'success', got %q", loc.Status)
	}
	if loc.CountryCode != "US" {
		t.Errorf("expected country code 'US', got %q", loc.CountryCode)
	}
	if loc.City != "San Francisco" {
		t.Errorf("expected city 'San Francisco', got %q", loc.City)
	}
	if loc.Lat != 37.7898 {
		t.Errorf("expected lat 37.7898, got %f", loc.Lat)
	}
	if loc.Query != "1.2.3.4" {
		t.Errorf("expected query '1.2.3.4', got %q", loc.Query)
	}
}

func TestGeoLocationFailStatus(t *testing.T) {
	data := `{"status": "fail", "message": "invalid query"}`

	var loc GeoLocation
	if err := json.Unmarshal([]byte(data), &loc); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if loc.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", loc.Status)
	}
}

func TestMeasureLatencyTimeout(t *testing.T) {
	t.Parallel()

	endpoint := &protocol.VPNEndpoint{
		IP:   net.ParseIP("10.255.255.1"),
		Port: 51820,
	}

	start := time.Now()
	latency := MeasureLatency(endpoint)
	elapsed := time.Since(start)

	if latency != time.Hour {
		t.Errorf("expected timeout (1 hour), got %v", latency)
	}

	if elapsed > 2*time.Second {
		t.Errorf("measureLatency took too long: %v", elapsed)
	}
}

func TestGeoLocationInvalidJSON(t *testing.T) {
	invalidJSON := []byte(`not valid json`)
	var loc GeoLocation
	err := json.Unmarshal(invalidJSON, &loc)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestEnrichedVPNEndpoint(t *testing.T) {
	ann := &blockchain.ProviderAnnouncement{}
	enriched := &EnrichedVPNEndpoint{
		ProviderAnnouncement: ann,
		Country:              "US",
		Latency:              50 * time.Millisecond,
	}

	if enriched.Country != "US" {
		t.Errorf("expected country 'US', got %q", enriched.Country)
	}
	if enriched.Latency != 50*time.Millisecond {
		t.Errorf("expected latency 50ms, got %v", enriched.Latency)
	}
}

func TestEnrichEndpointsEmpty(t *testing.T) {
	t.Parallel()

	announcements := []*blockchain.ProviderAnnouncement{}
	enriched := EnrichEndpoints(announcements)

	if len(enriched) != 0 {
		t.Errorf("expected 0 enriched endpoints, got %d", len(enriched))
	}
}

func TestGeoLocationAllFields(t *testing.T) {
	data := `{
		"status": "success",
		"country": "Germany",
		"countryCode": "DE",
		"region": "BY",
		"regionName": "Bavaria",
		"city": "Munich",
		"zip": "80331",
		"lat": 48.1371,
		"lon": 11.5764,
		"timezone": "Europe/Berlin",
		"isp": "Deutsche Telekom",
		"org": "Telekom Deutschland GmbH",
		"as": "AS3320",
		"query": "8.8.8.8"
	}`

	var loc GeoLocation
	if err := json.Unmarshal([]byte(data), &loc); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if loc.Country != "Germany" {
		t.Errorf("expected country 'Germany', got %q", loc.Country)
	}
	if loc.Region != "BY" {
		t.Errorf("expected region 'BY', got %q", loc.Region)
	}
	if loc.RegionName != "Bavaria" {
		t.Errorf("expected regionName 'Bavaria', got %q", loc.RegionName)
	}
	if loc.City != "Munich" {
		t.Errorf("expected city 'Munich', got %q", loc.City)
	}
	if loc.Zip != "80331" {
		t.Errorf("expected zip '80331', got %q", loc.Zip)
	}
	if loc.Lon != 11.5764 {
		t.Errorf("expected lon 11.5764, got %f", loc.Lon)
	}
	if loc.Timezone != "Europe/Berlin" {
		t.Errorf("expected timezone 'Europe/Berlin', got %q", loc.Timezone)
	}
	if loc.ISP != "Deutsche Telekom" {
		t.Errorf("expected ISP 'Deutsche Telekom', got %q", loc.ISP)
	}
	if loc.Org != "Telekom Deutschland GmbH" {
		t.Errorf("expected org 'Telekom Deutschland GmbH', got %q", loc.Org)
	}
	if loc.AS != "AS3320" {
		t.Errorf("expected AS 'AS3320', got %q", loc.AS)
	}
}

func TestGeoIPDatabasePath(t *testing.T) {
	if GeoIPDatabasePath != "GeoLite2-Country.mmdb" {
		t.Errorf("expected GeoIPDatabasePath 'GeoLite2-Country.mmdb', got %q", GeoIPDatabasePath)
	}
}
