//go:build linux

package tunnel

import (
	"bufio"
	"os"
	"strings"
)

func readConfiguredDNSServers() ([]string, error) {
	f, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var servers []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "nameserver ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				servers = append(servers, fields[1])
			}
		}
	}
	return servers, scanner.Err()
}
