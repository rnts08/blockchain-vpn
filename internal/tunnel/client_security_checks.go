package tunnel

import (
	"context"
	"log"
	"net"
	"strings"
	"time"

	"blockchain-vpn/internal/geoip"
	"blockchain-vpn/internal/util"
)

func runClientPostConnectChecks(ctx context.Context, providerHost string, expectedCountry string, preConnectIP net.IP) {
	log.Printf("Running post-connect security checks...")
	checkDNSReachability(ctx)

	postConnectIP, err := util.GetPublicIP()
	if err != nil {
		log.Printf("Security check warning: could not determine post-connect egress IP: %v", err)
		return
	}
	log.Printf("Security check: detected egress IP %s", postConnectIP.String())

	if preConnectIP != nil {
		if preConnectIP.Equal(postConnectIP) {
			log.Printf("Security check warning: egress IP did not change (%s). Tunnel may not be active.", postConnectIP.String())
		} else {
			log.Printf("Security check: egress IP changed from %s to %s", preConnectIP.String(), postConnectIP.String())
		}
	}

	if ip := net.ParseIP(strings.TrimSpace(providerHost)); ip != nil && !ip.Equal(postConnectIP) {
		log.Printf("Security check warning: egress IP %s does not match provider endpoint IP %s", postConnectIP.String(), ip.String())
	}

	checkDNSLeakHeuristic(ctx, postConnectIP)
	checkCountryHeuristic(postConnectIP, expectedCountry)
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

func checkDNSLeakHeuristic(ctx context.Context, egressIP net.IP) {
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
		log.Printf("Security check warning: DNS resolver egress %s differs from tunnel egress %s (possible DNS leak)", dnsObservedIP.String(), egressIP.String())
		return
	}
	log.Printf("Security check: DNS leak heuristic passed (resolver egress matches tunnel egress)")
}

func checkCountryHeuristic(egressIP net.IP, expectedCountry string) {
	expected := strings.ToUpper(strings.TrimSpace(expectedCountry))
	if expected == "" || expected == "N/A" {
		log.Printf("Security check: provider country verification skipped (no expected country)")
		return
	}

	loc, err := geoip.AutoLocate()
	if err != nil {
		log.Printf("Security check warning: could not geolocate egress IP %s: %v", egressIP.String(), err)
		return
	}
	detected := strings.ToUpper(strings.TrimSpace(loc.CountryCode))
	if detected == "" {
		log.Printf("Security check warning: geolocation returned empty country code")
		return
	}
	if detected != expected {
		log.Printf("Security check warning: provider country mismatch (expected %s, detected %s)", expected, detected)
		return
	}
	log.Printf("Security check: provider country matched expected %s", expected)
}
