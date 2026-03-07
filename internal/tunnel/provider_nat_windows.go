//go:build windows

package tunnel

import (
	"fmt"
	"strings"
)

func setupProviderEgressNAT(tunIfName, tunIP, tunSubnet, outboundIf string) (func(), error) {
	if strings.TrimSpace(tunIfName) == "" {
		return nil, fmt.Errorf("provider TUN interface name is required")
	}

	if strings.TrimSpace(outboundIf) == "" {
		def, err := getWindowsDefaultRoute()
		if err != nil {
			return nil, err
		}
		outboundIf = def.InterfaceAlias
	}

	cidr, err := cidrFromIPPrefix(tunIP, tunSubnet)
	if err != nil {
		return nil, err
	}

	natName := "BlockchainVPN-" + sanitizeWindowsNatName(tunIfName)

	// Best-effort cleanup of previous stale state.
	_, _ = runPowerShell(fmt.Sprintf(`if (Get-NetNat -Name '%s' -ErrorAction SilentlyContinue) { Remove-NetNat -Name '%s' -Confirm:$false }`, psEscape(natName), psEscape(natName)))

	_, err = runPowerShell(fmt.Sprintf(`Set-NetIPInterface -InterfaceAlias '%s' -Forwarding Enabled -ErrorAction Stop`, psEscape(tunIfName)))
	if err != nil {
		return nil, fmt.Errorf("failed to enable forwarding on provider TUN interface: %w", err)
	}
	_, err = runPowerShell(fmt.Sprintf(`Set-NetIPInterface -InterfaceAlias '%s' -Forwarding Enabled -ErrorAction Stop`, psEscape(outboundIf)))
	if err != nil {
		return nil, fmt.Errorf("failed to enable forwarding on outbound interface %s: %w", outboundIf, err)
	}

	_, err = runPowerShell(fmt.Sprintf(`New-NetNat -Name '%s' -InternalIPInterfaceAddressPrefix '%s' -ErrorAction Stop`, psEscape(natName), psEscape(cidr)))
	if err != nil {
		return nil, fmt.Errorf("failed to create NetNat %s for %s: %w", natName, cidr, err)
	}

	cleanup := func() {
		_, _ = runPowerShell(fmt.Sprintf(`if (Get-NetNat -Name '%s' -ErrorAction SilentlyContinue) { Remove-NetNat -Name '%s' -Confirm:$false }`, psEscape(natName), psEscape(natName)))
	}
	return cleanup, nil
}

func sanitizeWindowsNatName(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "provider"
	}
	var b strings.Builder
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return b.String()
}
