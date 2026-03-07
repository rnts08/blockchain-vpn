package tunnel

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type runtimeMetricsState struct {
	providerRunning atomic.Bool
	clientConnected atomic.Bool
	activeSessions  atomic.Int64
	totalUpBytes    atomic.Int64
	totalDownBytes  atomic.Int64
	errorCount      atomic.Int64

	healthMu        sync.RWMutex
	lastTunOK       bool
	lastTunCheck    time.Time
	lastListenOK    bool
	lastListenCheck time.Time
	lastError       string
}

var runtimeMetrics = &runtimeMetricsState{}
var metricsServers sync.Map // key=addr

func startMetricsServer(addr, token string) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return
	}
	key := addr + "|" + token
	if _, loaded := metricsServers.LoadOrStore(key, struct{}{}); loaded {
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics.json", func(w http.ResponseWriter, r *http.Request) {
		if token != "" {
			candidate := strings.TrimSpace(r.Header.Get("X-BCVPN-Metrics-Token"))
			if candidate == "" {
				candidate = strings.TrimSpace(r.URL.Query().Get("token"))
			}
			if candidate != token {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		runtimeMetrics.healthMu.RLock()
		payload := map[string]any{
			"provider_running": runtimeMetrics.providerRunning.Load(),
			"client_connected": runtimeMetrics.clientConnected.Load(),
			"active_sessions":  runtimeMetrics.activeSessions.Load(),
			"total_up_bytes":   runtimeMetrics.totalUpBytes.Load(),
			"total_down_bytes": runtimeMetrics.totalDownBytes.Load(),
			"error_count":      runtimeMetrics.errorCount.Load(),
			"health": map[string]any{
				"tun_ok":              runtimeMetrics.lastTunOK,
				"tun_checked_at":      runtimeMetrics.lastTunCheck.UTC().Format(time.RFC3339),
				"listener_ok":         runtimeMetrics.lastListenOK,
				"listener_checked_at": runtimeMetrics.lastListenCheck.UTC().Format(time.RFC3339),
			},
			"last_error": runtimeMetrics.lastError,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		}
		runtimeMetrics.healthMu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	})

	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("Metrics server failed on %s: %v", addr, err)
		}
	}()
	log.Printf("Metrics endpoint enabled at http://%s/metrics.json", addr)
}

func recordRuntimeError(err error) {
	if err == nil {
		return
	}
	runtimeMetrics.errorCount.Add(1)
	runtimeMetrics.healthMu.Lock()
	runtimeMetrics.lastError = err.Error()
	runtimeMetrics.healthMu.Unlock()
}

func recordTraffic(upBytes, downBytes int64) {
	if upBytes > 0 {
		runtimeMetrics.totalUpBytes.Add(upBytes)
	}
	if downBytes > 0 {
		runtimeMetrics.totalDownBytes.Add(downBytes)
	}
}

func setProviderRunning(v bool) {
	runtimeMetrics.providerRunning.Store(v)
}

func setClientConnected(v bool) {
	runtimeMetrics.clientConnected.Store(v)
}

func sessionOpened() {
	runtimeMetrics.activeSessions.Add(1)
}

func sessionClosed() {
	runtimeMetrics.activeSessions.Add(-1)
}

func recordHealthStatus(tunOK, listenerOK bool) {
	runtimeMetrics.healthMu.Lock()
	runtimeMetrics.lastTunOK = tunOK
	runtimeMetrics.lastListenOK = listenerOK
	runtimeMetrics.lastTunCheck = time.Now()
	runtimeMetrics.lastListenCheck = time.Now()
	runtimeMetrics.healthMu.Unlock()
}

func GetRuntimeMetricsSnapshot() map[string]any {
	runtimeMetrics.healthMu.RLock()
	defer runtimeMetrics.healthMu.RUnlock()
	return map[string]any{
		"provider_running": runtimeMetrics.providerRunning.Load(),
		"client_connected": runtimeMetrics.clientConnected.Load(),
		"active_sessions":  runtimeMetrics.activeSessions.Load(),
		"total_up_bytes":   runtimeMetrics.totalUpBytes.Load(),
		"total_down_bytes": runtimeMetrics.totalDownBytes.Load(),
		"error_count":      runtimeMetrics.errorCount.Load(),
		"health": map[string]any{
			"tun_ok":              runtimeMetrics.lastTunOK,
			"tun_checked_at":      runtimeMetrics.lastTunCheck.UTC().Format(time.RFC3339),
			"listener_ok":         runtimeMetrics.lastListenOK,
			"listener_checked_at": runtimeMetrics.lastListenCheck.UTC().Format(time.RFC3339),
		},
		"last_error": runtimeMetrics.lastError,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}
}
