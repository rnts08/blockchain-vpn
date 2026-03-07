package tunnel

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"blockchain-vpn/internal/geoip"
	"blockchain-vpn/internal/util"
)

type ClientSecurityExpectations struct {
	ProviderHost          string
	ExpectedCountry       string
	ExpectedBandwidthKB   uint32
	StrictVerification    bool
	VerifyThroughputAfter bool
}

var (
	getPublicIPFn       = util.GetPublicIP
	autoLocateFn        = geoip.AutoLocate
	measureThroughputFn = MeasureProviderThroughputKbps
)

func runClientPostConnectChecks(ctx context.Context, expected ClientSecurityExpectations, preConnectIP net.IP) error {
	log.Printf("Running post-connect security checks...")
	checkDNSReachability(ctx)
	checkDNSConfiguredServers()

	postConnectIP, err := getPublicIPFn()
	if err != nil {
		if expected.StrictVerification {
			return fmt.Errorf("strict verification failed: could not determine post-connect egress IP: %w", err)
		}
		log.Printf("Security check warning: could not determine post-connect egress IP: %v", err)
		return nil
	}
	log.Printf("Security check: detected egress IP %s", postConnectIP.String())

	if err := verifyEgressIP(preConnectIP, postConnectIP, expected); err != nil {
		return err
	}
	checkDNSLeakHeuristic(ctx, postConnectIP, expected.StrictVerification)
	if err := checkCountryHeuristic(postConnectIP, expected.ExpectedCountry, expected.StrictVerification); err != nil {
		return err
	}
	if expected.VerifyThroughputAfter && expected.ExpectedBandwidthKB > 0 {
		if err := verifyProviderThroughput(ctx, expected.ProviderHost, expected.ExpectedBandwidthKB, expected.StrictVerification); err != nil {
			return err
		}
	}
	return nil
}

func checkDNSReachability(ctx context.Context) {
	resolveCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupHost(resolveCtx, "example.com")
	if err != nil || len(ips) == 0 {
		log.Printf("Security check warning: DNS lookup failed after connect: %v", err)
		return
	}
	log.Printf("Security check: DNS reachability OK")
}

func checkDNSConfiguredServers() {
	servers, err := readConfiguredDNSServers()
	if err != nil {
		log.Printf("Security check warning: DNS server introspection unavailable: %v", err)
		return
	}
	if len(servers) == 0 {
		log.Printf("Security check warning: DNS introspection found no configured resolvers")
		return
	}
	if !hasExpectedSecureDNS(servers) {
		log.Printf("Security check warning: DNS resolvers do not include expected secure servers (1.1.1.1/8.8.8.8): %v", servers)
		return
	}
	log.Printf("Security check: DNS introspection found expected secure resolvers")
}

func verifyEgressIP(preConnectIP, postConnectIP net.IP, expected ClientSecurityExpectations) error {
	if preConnectIP != nil {
		if preConnectIP.Equal(postConnectIP) {
			msg := fmt.Sprintf("egress IP did not change (%s). Tunnel may not be active", postConnectIP.String())
			if expected.StrictVerification {
				return fmt.Errorf("strict verification failed: %s", msg)
			}
			log.Printf("Security check warning: %s", msg)
		} else {
			log.Printf("Security check: egress IP changed from %s to %s", preConnectIP.String(), postConnectIP.String())
		}
	}

	if ip := net.ParseIP(strings.TrimSpace(expected.ProviderHost)); ip != nil && !ip.Equal(postConnectIP) {
		msg := fmt.Sprintf("egress IP %s does not match provider endpoint IP %s", postConnectIP.String(), ip.String())
		if expected.StrictVerification {
			return fmt.Errorf("strict verification failed: %s", msg)
		}
		log.Printf("Security check warning: %s", msg)
	}
	return nil
}

func checkDNSLeakHeuristic(ctx context.Context, egressIP net.IP, strict bool) {
	resolveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	txt, err := net.DefaultResolver.LookupTXT(resolveCtx, "whoami.cloudflare")
	if err != nil {
		log.Printf("Security check warning: DNS leak heuristic unavailable (whoami lookup failed): %v", err)
		return
	}
	var dnsObservedIP net.IP
	for _, rec := range txt {
		if ip := net.ParseIP(strings.TrimSpace(rec)); ip != nil {
			dnsObservedIP = ip
			break
		}
	}
	if dnsObservedIP == nil {
		log.Printf("Security check warning: DNS leak heuristic did not return an IP result")
		return
	}
	if egressIP != nil && !dnsObservedIP.Equal(egressIP) {
		msg := fmt.Sprintf("DNS resolver egress %s differs from tunnel egress %s (possible DNS leak)", dnsObservedIP.String(), egressIP.String())
		if strict {
			log.Printf("Security check warning: %s", msg)
			return
		}
		log.Printf("Security check warning: %s", msg)
		return
	}
	log.Printf("Security check: DNS leak heuristic passed (resolver egress matches tunnel egress)")
}

func checkCountryHeuristic(egressIP net.IP, expectedCountry string, strict bool) error {
	expected := strings.ToUpper(strings.TrimSpace(expectedCountry))
	if expected == "" || expected == "N/A" {
		log.Printf("Security check: provider country verification skipped (no expected country)")
		return nil
	}

	loc, err := autoLocateFn()
	if err != nil {
		msg := fmt.Sprintf("could not geolocate egress IP %s: %v", egressIP.String(), err)
		if strict {
			return fmt.Errorf("strict verification failed: %s", msg)
		}
		log.Printf("Security check warning: %s", msg)
		return nil
	}
	detected := strings.ToUpper(strings.TrimSpace(loc.CountryCode))
	if detected == "" {
		msg := "geolocation returned empty country code"
		if strict {
			return fmt.Errorf("strict verification failed: %s", msg)
		}
		log.Printf("Security check warning: %s", msg)
		return nil
	}
	if detected != expected {
		msg := fmt.Sprintf("provider country mismatch (expected %s, detected %s)", expected, detected)
		if strict {
			return fmt.Errorf("strict verification failed: %s", msg)
		}
		log.Printf("Security check warning: %s", msg)
		return nil
	}
	log.Printf("Security check: provider country matched expected %s", expected)
	return nil
}

func verifyProviderThroughput(ctx context.Context, endpointHost string, expectedBandwidthKB uint32, strict bool) error {
	// Extract just the IP if it includes a port
	host := endpointHost
	if h, _, err := net.SplitHostPort(endpointHost); err == nil {
		host = h
	}

	addr := fmt.Sprintf("%s:51821", host) // By default, throughput port is 51821 // TODO: pass actual port via ProviderAnnouncement
	measured, err := MeasureProviderThroughputKbps(ctx, addr)
	if err != nil {
		msg := fmt.Sprintf("provider-assisted throughput verification unavailable or failed: %v", err)
		if strict {
			return fmt.Errorf("strict verification failed: %s", msg)
		}
		log.Printf("Security check warning: %s", msg)
		return nil
	}
	log.Printf("Security check: measured provider-assisted downstream throughput %d Kbps", measured)
	threshold := float64(expectedBandwidthKB) * 0.50
	if float64(measured) < threshold {
		msg := fmt.Sprintf("measured throughput %d Kbps below expected threshold %.0f Kbps", measured, threshold)
		if strict {
			return fmt.Errorf("strict verification failed: %s", msg)
		}
		log.Printf("Security check warning: %s", msg)
		return nil
	}
	log.Printf("Security check: provider throughput verification passed against advertised %d Kbps", expectedBandwidthKB)
	return nil
}
