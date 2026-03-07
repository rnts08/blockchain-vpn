//go:build windows

package tunnel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
)

type windowsDefaultRoute struct {
	InterfaceAlias string
	InterfaceIndex int
	NextHop        string
}

var (
	windowsRunCommand    = runWindowsCmd
	windowsRunPowerShell = runPowerShell
)

func configureTunInterface(ifaceName, ip, subnetMask string) error {
	mask, err := cidrMaskFromPrefix(subnetMask)
	if err != nil {
		return err
	}
	_, err = windowsRunCommand("netsh", "interface", "ipv4", "set", "address", fmt.Sprintf("name=%s", ifaceName), "static", ip, mask)
	if err != nil {
		return fmt.Errorf("failed to configure TUN interface %s: %w", ifaceName, err)
	}
	return nil
}

func configureClientNetwork(ifaceName, providerHost string) (func(), error) {
	defaultRoute, err := getWindowsDefaultRoute()
	if err != nil {
		return nil, err
	}
	tunIfIndex, err := getWindowsInterfaceIndex(ifaceName)
	if err != nil {
		return nil, err
	}

	if err := setupWindowsRouting(tunIfIndex, providerHost, defaultRoute.NextHop, defaultRoute.InterfaceIndex); err != nil {
		return nil, err
	}

	restoreDNS, err := setupWindowsDNS(defaultRoute.InterfaceAlias)
	if err != nil {
		restoreWindowsRouting(tunIfIndex, providerHost)
		return nil, err
	}

	return func() {
		restoreWindowsRouting(tunIfIndex, providerHost)
		restoreDNS()
	}, nil
}

func setupWindowsRouting(tunIfIndex int, providerHost, defaultGW string, defaultIfIndex int) error {
	if providerHost == "" {
		return fmt.Errorf("provider host is empty")
	}

	// Keep provider control channel outside tunnel.
	_, _ = windowsRunCommand("route", "ADD", providerHost, "MASK", "255.255.255.255", defaultGW, "METRIC", "1", "IF", strconv.Itoa(defaultIfIndex))

	// Full tunnel split default route.
	if _, err := windowsRunCommand("route", "ADD", "0.0.0.0", "MASK", "128.0.0.0", "0.0.0.0", "IF", strconv.Itoa(tunIfIndex)); err != nil {
		return fmt.Errorf("failed to add route 0.0.0.0/1 via interface index %d: %w", tunIfIndex, err)
	}
	if _, err := windowsRunCommand("route", "ADD", "128.0.0.0", "MASK", "128.0.0.0", "0.0.0.0", "IF", strconv.Itoa(tunIfIndex)); err != nil {
		_, _ = windowsRunCommand("route", "DELETE", "0.0.0.0", "MASK", "128.0.0.0", "0.0.0.0", "IF", strconv.Itoa(tunIfIndex))
		return fmt.Errorf("failed to add route 128.0.0.0/1 via interface index %d: %w", tunIfIndex, err)
	}
	return nil
}

func restoreWindowsRouting(tunIfIndex int, providerHost string) {
	_, _ = windowsRunCommand("route", "DELETE", providerHost)
	_, _ = windowsRunCommand("route", "DELETE", "0.0.0.0", "MASK", "128.0.0.0", "0.0.0.0", "IF", strconv.Itoa(tunIfIndex))
	_, _ = windowsRunCommand("route", "DELETE", "128.0.0.0", "MASK", "128.0.0.0", "0.0.0.0", "IF", strconv.Itoa(tunIfIndex))
}

func setupWindowsDNS(ifaceAlias string) (func(), error) {
	servers, err := getWindowsDNSServers(ifaceAlias)
	if err != nil {
		return nil, err
	}

	if _, err := windowsRunPowerShell(fmt.Sprintf(`Set-DnsClientServerAddress -InterfaceAlias '%s' -ServerAddresses @('1.1.1.1','8.8.8.8')`, psEscape(ifaceAlias))); err != nil {
		return nil, fmt.Errorf("failed to set DNS servers for %s: %w", ifaceAlias, err)
	}

	return func() {
		if len(servers) == 0 {
			_, _ = windowsRunPowerShell(fmt.Sprintf(`Set-DnsClientServerAddress -InterfaceAlias '%s' -ResetServerAddresses`, psEscape(ifaceAlias)))
			return
		}
		var quoted []string
		for _, s := range servers {
			quoted = append(quoted, fmt.Sprintf("'%s'", psEscape(s)))
		}
		_, _ = windowsRunPowerShell(fmt.Sprintf(`Set-DnsClientServerAddress -InterfaceAlias '%s' -ServerAddresses @(%s)`, psEscape(ifaceAlias), strings.Join(quoted, ",")))
	}, nil
}

func getWindowsDNSServers(ifaceAlias string) ([]string, error) {
	out, err := windowsRunPowerShell(fmt.Sprintf(`$x = Get-DnsClientServerAddress -InterfaceAlias '%s' -AddressFamily IPv4; if ($x -eq $null -or $x.ServerAddresses.Count -eq 0) { '[]' } else { $x.ServerAddresses | ConvertTo-Json -Compress }`, psEscape(ifaceAlias)))
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS settings: %w", err)
	}
	out = strings.TrimSpace(out)
	if out == "" || out == "null" || out == "[]" {
		return nil, nil
	}

	// ConvertTo-Json returns string literal when only one entry.
	if strings.HasPrefix(out, "\"") {
		var one string
		if err := json.Unmarshal([]byte(out), &one); err != nil {
			return nil, err
		}
		return []string{one}, nil
	}

	var servers []string
	if err := json.Unmarshal([]byte(out), &servers); err != nil {
		return nil, err
	}
	return servers, nil
}

func getWindowsInterfaceIndex(ifaceAlias string) (int, error) {
	out, err := windowsRunPowerShell(fmt.Sprintf(`(Get-NetIPInterface -AddressFamily IPv4 -InterfaceAlias '%s' | Select-Object -First 1 -ExpandProperty InterfaceIndex)`, psEscape(ifaceAlias)))
	if err != nil {
		return 0, fmt.Errorf("failed to get interface index for %s: %w", ifaceAlias, err)
	}
	idx, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return 0, fmt.Errorf("invalid interface index %q for %s", out, ifaceAlias)
	}
	return idx, nil
}

func getWindowsDefaultRoute() (*windowsDefaultRoute, error) {
	out, err := windowsRunPowerShell(`$r = Get-NetRoute -AddressFamily IPv4 -DestinationPrefix '0.0.0.0/0' | Sort-Object RouteMetric,InterfaceMetric | Select-Object -First 1 InterfaceAlias,InterfaceIndex,NextHop; $r | ConvertTo-Json -Compress`)
	if err != nil {
		return nil, fmt.Errorf("failed to query default route: %w", err)
	}

	var r windowsDefaultRoute
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &r); err != nil {
		return nil, fmt.Errorf("failed to parse default route JSON: %w", err)
	}
	if net.ParseIP(r.NextHop) == nil {
		return nil, fmt.Errorf("invalid default gateway: %s", r.NextHop)
	}
	if r.InterfaceAlias == "" || r.InterfaceIndex == 0 {
		return nil, fmt.Errorf("could not determine default route interface")
	}
	return &r, nil
}

func cidrMaskFromPrefix(prefix string) (string, error) {
	bits, err := strconv.Atoi(strings.TrimSpace(prefix))
	if err != nil {
		return "", fmt.Errorf("invalid subnet prefix %q: %w", prefix, err)
	}
	if bits < 0 || bits > 32 {
		return "", fmt.Errorf("invalid IPv4 prefix length: %d", bits)
	}
	mask := net.CIDRMask(bits, 32)
	return net.IP(mask).String(), nil
}

func psEscape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func runPowerShell(cmd string) (string, error) {
	return runWindowsCmd("powershell", "-NoProfile", "-NonInteractive", "-Command", cmd)
}

func runWindowsCmd(name string, args ...string) (string, error) {
	c := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		return "", fmt.Errorf("%s %v failed: %w (%s)", name, args, err, msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}
