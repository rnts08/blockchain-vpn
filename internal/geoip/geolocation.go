package geoip

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GeoLocation represents the JSON response from the geolocation API.
type GeoLocation struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Zip         string  `json:"zip"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
	AS          string  `json:"as"`
	Query       string  `json:"query"` // The IP address
}

// AutoLocate fetches the geographical location of the current machine
// using an external API (ip-api.com).
func AutoLocate() (*GeoLocation, error) {
	// ip-api.com provides a free endpoint for non-commercial use (up to 45 requests/minute).
	url := "http://ip-api.com/json/"
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to contact geolocation API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geolocation API returned status: %s", resp.Status)
	}

	var loc GeoLocation
	if err := json.NewDecoder(resp.Body).Decode(&loc); err != nil {
		return nil, fmt.Errorf("failed to decode geolocation response: %w", err)
	}

	if loc.Status == "fail" {
		return nil, fmt.Errorf("geolocation API returned failure status")
	}

	return &loc, nil
}
