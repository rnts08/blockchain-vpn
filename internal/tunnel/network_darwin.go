//go:build darwin

package tunnel

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"os/exec"
	"regexp"
	"strings"
)

var serviceLineRe = regexp.MustCompile(`^\(\d+\)\s+(.+)$`)
var darwinRunCommand = runCmd

func configureTunInterface(ifaceName, ip, subnetMask string) error {
	mask, err := cidrMaskFromPrefix(subnetMask)
	if err != nil {
		return err
	}

	_, err = darwinRunCommand("ifconfig", ifaceName, "inet", ip, ip, "netmask", mask, "up")
	if err != nil {
		return fmt.Errorf("failed to configure TUN interface %s: %w", ifaceName, err)
	}
	return nil
}

func configureClientNetwork(ifaceName, providerHost string) (func(), error) {
	defaultGW, defaultIface, err := getDefaultRouteInfo()
	if err != nil {
		return nil, err
	}

	if err := setupRouting(ifaceName, providerHost, defaultGW); err != nil {
		return nil, err
	}

	restoreDNS := func() {}
	var dnsService string
	var dnsServers []string
	var hadCustomDNS bool
	if fn, err := setupDNS(defaultIface); err != nil {
		log.Printf("Warning: failed to set DNS automatically on macOS: %v", err)
	} else {
		restoreDNS = fn
		svc, servers, hadCustom, getErr := snapshotDNSState(defaultIface)
		if getErr == nil {
			dnsService = svc
			dnsServers = servers
			hadCustomDNS = hadCustom
		}
	}
	_ = writeCleanupMarker(networkCleanupMarker{
		IfaceName:     ifaceName,
		ProviderHost:  providerHost,
		DefaultGW:     defaultGW,
		DNSConfigured: dnsService != "",
		DNSService:    dnsService,
		DNSServers:    dnsServers,
	})

	return func() {
		restoreRouting(ifaceName, providerHost, defaultGW)
		restoreDNS()
		_ = hadCustomDNS
		clearCleanupMarker()
	}, nil
}

func setupRouting(ifaceName, providerHost, defaultGW string) error {
	if providerHost == "" {
		return fmt.Errorf("provider host is empty")
	}

	_ = routeCmd("-n", "add", "-host", providerHost, defaultGW)
	if err := routeCmd("-n", "add", "-net", "0.0.0.0/1", "-interface", ifaceName); err != nil {
		return fmt.Errorf("failed to add route 0.0.0.0/1 via %s: %w", ifaceName, err)
	}
	if err := routeCmd("-n", "add", "-net", "128.0.0.0/1", "-interface", ifaceName); err != nil {
		_ = routeCmd("-n", "delete", "-net", "0.0.0.0/1", "-interface", ifaceName)
		return fmt.Errorf("failed to add route 128.0.0.0/1 via %s: %w", ifaceName, err)
	}
	return nil
}

func restoreRouting(ifaceName, providerHost, defaultGW string) {
	if strings.TrimSpace(defaultGW) != "" {
		_ = routeCmd("-n", "delete", "-host", providerHost, defaultGW)
	} else {
		_ = routeCmd("-n", "delete", "-host", providerHost)
	}
	_ = routeCmd("-n", "delete", "-net", "0.0.0.0/1", "-interface", ifaceName)
	_ = routeCmd("-n", "delete", "-net", "128.0.0.0/1", "-interface", ifaceName)
}

func setupDNS(defaultIface string) (func(), error) {
	service, err := findNetworkServiceForDevice(defaultIface)
	if err != nil {
		return nil, err
	}

	servers, hadCustom, err := getCurrentDNSServers(service)
	if err != nil {
		return nil, err
	}

	if _, err := darwinRunCommand("networksetup", "-setdnsservers", service, "1.1.1.1", "8.8.8.8"); err != nil {
		return nil, fmt.Errorf("failed to set DNS servers for %s: %w", service, err)
	}

	restore := func() {
		if hadCustom && len(servers) > 0 {
			args := append([]string{"-setdnsservers", service}, servers...)
			if _, err := darwinRunCommand("networksetup", args...); err != nil {
				log.Printf("Warning: failed to restore DNS for %s: %v", service, err)
			}
			return
		}
		if _, err := darwinRunCommand("networksetup", "-setdnsservers", service, "Empty"); err != nil {
			log.Printf("Warning: failed to clear DNS for %s: %v", service, err)
		}
	}

	return restore, nil
}

func snapshotDNSState(defaultIface string) (service string, servers []string, hadCustom bool, err error) {
	service, err = findNetworkServiceForDevice(defaultIface)
	if err != nil {
		return "", nil, false, err
	}
	servers, hadCustom, err = getCurrentDNSServers(service)
	if err != nil {
		return "", nil, false, err
	}
	return service, servers, hadCustom, nil
}

func restoreDNSForService(service string, servers []string, hadCustom bool) {
	if strings.TrimSpace(service) == "" {
		return
	}
	if hadCustom && len(servers) > 0 {
		args := append([]string{"-setdnsservers", service}, servers...)
		_, _ = darwinRunCommand("networksetup", args...)
		return
	}
	_, _ = darwinRunCommand("networksetup", "-setdnsservers", service, "Empty")
}

func getCurrentDNSServers(service string) ([]string, bool, error) {
	out, err := darwinRunCommand("networksetup", "-getdnsservers", service)
	if err != nil {
		return nil, false, err
	}

	lines := splitNonEmptyLines(out)
	if len(lines) == 0 {
		return nil, false, nil
	}
	if strings.Contains(lines[0], "There aren't any DNS Servers set on") {
		return nil, false, nil
	}
	return lines, true, nil
}

func findNetworkServiceForDevice(device string) (string, error) {
	out, err := darwinRunCommand("networksetup", "-listnetworkserviceorder")
	if err != nil {
		return "", fmt.Errorf("failed to list network services: %w", err)
	}

	var currentService string
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if m := serviceLineRe.FindStringSubmatch(line); len(m) == 2 {
			currentService = m[1]
			continue
		}
		if strings.Contains(line, "Device: "+device+")") && currentService != "" {
			return currentService, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("could not find macOS network service for device %s", device)
}

func getDefaultRouteInfo() (gateway string, iface string, err error) {
	out, err := darwinRunCommand("route", "-n", "get", "default")
	if err != nil {
		return "", "", fmt.Errorf("failed to get default route: %w", err)
	}

	for _, line := range splitNonEmptyLines(out) {
		if strings.HasPrefix(line, "gateway:") {
			gateway = strings.TrimSpace(strings.TrimPrefix(line, "gateway:"))
		}
		if strings.HasPrefix(line, "interface:") {
			iface = strings.TrimSpace(strings.TrimPrefix(line, "interface:"))
		}
	}
	if net.ParseIP(gateway) == nil {
		return "", "", fmt.Errorf("could not parse default gateway from route output")
	}
	if iface == "" {
		return "", "", fmt.Errorf("could not parse default interface from route output")
	}
	return gateway, iface, nil
}

func cidrMaskFromPrefix(prefix string) (string, error) {
	var bits int
	if _, err := fmt.Sscanf(prefix, "%d", &bits); err != nil {
		return "", fmt.Errorf("invalid subnet prefix %q: %w", prefix, err)
	}
	if bits < 0 || bits > 32 {
		return "", fmt.Errorf("invalid IPv4 prefix length: %d", bits)
	}
	mask := net.CIDRMask(bits, 32)
	if len(mask) != 4 {
		return "", fmt.Errorf("invalid IPv4 CIDR mask for prefix %d", bits)
	}
	return net.IP(mask).String(), nil
}

func routeCmd(args ...string) error {
	_, err := darwinRunCommand("route", args...)
	if err != nil {
		// Route add/delete might fail for already-present/absent entries; treat as non-fatal.
		if strings.Contains(err.Error(), "File exists") || strings.Contains(err.Error(), "not in table") {
			return nil
		}
	}
	return err
}

func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		return "", fmt.Errorf("%s %v failed: %w (%s)", name, args, err, msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func splitNonEmptyLines(s string) []string {
	raw := strings.Split(s, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		t := strings.TrimSpace(line)
		if t != "" {
			lines = append(lines, t)
		}
	}
	return lines
}
