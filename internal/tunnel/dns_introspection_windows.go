//go:build windows

package tunnel

import (
	"encoding/json"
	"strings"
)

func readConfiguredDNSServers() ([]string, error) {
	out, err := windowsRunPowerShell(`$x = Get-DnsClientServerAddress -AddressFamily IPv4 | Where-Object { $_.ServerAddresses -and $_.ServerAddresses.Count -gt 0 } | Select-Object -ExpandProperty ServerAddresses; if ($x -eq $null) { '[]' } else { $x | ConvertTo-Json -Compress }`)
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" || out == "[]" || out == "null" {
		return nil, nil
	}
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
