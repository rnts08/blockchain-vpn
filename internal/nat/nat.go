package nat

import (
	"context"
	"fmt"
	"net"
)

// PortMappingResult holds the results of a successful port mapping.
type PortMappingResult struct {
	ExternalIP net.IP
	TCPPort    int
	UDPPort    int
	Cleanup    func()
}

// DiscoverAndMapPorts attempts to discover a NAT device and map the required ports.
// This build currently provides a fallback that reports NAT traversal as unavailable.
func DiscoverAndMapPorts(ctx context.Context, internalTCPPort, internalUDPPort int) (*PortMappingResult, error) {
	_ = ctx
	_ = internalTCPPort
	_ = internalUDPPort
	return nil, fmt.Errorf("NAT traversal is currently unavailable: no supported UPnP/NAT-PMP backend configured")
}
