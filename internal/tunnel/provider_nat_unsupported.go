//go:build !linux

package tunnel

import "fmt"

func setupProviderEgressNAT(tunIfName, tunIP, tunSubnet, outboundIf string) (func(), error) {
	_ = tunIfName
	_ = tunIP
	_ = tunSubnet
	_ = outboundIf
	return nil, fmt.Errorf("provider egress NAT backend is not implemented for this platform")
}
