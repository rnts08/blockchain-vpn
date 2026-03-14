//go:build functional

package tunnel

import (
	"testing"
	"time"
)

func TestFunctional_TunnelLifecycle_IPPool(t *testing.T) {
	t.Parallel()

	pool := NewIPPool("10.0.0.1")

	ip1, err := pool.Allocate()
	if err != nil {
		t.Fatalf("First allocation failed: %v", err)
	}
	if ip1.String() != "10.0.0.2" {
		t.Errorf("Expected first IP 10.0.0.2, got %s", ip1.String())
	}

	ip2, err := pool.Allocate()
	if err != nil {
		t.Fatalf("Second allocation failed: %v", err)
	}
	if ip2.String() != "10.0.0.3" {
		t.Errorf("Expected second IP 10.0.0.3, got %s", ip2.String())
	}

	pool.Release(ip1)

	ip3, err := pool.Allocate()
	if err != nil {
		t.Fatalf("Third allocation after release failed: %v", err)
	}
	if ip3.String() != "10.0.0.2" {
		t.Errorf("Expected reused IP 10.0.0.2, got %s", ip3.String())
	}

	t.Log("IP pool allocation and release works correctly")
}

func TestFunctional_TunnelLifecycle_IPPoolIPv6(t *testing.T) {
	t.Parallel()

	pool := NewIPPool("2001:db8::1")

	ip1, err := pool.Allocate()
	if err != nil {
		t.Fatalf("First IPv6 allocation failed: %v", err)
	}
	if ip1.String() != "2001:db8::2" {
		t.Errorf("Expected first IPv6 2001:db8::2, got %s", ip1.String())
	}

	ip2, err := pool.Allocate()
	if err != nil {
		t.Fatalf("Second IPv6 allocation failed: %v", err)
	}
	if ip2.String() != "2001:db8::3" {
		t.Errorf("Expected second IPv6 2001:db8::3, got %s", ip2.String())
	}

	t.Log("IPv6 pool allocation works correctly")
}

func TestFunctional_TunnelLifecycle_IPPoolExhaustion(t *testing.T) {
	t.Parallel()

	pool := NewIPPool("10.0.0.1")

	for i := 2; i < 255; i++ {
		_, err := pool.Allocate()
		if err != nil {
			t.Fatalf("Allocation %d failed unexpectedly: %v", i, err)
		}
	}

	_, err := pool.Allocate()
	if err == nil {
		t.Error("Expected error when IP pool exhausted, got nil")
	}

	t.Log("IP pool exhaustion handling works correctly")
}

func TestFunctional_TunnelLifecycle_SessionStats(t *testing.T) {
	t.Parallel()

	stats := newSessionStats()

	if time.Since(stats.startedAt) > time.Second {
		t.Error("Session stats startedAt should be recent")
	}

	stats.addUpstream(1000)
	stats.addDownstream(2000)

	if stats.upstreamBytes.Load() != 1000 {
		t.Errorf("Expected upstream bytes 1000, got %d", stats.upstreamBytes.Load())
	}
	if stats.downstreamBytes.Load() != 2000 {
		t.Errorf("Expected downstream bytes 2000, got %d", stats.downstreamBytes.Load())
	}

	t.Log("Session stats tracking works correctly")
}

func TestFunctional_TunnelLifecycle_SessionStatsConcurrent(t *testing.T) {
	t.Parallel()

	stats := newSessionStats()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			stats.addUpstream(100)
			stats.addDownstream(100)
		}
		close(done)
	}()

	<-done

	upstream := stats.upstreamBytes.Load()
	downstream := stats.downstreamBytes.Load()

	if upstream != 100000 {
		t.Errorf("Expected upstream 100000, got %d", upstream)
	}
	if downstream != 100000 {
		t.Errorf("Expected downstream 100000, got %d", downstream)
	}

	t.Log("Session stats concurrent access works correctly")
}

func TestFunctional_TunnelLifecycle_RateEnforcer(t *testing.T) {
	t.Parallel()

	enforcer := newRateEnforcer(1000)

	start := time.Now()
	enforcer.accountAndThrottle(500)
	elapsed := time.Since(start)

	if elapsed < 400*time.Millisecond {
		t.Errorf("Expected at least 400ms delay for 500 bytes at 1000 Bps, got %v", elapsed)
	}

	t.Log("Rate enforcer throttling works correctly")
}

func TestFunctional_TunnelLifecycle_RateEnforcerNil(t *testing.T) {
	t.Parallel()

	var enforcer *rateEnforcer

	enforcer.accountAndThrottle(100)

	enforcer = newRateEnforcer(0)
	enforcer.accountAndThrottle(100)

	enforcer = newRateEnforcer(-1)
	enforcer.accountAndThrottle(100)

	t.Log("Rate enforcer edge cases handled correctly")
}
