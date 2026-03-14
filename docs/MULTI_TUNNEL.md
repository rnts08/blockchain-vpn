# Multi-Tunnel Concurrency Implementation

This document describes the multi-tunnel concurrency architecture for BlockchainVPN.

## Overview

The MultiTunnelManager allows clients to connect to multiple VPN providers simultaneously, enabling:
- Load balancing across providers
- Redundant connections for reliability
- Geographic diversity
- Split tunneling to different providers

## Core Components

### MultiTunnelManager (`internal/tunnel/multi_tunnel.go`)

Manages multiple concurrent VPN tunnel connections.

**Structure:**
```go
type MultiTunnelManager struct {
    tunnels map[string]*ActiveTunnel
    mu      sync.RWMutex
}
```

**Key Methods:**

| Method | Description |
|--------|-------------|
| `Add()` | Starts a new tunnel in a goroutine |
| `Cancel()` | Stops a specific tunnel by ID |
| `CancelAll()` | Shuts down all active tunnels |
| `List()` | Returns map of tunnel IDs to interface names |
| `ActiveCount()` | Returns number of active tunnels |

### ActiveTunnel

Represents a single running VPN connection:

```go
type ActiveTunnel struct {
    ID        string
    ctx       context.Context
    cancel    context.CancelFunc
    done      chan struct{}
    err       error
    Interface string  // TUN interface name (e.g., bcvpn0)
}
```

## Connection Flow

1. **Add tunnel**: Client calls `Add()` with:
   - Unique tunnel ID
   - Network interface name
   - Provider endpoint information
   - Security configuration
   - Optional spending manager

2. **Goroutine launch**: Tunnel starts in separate goroutine with:
   - 30-second connection timeout
   - Context for cancellation
   - Done channel for completion notification

3. **Connection**: `ConnectToProvider()` establishes:
   - TLS handshake with provider
   - TUN interface setup
   - Payment verification

4. **Tracking**: Manager tracks active tunnels in map

5. **Cleanup**: On completion:
   - Tunnel removes itself from map
   - Resources are released

## Concurrency Model

### Thread Safety

- `tunnels` map protected by `sync.RWMutex`
- Reads (List, ActiveCount) use RLock
- Writes (Add, Cancel, delete) use Lock

### Cancellation

- Each tunnel has its own context with timeout
- `Cancel()` calls context cancel function
- Waits for tunnel goroutine to exit via done channel

### Graceful Shutdown

```go
func (m *MultiTunnelManager) CancelAll() {
    m.mu.RLock()
    for id, t := range m.tunnels {
        go func(tID string, tunnel *ActiveTunnel) {
            tunnel.cancel()
            <-tunnel.done
        }(id, t)
    }
    m.mu.RUnlock()
    wg.Wait()
}
```

## Interface Management

Each tunnel gets a unique network interface:
- `bcvpn0`, `bcvpn1`, etc. (on Linux)
- Traffic routed through specific interface
- Enables per-tunnel routing policies

## Error Handling

- Connection errors are captured in `tunnel.err`
- Errors are logged before tunnel removal
- Client can check errors via callback or polling

## Usage Example

```go
manager := NewMultiTunnelManager()

// Connect to multiple providers
manager.Add("us-east", "bcvpn0", cfg, secCfg, localKey, usProviderPubKey, "us-provider:443", expectations, spendingMgr)
manager.Add("eu-west", "bcvpn1", cfg, secCfg, localKey, euProviderPubKey, "eu-provider:443", expectations, spendingMgr)

// List active tunnels
for id, iface := range manager.List() {
    fmt.Printf("Tunnel %s on %s\n", id, iface)
}

// Cancel specific tunnel
manager.Cancel("us-east")

// Shutdown all
manager.CancelAll()
```

## Limitations

- Maximum concurrent tunnels depends on system resources
- Each tunnel requires separate TUN interface
- Network routing must be configured per-tunnel
