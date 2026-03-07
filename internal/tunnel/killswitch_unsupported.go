//go:build !linux && !darwin && !windows

package tunnel

import "fmt"

func setupKillSwitch(tunIfName, providerHost string) (func(), error) {
	_ = tunIfName
	_ = providerHost
	return nil, fmt.Errorf("kill switch is not supported on this platform")
}
