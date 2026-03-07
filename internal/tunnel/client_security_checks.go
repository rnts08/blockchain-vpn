package tunnel

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
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
	getPublicIPFn = util.GetPublicIP
	autoLocateFn  = geoip.AutoLocate
	httpClientFn  = func(timeout time.Duration) *http.Client {
		return &http.Client{Timeout: timeout}
	}
	measureThroughputFn = measureDownloadThroughputKbps
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
		if err := checkThroughput(ctx, expected.ExpectedBandwidthKB, expected.StrictVerification); err != nil {
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

func checkThroughput(ctx context.Context, expectedBandwidthKB uint32, strict bool) error {
	measured, err := measureThroughputFn(ctx)
	if err != nil {
		msg := fmt.Sprintf("throughput verification unavailable: %v", err)
		if strict {
			return fmt.Errorf("strict verification failed: %s", msg)
		}
		log.Printf("Security check warning: %s", msg)
		return nil
	}
	log.Printf("Security check: measured downstream throughput %d Kbps", measured)
	threshold := float64(expectedBandwidthKB) * 0.50
	if float64(measured) < threshold {
		msg := fmt.Sprintf("measured throughput %d Kbps below expected threshold %.0f Kbps", measured, threshold)
		if strict {
			return fmt.Errorf("strict verification failed: %s", msg)
		}
		log.Printf("Security check warning: %s", msg)
		return nil
	}
	log.Printf("Security check: throughput verification passed against advertised %d Kbps", expectedBandwidthKB)
	return nil
}

func measureDownloadThroughputKbps(ctx context.Context) (uint32, error) {
	sources := []string{
		"https://speed.cloudflare.com/__down?bytes=4000000",
		"https://speed.hetzner.de/10MB.bin",
		"https://proof.ovh.net/files/1Mb.dat",
	}
	for _, u := range sources {
		val, err := measureURLThroughputKbps(ctx, u)
		if err == nil {
			return val, nil
		}
	}
	return 0, fmt.Errorf("all throughput sources failed")
}

func measureURLThroughputKbps(ctx context.Context, url string) (uint32, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	start := time.Now()
	resp, err := httpClientFn(8 * time.Second).Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("status %s", resp.Status)
	}
	n, err := io.CopyN(io.Discard, resp.Body, 2*1024*1024)
	if err != nil && err != io.EOF {
		return 0, err
	}
	elapsed := time.Since(start)
	if elapsed <= 0 {
		return 0, fmt.Errorf("invalid elapsed time")
	}
	kbps := uint32(float64(n*8) / elapsed.Seconds() / 1000.0)
	if kbps == 0 {
		return 0, fmt.Errorf("measured zero throughput")
	}
	return kbps, nil
}
