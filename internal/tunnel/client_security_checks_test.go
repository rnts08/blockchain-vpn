package tunnel

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"blockchain-vpn/internal/geoip"
)

func TestRunClientPostConnectChecks_StrictFailsWhenIPUnchanged(t *testing.T) {
	restore := mockSecurityCheckFns(
		func() (net.IP, error) { return net.ParseIP("203.0.113.5"), nil },
		func() (*geoip.GeoLocation, error) { return &geoip.GeoLocation{CountryCode: "US"}, nil },
		func(context.Context, string) (uint32, error) { return 50000, nil },
	)
	defer restore()

	_, err := runClientPostConnectChecks(context.Background(), ClientSecurityExpectations{
		ExpectedCountry:       "US",
		StrictVerification:    true,
		ExpectedBandwidthKB:   1000,
		VerifyThroughputAfter: true,
	}, net.ParseIP("203.0.113.5"))
	if err == nil || !strings.Contains(err.Error(), "did not change") {
		t.Fatalf("expected strict unchanged-IP failure, got: %v", err)
	}
}

func TestRunClientPostConnectChecks_StrictFailsOnCountryMismatch(t *testing.T) {
	restore := mockSecurityCheckFns(
		func() (net.IP, error) { return net.ParseIP("203.0.113.10"), nil },
		func() (*geoip.GeoLocation, error) { return &geoip.GeoLocation{CountryCode: "DE"}, nil },
		func(context.Context, string) (uint32, error) { return 50000, nil },
	)
	defer restore()

	_, err := runClientPostConnectChecks(context.Background(), ClientSecurityExpectations{
		ExpectedCountry:       "US",
		StrictVerification:    true,
		ExpectedBandwidthKB:   1000,
		VerifyThroughputAfter: true,
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "country mismatch") {
		t.Fatalf("expected strict country mismatch failure, got: %v", err)
	}
}

func TestRunClientPostConnectChecks_NonStrictAllowsWarnings(t *testing.T) {
	restore := mockSecurityCheckFns(
		func() (net.IP, error) { return net.ParseIP("203.0.113.10"), nil },
		func() (*geoip.GeoLocation, error) { return &geoip.GeoLocation{CountryCode: "DE"}, nil },
		func(context.Context, string) (uint32, error) { return 1, nil },
	)
	defer restore()

	if _, err := runClientPostConnectChecks(context.Background(), ClientSecurityExpectations{
		ExpectedCountry:       "US",
		StrictVerification:    false,
		ExpectedBandwidthKB:   1000000,
		VerifyThroughputAfter: true,
	}, net.ParseIP("203.0.113.9")); err != nil {
		t.Fatalf("expected non-strict mode to continue on warnings, got: %v", err)
	}
}

func mockSecurityCheckFns(
	pubIP func() (net.IP, error),
	locate func() (*geoip.GeoLocation, error),
	throughput func(context.Context, string) (uint32, error),
) func() {
	prevPub := getPublicIPFn
	prevLoc := autoLocateFn
	prevMeasure := measureThroughputFn
	getPublicIPFn = pubIP
	autoLocateFn = locate
	measureThroughputFn = throughput
	return func() {
		getPublicIPFn = prevPub
		autoLocateFn = prevLoc
		measureThroughputFn = prevMeasure
	}
}

func TestCheckConnectionQuality_AllPassed(t *testing.T) {
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
	if !quality.Passed {
		t.Errorf("expected quality to pass, score=%.2f, threshold=%.2f", quality.QualityScore, quality.QualityThreshold)
	}
	if !quality.EgressIPVerified {
		t.Error("expected egress IP to be verified (changed)")
	}
	if !quality.DNSLeakPassed {
		t.Error("expected DNS leak to pass")
	}
	if !quality.CountryVerified {
		t.Error("expected country to be verified")
	}
}

func TestCheckConnectionQuality_DefaultThreshold(t *testing.T) {
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
		QualityThreshold:      0, // Should default to 0.75
	}, net.ParseIP("203.0.113.5"))
	if err != nil {
		t.Fatalf("CheckConnectionQuality failed: %v", err)
	}
	if quality.QualityThreshold != 0.75 {
		t.Errorf("expected default threshold 0.75, got %.2f", quality.QualityThreshold)
	}
}

func TestCheckConnectionQuality_IPUnchanged(t *testing.T) {
	restore := mockSecurityCheckFns(
		func() (net.IP, error) { return net.ParseIP("203.0.113.5"), nil },
		func() (*geoip.GeoLocation, error) { return &geoip.GeoLocation{CountryCode: "US"}, nil },
		func(context.Context, string) (uint32, error) { return 50000, nil },
	)
	defer restore()

	quality, err := CheckConnectionQuality(context.Background(), ClientSecurityExpectations{
		ExpectedCountry:       "US",
		ExpectedBandwidthKB:   10000,
		VerifyThroughputAfter: true,
		ThroughputProbePort:   51821,
	}, net.ParseIP("203.0.113.5"))
	if err != nil {
		t.Fatalf("CheckConnectionQuality failed: %v", err)
	}
	if quality.EgressIPVerified {
		t.Error("expected egress IP not verified when unchanged")
	}
	if quality.QualityScore >= 1.0 {
		t.Error("expected quality score to decrease for unchanged IP")
	}
}

func TestCheckConnectionQuality_BandwidthBelowThreshold(t *testing.T) {
	restore := mockSecurityCheckFns(
		func() (net.IP, error) { return net.ParseIP("203.0.113.10"), nil },
		func() (*geoip.GeoLocation, error) { return &geoip.GeoLocation{CountryCode: "US"}, nil },
		func(context.Context, string) (uint32, error) { return 500, nil }, // Very low: 5% of expected
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
	if quality.BandwidthPercentage >= 0.75 {
		t.Errorf("expected bandwidth percentage below 0.75, got %.2f", quality.BandwidthPercentage)
	}
	if quality.QualityScore >= 0.75 {
		t.Error("expected quality score to decrease for low bandwidth")
	}
}

func TestCheckConnectionQuality_QualityScoreClamped(t *testing.T) {
	restore := mockSecurityCheckFns(
		func() (net.IP, error) { return net.ParseIP("203.0.113.5"), nil },
		func() (*geoip.GeoLocation, error) { return nil, fmt.Errorf("geolocation failed") },
		func(context.Context, string) (uint32, error) { return 1, nil },
	)
	defer restore()

	quality, err := CheckConnectionQuality(context.Background(), ClientSecurityExpectations{
		ExpectedCountry:       "US",
		ExpectedBandwidthKB:   1000000,
		VerifyThroughputAfter: true,
		ThroughputProbePort:   51821,
		QualityThreshold:      0.5,
	}, net.ParseIP("203.0.113.5"))
	if err != nil {
		t.Fatalf("CheckConnectionQuality failed: %v", err)
	}
	if quality.QualityScore >= 0.5 {
		t.Errorf("expected quality score below threshold 0.5, got %.2f", quality.QualityScore)
	}
	if quality.QualityScore < 0 {
		t.Errorf("expected quality score >= 0, got %.2f", quality.QualityScore)
	}
}

func TestCheckConnectionQuality_CountrySkippedWhenNoExpectation(t *testing.T) {
	restore := mockSecurityCheckFns(
		func() (net.IP, error) { return net.ParseIP("203.0.113.10"), nil },
		func() (*geoip.GeoLocation, error) { return nil, nil },
		func(context.Context, string) (uint32, error) { return 50000, nil },
	)
	defer restore()

	quality, err := CheckConnectionQuality(context.Background(), ClientSecurityExpectations{
		ExpectedCountry:       "", // No expectation
		ExpectedBandwidthKB:   10000,
		VerifyThroughputAfter: true,
		ThroughputProbePort:   51821,
	}, net.ParseIP("203.0.113.5"))
	if err != nil {
		t.Fatalf("CheckConnectionQuality failed: %v", err)
	}
	if !quality.CountryVerified {
		t.Error("expected country to be skipped when no expectation")
	}
}

func TestCheckConnectionQuality_CountrySkippedNA(t *testing.T) {
	restore := mockSecurityCheckFns(
		func() (net.IP, error) { return net.ParseIP("203.0.113.10"), nil },
		func() (*geoip.GeoLocation, error) { return nil, nil },
		func(context.Context, string) (uint32, error) { return 50000, nil },
	)
	defer restore()

	quality, err := CheckConnectionQuality(context.Background(), ClientSecurityExpectations{
		ExpectedCountry:       "N/A",
		ExpectedBandwidthKB:   10000,
		VerifyThroughputAfter: true,
		ThroughputProbePort:   51821,
	}, net.ParseIP("203.0.113.5"))
	if err != nil {
		t.Fatalf("CheckConnectionQuality failed: %v", err)
	}
	if !quality.CountryVerified {
		t.Error("expected country to be skipped when N/A")
	}
}

func TestCheckConnectionQuality_NilPreConnectIP(t *testing.T) {
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
	}, nil)
	if err != nil {
		t.Fatalf("CheckConnectionQuality failed: %v", err)
	}
	if !quality.EgressIPVerified {
		t.Error("expected egress IP to be verified when preConnectIP is nil")
	}
}

func TestCheckConnectionQuality_BandwidthNoVerify(t *testing.T) {
	restore := mockSecurityCheckFns(
		func() (net.IP, error) { return net.ParseIP("203.0.113.10"), nil },
		func() (*geoip.GeoLocation, error) { return &geoip.GeoLocation{CountryCode: "US"}, nil },
		func(context.Context, string) (uint32, error) { return 50000, nil },
	)
	defer restore()

	quality, err := CheckConnectionQuality(context.Background(), ClientSecurityExpectations{
		ExpectedCountry:       "US",
		ExpectedBandwidthKB:   10000,
		VerifyThroughputAfter: false, // No throughput verification
		ThroughputProbePort:   51821,
	}, net.ParseIP("203.0.113.5"))
	if err != nil {
		t.Fatalf("CheckConnectionQuality failed: %v", err)
	}
	if quality.BandwidthKB != 0 {
		t.Errorf("expected bandwidth KB to be 0 when not verified, got %d", quality.BandwidthKB)
	}
}
