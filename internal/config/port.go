package config

import (
	"fmt"
	"net"
	"strconv"
)

const (
	UnprivilegedPortStart = 1024
	MaxPort               = 65535
)

func FindAvailablePort(startPort int, maxAttempts int) (int, error) {
	if startPort < 1 {
		startPort = UnprivilegedPortStart
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		port := startPort + attempt
		if port > MaxPort {
			port = UnprivilegedPortStart + (port-UnprivilegedPortStart)%(MaxPort-UnprivilegedPortStart+1)
		}

		ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
		if err != nil {
			continue
		}
		ln.Close()

		udpLn, err := net.ListenUDP("udp", &net.UDPAddr{Port: port})
		if err != nil {
			continue
		}
		udpLn.Close()

		return port, nil
	}

	return 0, fmt.Errorf("no available port found after %d attempts starting from %d", maxAttempts, startPort)
}

func CheckPortAvailable(port int) (bool, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return false, nil
	}
	defer ln.Close()

	udpLn, err := net.ListenUDP("udp", &net.UDPAddr{Port: port})
	if err != nil {
		return false, nil
	}
	defer udpLn.Close()

	return true, nil
}

func DetectPortConflict(providerPort int, clientPort int, providerMetricsPort int, clientMetricsPort int) []string {
	var conflicts []string

	ports := map[string]int{
		"provider.listen_port":         providerPort,
		"client metrics (if same)":     clientPort,
		"provider.metrics_listen_addr": providerMetricsPort,
		"client.metrics_listen_addr":   clientMetricsPort,
	}

	seen := make(map[int]string)
	for name, port := range ports {
		if port <= 0 {
			continue
		}
		if existing, ok := seen[port]; ok {
			conflicts = append(conflicts, fmt.Sprintf("Port %d used by both %s and %s", port, existing, name))
		}
		seen[port] = name
	}

	return conflicts
}

func ParsePortFromAddr(addr string) int {
	if addr == "" {
		return 0
	}
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return 0
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0
	}
	return port
}
