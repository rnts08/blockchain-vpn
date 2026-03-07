package tunnel

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"blockchain-vpn/internal/config"
)

func startProviderHealthChecks(ctx context.Context, cfg *config.ProviderConfig, tunIfName, listenAddr string) {
	if !cfg.HealthCheckEnabled {
		return
	}

	interval := 30 * time.Second
	if cfg.HealthCheckInterval != "" {
		if d, err := time.ParseDuration(cfg.HealthCheckInterval); err == nil && d > 0 {
			interval = d
		}
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runProviderHealthCheck(tunIfName, listenAddr)
			}
		}
	}()
}

func runProviderHealthCheck(tunIfName, listenAddr string) {
	tunErr := checkTunInterfaceUp(tunIfName)
	if tunErr != nil {
		log.Printf("Healthcheck warning: TUN interface unhealthy (%s): %v", tunIfName, tunErr)
	}

	listenerErr := checkListenerBound(listenAddr)
	if listenerErr != nil {
		log.Printf("Healthcheck warning: listener unhealthy (%s): %v", listenAddr, listenerErr)
	}
	recordHealthStatus(tunErr == nil, listenerErr == nil)
}

func checkTunInterfaceUp(name string) error {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return err
	}
	if iface.Flags&net.FlagUp == 0 {
		return fmt.Errorf("interface is down")
	}
	return nil
}

func checkListenerBound(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err == nil {
		_ = ln.Close()
		return fmt.Errorf("port is unexpectedly free")
	}

	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "address already in use") {
		return nil
	}
	return err
}
