package tunnel

import (
	"context"
	"sync"
	"testing"
	"time"

	"blockchain-vpn/internal/config"
)

func TestMultiTunnelManager_New(t *testing.T) {
	m := NewMultiTunnelManager()
	if m == nil {
		t.Fatal("NewMultiTunnelManager returned nil")
	}
	if m.tunnels == nil {
		t.Error("tunnels map is nil")
	}
}

func TestMultiTunnelManager_AddDuplicate(t *testing.T) {
	m := NewMultiTunnelManager()

	clientCfg := &config.ClientConfig{
		InterfaceName: "bcvpn1",
		TunIP:         "10.10.0.2",
		TunSubnet:     "24",
	}

	err := m.Add("test-id", "eth0", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil, nil)
	if err != nil {
		t.Fatalf("first Add failed: %v", err)
	}

	err = m.Add("test-id", "eth1", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil, nil)
	if err == nil {
		t.Error("expected error for duplicate ID")
	}
}

func TestMultiTunnelManager_ActiveCount(t *testing.T) {
	m := NewMultiTunnelManager()

	clientCfg := &config.ClientConfig{
		InterfaceName: "bcvpn1",
		TunIP:         "10.10.0.2",
		TunSubnet:     "24",
	}

	if m.ActiveCount() != 0 {
		t.Errorf("expected 0 active tunnels, got %d", m.ActiveCount())
	}

	err := m.Add("id1", "eth0", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil, nil)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if m.ActiveCount() != 1 {
		t.Errorf("expected 1 active tunnel, got %d", m.ActiveCount())
	}
}

func TestMultiTunnelManager_List(t *testing.T) {
	m := NewMultiTunnelManager()

	clientCfg := &config.ClientConfig{
		InterfaceName: "bcvpn1",
		TunIP:         "10.10.0.2",
		TunSubnet:     "24",
	}

	list := m.List()
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}

	m.Add("id1", "eth0", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil, nil)
	m.Add("id2", "eth1", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil, nil)

	list = m.List()
	if len(list) != 2 {
		t.Errorf("expected 2 items, got %d", len(list))
	}
	if list["id1"] != "eth0" {
		t.Errorf("expected id1 -> eth0, got %s", list["id1"])
	}
	if list["id2"] != "eth1" {
		t.Errorf("expected id2 -> eth1, got %s", list["id2"])
	}
}

func TestMultiTunnelManager_Cancel(t *testing.T) {
	m := NewMultiTunnelManager()

	clientCfg := &config.ClientConfig{
		InterfaceName: "bcvpn1",
		TunIP:         "10.10.0.2",
		TunSubnet:     "24",
	}

	err := m.Add("test-id", "eth0", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil, nil)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if m.ActiveCount() != 1 {
		t.Fatalf("expected 1 active tunnel")
	}

	m.Cancel("test-id")

	time.Sleep(100 * time.Millisecond)

	if m.ActiveCount() != 0 {
		t.Errorf("expected 0 active tunnels after cancel, got %d", m.ActiveCount())
	}
}

func TestMultiTunnelManager_CancelNonExistent(t *testing.T) {
	m := NewMultiTunnelManager()

	m.Cancel("non-existent")

	if m.ActiveCount() != 0 {
		t.Errorf("expected 0 active tunnels, got %d", m.ActiveCount())
	}
}

func TestMultiTunnelManager_CancelAll(t *testing.T) {
	m := NewMultiTunnelManager()

	clientCfg := &config.ClientConfig{
		InterfaceName: "bcvpn1",
		TunIP:         "10.10.0.2",
		TunSubnet:     "24",
	}

	m.Add("id1", "eth0", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil, nil)
	m.Add("id2", "eth1", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil, nil)
	m.Add("id3", "eth2", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil, nil)

	if m.ActiveCount() != 3 {
		t.Fatalf("expected 3 active tunnels")
	}

	m.CancelAll()

	time.Sleep(100 * time.Millisecond)

	if m.ActiveCount() != 0 {
		t.Errorf("expected 0 active tunnels after CancelAll, got %d", m.ActiveCount())
	}
}

func TestMultiTunnelManager_ConcurrentAdd(t *testing.T) {
	m := NewMultiTunnelManager()

	clientCfg := &config.ClientConfig{
		InterfaceName: "bcvpn1",
		TunIP:         "10.10.0.2",
		TunSubnet:     "24",
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			m.Add("id", "eth0", clientCfg, nil, nil, nil, "", ClientSecurityExpectations{}, nil, nil)
		}(i)
	}
	wg.Wait()
}

func TestActiveTunnel_String(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	tunnel := &ActiveTunnel{
		ID:        "test-id",
		ctx:       ctx,
		cancel:    cancel,
		done:      make(chan struct{}),
		Interface: "eth0",
	}

	if tunnel.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %s", tunnel.ID)
	}
	if tunnel.Interface != "eth0" {
		t.Errorf("expected Interface 'eth0', got %s", tunnel.Interface)
	}
}
