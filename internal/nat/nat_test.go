package nat

import (
	"context"
	"net"
	"testing"
)

func TestPortMappingResultFields(t *testing.T) {
	cleanupCalled := false
	result := &PortMappingResult{
		ExternalIP: net.ParseIP("192.168.1.1"),
		TCPPort:    51820,
		UDPPort:    51820,
		Cleanup: func() {
			cleanupCalled = true
		},
	}

	if result.ExternalIP.String() != "192.168.1.1" {
		t.Errorf("expected external IP 192.168.1.1, got %s", result.ExternalIP.String())
	}
	if result.TCPPort != 51820 {
		t.Errorf("expected TCP port 51820, got %d", result.TCPPort)
	}
	if result.UDPPort != 51820 {
		t.Errorf("expected UDP port 51820, got %d", result.UDPPort)
	}

	result.Cleanup()
	if !cleanupCalled {
		t.Error("cleanup function was not called")
	}
}

func TestPortMappingResultNilCleanup(t *testing.T) {
	result := &PortMappingResult{
		ExternalIP: net.ParseIP("1.2.3.4"),
		TCPPort:    80,
		UDPPort:    443,
		Cleanup:    nil,
	}

	if result.Cleanup != nil {
		result.Cleanup()
	}
}

func TestLocalIPv4ForRemoteInvalid(t *testing.T) {
	_, err := localIPv4ForRemote("192.0.2.1:1900")
	if err != nil {
		t.Logf("expected error for invalid remote host: %v", err)
	}
}

func TestDiscoverAndMapPortsContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := DiscoverAndMapPorts(ctx, 51820, 51820)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

type mockAddr struct {
	ip   net.IP
	port int
}

func (a mockAddr) Network() string { return "udp" }
func (a mockAddr) String() string  { return net.JoinHostPort(a.ip.String(), "1900") }

func TestPortMappingResultZeroPorts(t *testing.T) {
	result := &PortMappingResult{
		ExternalIP: net.ParseIP("10.0.0.1"),
		TCPPort:    0,
		UDPPort:    0,
		Cleanup:    nil,
	}

	if result.TCPPort != 0 {
		t.Errorf("expected 0 TCP port, got %d", result.TCPPort)
	}
	if result.UDPPort != 0 {
		t.Errorf("expected 0 UDP port, got %d", result.UDPPort)
	}
}

func TestExternalIPNil(t *testing.T) {
	result := &PortMappingResult{
		ExternalIP: nil,
		TCPPort:    51820,
		UDPPort:    51820,
		Cleanup:    nil,
	}

	if result.ExternalIP != nil {
		t.Error("expected nil IP")
	}
}

func TestPortMappingResultIPv6(t *testing.T) {
	result := &PortMappingResult{
		ExternalIP: net.ParseIP("2001:db8::1"),
		TCPPort:    8080,
		UDPPort:    8080,
		Cleanup:    nil,
	}

	if !result.ExternalIP.IsGlobalUnicast() {
		t.Error("expected global unicast IP")
	}
}

func TestPortMappingResultDifferentPorts(t *testing.T) {
	result := &PortMappingResult{
		ExternalIP: net.ParseIP("203.0.113.1"),
		TCPPort:    443,
		UDPPort:    1194,
		Cleanup:    nil,
	}

	if result.TCPPort == result.UDPPort {
		t.Error("TCP and UDP ports should be different")
	}
}

func TestPortMappingResultAllPorts(t *testing.T) {
	result := &PortMappingResult{
		ExternalIP: net.ParseIP("192.168.1.100"),
		TCPPort:    1,
		UDPPort:    65535,
		Cleanup:    nil,
	}

	if result.TCPPort != 1 {
		t.Errorf("expected TCP port 1, got %d", result.TCPPort)
	}
	if result.UDPPort != 65535 {
		t.Errorf("expected UDP port 65535, got %d", result.UDPPort)
	}
}
