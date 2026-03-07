//go:build !linux && !darwin && !windows

package tunnel

import "fmt"

func configureTunInterface(ifaceName, ip, subnetMask string) error {
	return fmt.Errorf("automatic TUN configuration is not supported on this platform; configure interface %s manually with %s/%s", ifaceName, ip, subnetMask)
}

func configureClientNetwork(ifaceName, providerHost string) (func(), error) {
	_ = ifaceName
	_ = providerHost
	return nil, fmt.Errorf("automatic route and DNS configuration is not supported on this platform")
}
