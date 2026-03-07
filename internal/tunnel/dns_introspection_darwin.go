//go:build darwin

package tunnel

import (
	"strings"
)

func readConfiguredDNSServers() ([]string, error) {
	out, err := darwinRunCommand("scutil", "--dns")
	if err != nil {
		return nil, err
	}
	var servers []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "nameserver[") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				servers = append(servers, strings.TrimSpace(parts[1]))
			}
		}
	}
	return servers, nil
}
