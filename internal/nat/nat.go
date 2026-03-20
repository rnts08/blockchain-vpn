package nat

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/huin/goupnp/dcps/internetgateway1"
	"github.com/huin/goupnp/dcps/internetgateway2"
	"github.com/jackpal/gateway"
	natpmp "github.com/jackpal/go-nat-pmp"
)

type PortMappingResult struct {
	ExternalIP net.IP
	TCPPort    int
	UDPPort    int
	Cleanup    func()
}

func DiscoverAndMapPorts(ctx context.Context, internalTCPPort, internalUDPPort int) (*PortMappingResult, error) {
	type mapper func(context.Context, int, int) (*PortMappingResult, error)
	mappers := []mapper{
		mapWithUPnPIGD2,
		mapWithUPnPIGD1,
		mapWithNATPMP,
	}

	natTimeout := 10 * time.Second
	natCtx, cancel := context.WithTimeout(ctx, natTimeout)
	defer cancel()

	var lastErr error
	for _, m := range mappers {
		if err := natCtx.Err(); err != nil {
			return nil, fmt.Errorf("NAT traversal timed out after %v: %w", natTimeout, err)
		}
		res, err := m(natCtx, internalTCPPort, internalUDPPort)
		if err == nil {
			return res, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("failed NAT traversal with UPnP/NAT-PMP: %w", lastErr)
}

func mapWithUPnPIGD2(ctx context.Context, internalTCPPort, internalUDPPort int) (*PortMappingResult, error) {
	clients, _, err := internetgateway2.NewWANIPConnection1Clients()
	if err != nil || len(clients) == 0 {
		return nil, fmt.Errorf("igd2 discovery failed: %w", err)
	}
	c := clients[0]

	internalIP, err := localIPv4ForRemote(c.ServiceClient.Location.Host)
	if err != nil {
		return nil, err
	}
	externalIPStr, err := c.GetExternalIPAddress()
	if err != nil {
		return nil, err
	}
	externalIP := net.ParseIP(externalIPStr)
	if externalIP == nil {
		return nil, fmt.Errorf("invalid external IP returned: %q", externalIPStr)
	}

	if err := c.AddPortMapping("", uint16(internalTCPPort), "TCP", uint16(internalTCPPort), internalIP.String(), true, "BlockchainVPN TCP", 3600); err != nil {
		return nil, err
	}
	if err := c.AddPortMapping("", uint16(internalUDPPort), "UDP", uint16(internalUDPPort), internalIP.String(), true, "BlockchainVPN UDP", 3600); err != nil {
		_ = c.DeletePortMapping("", uint16(internalTCPPort), "TCP")
		return nil, err
	}

	cleanup := func() {
		_ = c.DeletePortMapping("", uint16(internalTCPPort), "TCP")
		_ = c.DeletePortMapping("", uint16(internalUDPPort), "UDP")
	}
	return &PortMappingResult{
		ExternalIP: externalIP,
		TCPPort:    internalTCPPort,
		UDPPort:    internalUDPPort,
		Cleanup:    cleanup,
	}, nil
}

func mapWithUPnPIGD1(ctx context.Context, internalTCPPort, internalUDPPort int) (*PortMappingResult, error) {
	clients, _, err := internetgateway1.NewWANIPConnection1Clients()
	if err != nil || len(clients) == 0 {
		return nil, fmt.Errorf("igd1 discovery failed: %w", err)
	}
	c := clients[0]

	internalIP, err := localIPv4ForRemote(c.ServiceClient.Location.Host)
	if err != nil {
		return nil, err
	}
	externalIPStr, err := c.GetExternalIPAddress()
	if err != nil {
		return nil, err
	}
	externalIP := net.ParseIP(externalIPStr)
	if externalIP == nil {
		return nil, fmt.Errorf("invalid external IP returned: %q", externalIPStr)
	}

	if err := c.AddPortMapping("", uint16(internalTCPPort), "TCP", uint16(internalTCPPort), internalIP.String(), true, "BlockchainVPN TCP", 3600); err != nil {
		return nil, err
	}
	if err := c.AddPortMapping("", uint16(internalUDPPort), "UDP", uint16(internalUDPPort), internalIP.String(), true, "BlockchainVPN UDP", 3600); err != nil {
		_ = c.DeletePortMapping("", uint16(internalTCPPort), "TCP")
		return nil, err
	}

	cleanup := func() {
		_ = c.DeletePortMapping("", uint16(internalTCPPort), "TCP")
		_ = c.DeletePortMapping("", uint16(internalUDPPort), "UDP")
	}
	return &PortMappingResult{
		ExternalIP: externalIP,
		TCPPort:    internalTCPPort,
		UDPPort:    internalUDPPort,
		Cleanup:    cleanup,
	}, nil
}

func mapWithNATPMP(ctx context.Context, internalTCPPort, internalUDPPort int) (*PortMappingResult, error) {
	gw, err := gateway.DiscoverGateway()
	if err != nil {
		return nil, err
	}
	client := natpmp.NewClient(gw)

	type pmpResult struct {
		resp *natpmp.GetExternalAddressResult
		err  error
	}
	ch := make(chan pmpResult, 1)
	go func() {
		resp, err := client.GetExternalAddress()
		select {
		case <-ctx.Done():
			return
		case ch <- pmpResult{resp: resp, err: err}:
		}
	}()
	var extResp *natpmp.GetExternalAddressResult
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case got := <-ch:
		if got.err != nil {
			return nil, got.err
		}
		extResp = got.resp
	}

	if _, err := client.AddPortMapping("tcp", internalTCPPort, internalTCPPort, 3600); err != nil {
		return nil, err
	}
	if _, err := client.AddPortMapping("udp", internalUDPPort, internalUDPPort, 3600); err != nil {
		_, _ = client.AddPortMapping("tcp", internalTCPPort, 0, 0)
		return nil, err
	}

	externalIP := net.IPv4(
		extResp.ExternalIPAddress[0],
		extResp.ExternalIPAddress[1],
		extResp.ExternalIPAddress[2],
		extResp.ExternalIPAddress[3],
	)
	cleanup := func() {
		_, _ = client.AddPortMapping("tcp", internalTCPPort, 0, 0)
		_, _ = client.AddPortMapping("udp", internalUDPPort, 0, 0)
	}

	return &PortMappingResult{
		ExternalIP: externalIP,
		TCPPort:    internalTCPPort,
		UDPPort:    internalUDPPort,
		Cleanup:    cleanup,
	}, nil
}

func localIPv4ForRemote(remoteHost string) (net.IP, error) {
	// UPnP location host includes host:port.
	conn, err := net.DialTimeout("udp", net.JoinHostPort(remoteHost, "1900"), 2*time.Second)
	if err != nil {
		// fallback for host already including port
		conn, err = net.DialTimeout("udp", remoteHost, 2*time.Second)
		if err != nil {
			return nil, err
		}
	}
	defer conn.Close()
	addr := conn.LocalAddr().(*net.UDPAddr)
	if ip := addr.IP.To4(); ip != nil {
		return ip, nil
	}
	return nil, fmt.Errorf("could not determine local IPv4 for remote host %s", remoteHost)
}
