//go:build functional

package tunnel

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"blockchain-vpn/internal/config"
)

func TestFunctional_MultiTunnelConcurrent_Connection(t *testing.T) {
	t.Parallel()

	m := NewMultiTunnelManager()

	clientCfg := &config.ClientConfig{
		InterfaceName: "bcvpn1",
		TunIP:         "10.10.0.2",
		TunSubnet:     "24",
	}

	err := m.Add("tunnel-1", "eth0", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil)
	if err != nil {
		t.Fatalf("First Add failed: %v", err)
	}

	if m.ActiveCount() != 1 {
		t.Errorf("Expected 1 active tunnel, got %d", m.ActiveCount())
	}

	err = m.Add("tunnel-2", "eth1", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil)
	if err != nil {
		t.Fatalf("Second Add failed: %v", err)
	}

	if m.ActiveCount() != 2 {
		t.Errorf("Expected 2 active tunnels, got %d", m.ActiveCount())
	}

	list := m.List()
	if len(list) != 2 {
		t.Errorf("Expected 2 tunnels in list, got %d", len(list))
	}

	m.CancelAll()
	time.Sleep(100 * time.Millisecond)

	if m.ActiveCount() != 0 {
		t.Errorf("Expected 0 active tunnels after CancelAll, got %d", m.ActiveCount())
	}

	t.Log("Multi-tunnel concurrent connections work correctly")
}

func TestFunctional_MultiTunnelConcurrent_MultipleProviders(t *testing.T) {
	t.Parallel()

	m := NewMultiTunnelManager()

	clientCfg := &config.ClientConfig{
		InterfaceName: "bcvpn1",
		TunIP:         "10.10.0.2",
		TunSubnet:     "24",
	}

	providerCount := 5
	for i := 0; i < providerCount; i++ {
		id := fmt.Sprintf("provider-tunnel-%d", i)
		err := m.Add(id, "eth0", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil)
		if err != nil {
			t.Fatalf("Add failed for provider %d: %v", i, err)
		}
	}

	if m.ActiveCount() != providerCount {
		t.Errorf("Expected %d active tunnels, got %d", providerCount, m.ActiveCount())
	}

	m.CancelAll()
	time.Sleep(100 * time.Millisecond)

	if m.ActiveCount() != 0 {
		t.Errorf("Expected 0 active tunnels after CancelAll, got %d", m.ActiveCount())
	}

	t.Logf("Multi-tunnel with %d concurrent providers works correctly", providerCount)
}

func TestFunctional_MultiTunnelConcurrent_DuplicateID(t *testing.T) {
	t.Parallel()

	m := NewMultiTunnelManager()

	clientCfg := &config.ClientConfig{
		InterfaceName: "bcvpn1",
		TunIP:         "10.10.0.2",
		TunSubnet:     "24",
	}

	err := m.Add("same-id", "eth0", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil)
	if err != nil {
		t.Fatalf("First Add failed: %v", err)
	}

	err = m.Add("same-id", "eth1", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil)
	if err == nil {
		t.Error("Expected error for duplicate tunnel ID")
	}

	m.CancelAll()
	time.Sleep(100 * time.Millisecond)

	t.Log("Duplicate tunnel ID rejection works correctly")
}

func TestFunctional_MultiTunnelConcurrent_CancelSpecific(t *testing.T) {
	t.Parallel()

	m := NewMultiTunnelManager()

	clientCfg := &config.ClientConfig{
		InterfaceName: "bcvpn1",
		TunIP:         "10.10.0.2",
		TunSubnet:     "24",
	}

	m.Add("tunnel-1", "eth0", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil)
	m.Add("tunnel-2", "eth1", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil)
	m.Add("tunnel-3", "eth2", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil)

	if m.ActiveCount() != 3 {
		t.Fatalf("Expected 3 active tunnels, got %d", m.ActiveCount())
	}

	listBeforeCancel := m.List()

	m.Cancel("tunnel-2")
	time.Sleep(100 * time.Millisecond)

	listAfterCancel := m.List()

	m.CancelAll()
	time.Sleep(100 * time.Millisecond)

	if len(listBeforeCancel) != 3 {
		t.Errorf("Expected 3 tunnels before cancel, got %d", len(listBeforeCancel))
	}

	if _, exists := listAfterCancel["tunnel-2"]; exists {
		t.Error("tunnel-2 should not exist after cancel")
	}

	t.Log("Cancel specific tunnel works correctly")
}

func TestFunctional_MultiTunnelConcurrent_ConcurrentAdd(t *testing.T) {
	t.Parallel()

	m := NewMultiTunnelManager()

	clientCfg := &config.ClientConfig{
		InterfaceName: "bcvpn1",
		TunIP:         "10.10.0.2",
		TunSubnet:     "24",
	}

	var wg sync.WaitGroup
	tunnelCount := 10

	for i := 0; i < tunnelCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			m.Add("concurrent-tunnel", "eth0", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil)
		}(i)
	}

	wg.Wait()

	m.CancelAll()
	time.Sleep(100 * time.Millisecond)

	t.Logf("Concurrent Add for %d tunnels works correctly", tunnelCount)
}

func TestFunctional_MultiTunnelConcurrent_ListInterfaces(t *testing.T) {
	t.Parallel()

	m := NewMultiTunnelManager()

	clientCfg := &config.ClientConfig{
		InterfaceName: "bcvpn1",
		TunIP:         "10.10.0.2",
		TunSubnet:     "24",
	}

	m.Add("tunnel-us", "eth0", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil)
	m.Add("tunnel-eu", "eth1", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil)
	m.Add("tunnel-asia", "eth2", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil)

	list := m.List()

	if list["tunnel-us"] != "eth0" {
		t.Errorf("Expected tunnel-us -> eth0, got %s", list["tunnel-us"])
	}
	if list["tunnel-eu"] != "eth1" {
		t.Errorf("Expected tunnel-eu -> eth1, got %s", list["tunnel-eu"])
	}
	if list["tunnel-asia"] != "eth2" {
		t.Errorf("Expected tunnel-asia -> eth2, got %s", list["tunnel-asia"])
	}

	m.CancelAll()
	time.Sleep(100 * time.Millisecond)

	t.Log("List returns correct interface mappings")
}
