# Provider Guide

This guide explains how to become a VPN provider on the BlockchainVPN network.

## Overview

As a provider, you offer VPN services to clients who discover and pay you through the blockchain. Your service is announced via OP_RETURN transactions, making it decentralized and permissionless.

## Prerequisites

- Go 1.25.x or later
- Access to an OrdexCoin full node (RPC)
- Root/sudo access (for network configuration)
- Wallet with funds for announcement fees

## Quick Start

```bash
# 1. Generate default configuration
sudo ./bcvpn generate-config

# 2. Edit configuration
sudo nano ~/.config/BlockchainVPN/config.json

# 3. Start as a provider
sudo ./bcvpn start-provider
```

## Configuration

### Generate Default Config

```bash
./bcvpn generate-config
```

This creates `~/.config/BlockchainVPN/config.json` with sensible defaults.

### Provider Settings

Edit the `provider` section of your config:

```json
{
  "provider": {
    "listen_port": 51820,
    
    // Location - leave empty for auto-detect
    "announce_ip": "",           // Auto-detect: ipify.org, ifconfig.me, etc.
    "country": "",              // Auto-detect: ip-api.com
    
    // Capacity
    "max_consumers": 5,        // Max concurrent clients (0 = unlimited)
    
    // Bandwidth
    "bandwidth_limit": "0",    // "0" = auto-test, or "10mbit", "100mbit"
    "bandwidth_auto_test": true, // Run speedtest on startup
    
    // Payment - choose ONE method:
    "pricing_method": "session",  // "session", "time", or "data"
    "billing_time_unit": "minute", // "minute" or "hour" (for time)
    "billing_data_unit": "GB", // "MB" or "GB" (for data)
    "price_sats_per_session": 1000,
    
    // NAT Traversal
    "nat_traversal_method": "auto", // "auto", "upnp", "natpmp", "none"
    
    // Advanced
    "isolation_mode": "none",     // "none" or "sandbox"
    "health_check_interval": "30s",
    "announcement_interval": "24h",
    
    // Access Control (optional)
    "allowlist_file": "",        // Path to allowed pubkeys
    "denylist_file": "",        // Path to blocked pubkeys
  }
}
```

### RPC Connection

Configure connection to your OrdexCoin node:

```json
{
  "rpc": {
    "host": "localhost:25173",
    "user": "rpcuser",
    "pass": "YOUR_PASSWORD",
    "network": "mainnet",  // mainnet, testnet, or regtest
    "enable_tls": false
  }
}
```

### Full Example

See `config.provider.example.json` for a complete example provider configuration.

## Payment Methods

### Session-Based (Default)

Pay per connection. Simple and predictable.

```json
{
  "pricing_method": "session",
  "price_sats_per_session": 1000
}
```

### Time-Based

Pay per minute or hour. Good for short-term usage.

```json
{
  "pricing_method": "time",
  "billing_time_unit": "minute",  // or "hour"
  "price_sats_per_session": 50   // per minute
}
```

### Data-Based

Pay per MB or GB. Ideal for variable usage patterns.

```json
{
  "pricing_method": "data",
  "billing_data_unit": "GB",     // or "MB"
  "price_sats_per_session": 100  // per GB
}
```

## Starting Your Provider

### Basic Start

```bash
sudo ./bcvpn start-provider
```

Requires root for:
- Creating TUN interfaces
- Setting up routing/NAT
- Binding to privileged ports (<1024)

### With Environment Variables

```bash
# Key password from environment
sudo ./bcvpn start-provider --key-password-env BCVPN_KEY_PASSWORD

# Custom config path
./bcvpn -config /path/to/config.json start-provider

# Debug logging
DEBUG=1 ./bcvpn start-provider
```

### Production Considerations

```bash
# Run in background with logging
sudo ./bcvpn start-provider 2>&1 | tee provider.log

# Or use systemd/launchd for automatic restart
```

## What Happens on Startup

### 1. Auto-Detection

If configured, the following are auto-detected:

| Setting | Services Used |
|---------|--------------|
| External IP | api.ipify.org, ifconfig.me/ip, icanhazip.com, checkip.amazonaws.com |
| Country | ip-api.com |

### 2. NAT Traversal

Attempts in order (when `nat_traversal_method` is "auto"):

1. UPnP (Internet Gateway Device v2)
2. UPnP (Internet Gateway Device v1)
3. NAT-PMP

Set to `"none"` to disable, or `"upnp"`/`"natpmp"` to use specific method.

### 3. Bandwidth Test

If `bandwidth_auto_test: true`:

- Downloads test data from speed test servers
- Measures throughput
- Advertises measured bandwidth to clients

Or set `bandwidth_limit` directly (e.g., `"10mbit"`, `"100mbit"`).

### 4. Blockchain Announcement

Your provider endpoint is broadcast via OP_RETURN:

- Public key (identity)
- IP address and port
- Price and pricing method
- Bandwidth
- Country
- Available slots

### 5. Ongoing Operations

| Operation | Interval |
|----------|----------|
| Re-announce service | Every 24h (configurable) |
| Heartbeat | Every 5min (configurable) |
| Payment monitoring | Every 30s (configurable) |
| Health checks | Every 30s (configurable) |

## Management Commands

### Check Status

```bash
./bcvpn status
```

### Re-announce

```bash
./bcvpn rebroadcast
```

### View Payment History

```bash
./bcvpn history
./bcvpn history --since-last-payment
```

### Rotate Provider Key

```bash
./bcvpn rotate-provider-key --key-file /path/to/key --old-password-env OLD --new-password-env NEW
```

### Diagnostics

```bash
./bcvpn doctor
```

## Network Requirements

### Inbound Ports

- `listen_port` (default 51820): VPN connections
- `throughput_probe_port` (default 51821): Bandwidth testing
- `websocket_fallback_port`: WebSocket fallback (optional)

### Outbound

- RPC connection to OrdexCoin node
- External services for auto-detection (ip-api.com, etc.)

### NAT/Firewall

- Enable UPnP/NAT-PMP for automatic port forwarding
- Or manually forward ports to your machine

## Security

### Key Management

Provider private keys are stored in:
- Default: `~/.config/BlockchainVPN/provider.key`
- Custom: `provider.private_key_file` in config

### Encryption

```json
{
  "security": {
    "key_storage_mode": "file",      // or "keychain", "libsecret"
    "key_storage_service": "bcvpn",
    "tls_min_version": "1.3",
    "tls_profile": "modern"
  }
}
```

### Access Control

Block or allow specific clients:

```json
{
  "provider": {
    "allowlist_file": "/path/to/allowed.txt",  // hex pubkeys, one per line
    "denylist_file": "/path/to/blocked.txt"
  }
}
```

### Isolation Mode

```json
{
  "provider": {
    "isolation_mode": "sandbox"  // Additional network isolation
  }
}
```

## Troubleshooting

### Port Already in Use

```bash
# Check what's using the port
sudo lsof -i :51820

# Enable auto-rotate to use different port
"auto_rotate_port": true
```

### Can't Bind to Port

- Use ports above 1024
- Or run with sudo

### Not Discovering NAT

- Check router supports UPnP/NAT-PMP
- Manually forward ports
- Set `"nat_traversal_method": "none"` and configure manually

### No Connections

- Check firewall allows inbound connections
- Verify announcement appears on blockchain: `./bcvpn scan`
- Check logs for errors: `./bcvpn doctor`

### RPC Connection Issues

- Verify OrdexCoin node is running
- Check RPC credentials in config
- Test with: `curl -u user:pass http://localhost:25173`

## Performance Tuning

### Bandwidth

```json
{
  "bandwidth_monitor_interval": "30s",
  "throughput_probe_port": 51821
}
```

### Session Limits

```json
{
  "max_session_duration_secs": 3600,  // 1 hour max
  "cert_lifetime_hours": 720,
  "cert_rotate_before_hours": 24
}
```

### Resource Limits

```json
{
  "max_consumers": 10,  // Limit concurrent clients
  "bandwidth_limit": "100mbit"  // Cap bandwidth
}
```

## Metrics and Monitoring

Expose Prometheus metrics:

```json
{
  "metrics_listen_addr": "127.0.0.1:9090"
}
```

Access at: `http://127.0.0.1:9090/metrics`

## Summary

As a BlockchainVPN provider:

1. Configure your settings in `config.json`
2. Connect to an OrdexCoin node via RPC
3. Run `./bcvpn start-provider`
4. Your service is automatically announced to the blockchain
5. Clients discover and pay you directly in satoshis
6. No middleman, no censorship, full decentralization
