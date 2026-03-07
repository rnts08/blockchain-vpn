//go:build !linux && !darwin && !windows

package tunnel

import "fmt"

func readConfiguredDNSServers() ([]string, error) {
	return nil, fmt.Errorf("DNS introspection not supported on this platform")
}
