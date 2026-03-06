package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// GetPublicIP attempts to determine the public IP address of the provider
// by querying external IP echo services.
func GetPublicIP() (net.IP, error) {
	services := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
		"https://checkip.amazonaws.com",
	}

	for _, service := range services {
		client := http.Client{
			Timeout: 5 * time.Second,
		}
		resp, err := client.Get(service)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		ipStr := strings.TrimSpace(string(body))
		ip := net.ParseIP(ipStr)
		if ip != nil {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("failed to determine public IP from any service")
}

// formatBytes is a helper function to format byte counts into human-readable strings.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}