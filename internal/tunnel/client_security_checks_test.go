package tunnel

import (
	"context"
	"net"
	"strings"
	"testing"

	"blockchain-vpn/internal/geoip"
)

func TestRunClientPostConnectChecks_StrictFailsWhenIPUnchanged(t *testing.T) {
	restore := mockSecurityCheckFns(
		func() (net.IP, error) { return net.ParseIP("203.0.113.5"), nil },
		func() (*geoip.GeoLocation, error) { return &geoip.GeoLocation{CountryCode: "US"}, nil },
		func(context.Context) (uint32, error) { return 50000, nil },
	)
	defer restore()

	err := runClientPostConnectChecks(context.Background(), ClientSecurityExpectations{
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
		func(context.Context) (uint32, error) { return 50000, nil },
	)
	defer restore()

	err := runClientPostConnectChecks(context.Background(), ClientSecurityExpectations{
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
		func(context.Context) (uint32, error) { return 1, nil },
	)
	defer restore()

	if err := runClientPostConnectChecks(context.Background(), ClientSecurityExpectations{
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
	throughput func(context.Context) (uint32, error),
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
