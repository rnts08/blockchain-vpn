# Consumer Guide

This guide explains how to use BlockchainVPN as a consumer/client to connect to VPN providers.

## Overview

As a consumer, you:
1. Scan the blockchain for available providers
2. Filter by country, bandwidth, price, and other criteria
3. Test the connection quality
4. Connect and pay providers directly in satoshis

## Prerequisites

- Access to an OrdexCoin full node (RPC)
- Wallet with funds for payments
- Root/sudo access (for network configuration)

## Quick Start

```bash
# 1. Generate default configuration
./bcvpn generate-config

# 2. Edit configuration  
nano ~/.config/BlockchainVPN/config.json

# 3. Scan for providers
./bcvpn scan

# 4. Follow interactive prompts to connect
```

## Scanning for Providers

### Basic Scan

```bash
./bcvpn scan
```

This will:
1. Scan the blockchain for provider announcements
2. Enrich with GeoIP and latency data
3. Display interactive list of providers
4. Allow you to select a provider to connect

### Filtered Scan

Filter providers by multiple criteria:

```bash
# By country
./bcvpn scan --country=US

# By maximum price (sats)
./bcvpn scan --max-price=1000

# By minimum bandwidth (Kbps)
./bcvpn scan --min-bandwidth-kbps=25000

# By maximum latency (ms)
./scan --max-latency-ms=80

# By minimum available slots
./bcvpn scan --min-available-slots=2

# By pricing method
./bcvpn scan --pricing-method=time

# Combined filters
./bcvpn scan --country=US --max-price=2000 --min-bandwidth-kbps=25000 --max-latency-ms=80 --min-available-slots=2
```

### Sorting Results

Sort providers by different metrics:

```bash
./bcvpn scan --sort=price       # Lowest price first
./bcvpn scan --sort=country     # By country
./bcvpn scan --sort=latency     # Fastest first
./bcvpn scan --sort=bandwidth   # Highest bandwidth
./bcvpn scan --sort=capacity    # Most available slots
./bcvpn scan --sort=score       # Best reputation score
./bcvpn scan --sort=latency     # Default: fastest
```

### Dry Run

Test without actually connecting:

```bash
./bcvpn scan --dry-run --country=US
```

This simulates the connection without spending funds or modifying interfaces.

## Connection Process

### Interactive Selection

When you run `./bcvpn scan`, you'll see an interactive menu:

```
1. US - New York (5 slots, 100Mbps, 50ms, 1000 sats)
2. DE - Frankfurt (3 slots, 50Mbps, 80ms, 500 sats)
3. JP - Tokyo (2 slots, 200Mbps, 120ms, 1500 sats)

Select provider (1-3): 
```

### What Happens During Connection

1. **Key Generation**: Creates a temporary key pair for TLS
2. **Payment**: Sends transaction to provider's address with OP_RETURN
3. **Tunnel Setup**: Establishes TLS-over-TUN connection
4. **Security Checks**: Verifies connection quality
5. **Auto-Settlement**: Ongoing micropayments as usage accumulates

## Security Checks

After connecting, the client performs automatic security verification:

### Egress IP Verification
- Verifies your traffic actually exits from the provider's IP
- Detects DNS leaks or routing issues

### DNS Leak Test
- Ensures DNS queries go through the VPN tunnel
- Prevents DNS-based location leaks

### Provider Country Verification
- Confirms provider's claimed location matches actual exit

### Throughput Verification
- Tests actual bandwidth matches advertised
- Enabled by default: `verify_throughput_after_connect: true`

Configure in `config.json`:

```json
{
  "client": {
    "strict_verification": true,
    "verify_throughput_after_connect": true
  }
}
```

## Payment & Settlement

### Payment Address

Configure which wallet address to pay from:

```json
{
  "rpc": {
    "host": "localhost:25173",
    "user": "rpcuser", 
    "pass": "YOUR_PASSWORD",
    "network": "mainnet"
  }
}
```

The client uses your RPC wallet to send payments to providers.

### Payment Methods

Providers can use different pricing models:

| Method | Description | Example |
|--------|-------------|---------|
| Session | Pay once per connection | 1000 sats |
| Time | Pay per minute/hour | 50 sats/minute |
| Data | Pay per MB/GB | 100 sats/GB |

### Auto-Settlement

The client automatically settles payments:

- Monitors usage (time/data)
- Sends additional payments as threshold reached
- Configurable via `client` settings

```json
{
  "client": {
    "spending_limit_enabled": true,
    "spending_limit_sats": 50000,
    "spending_warning_percent": 80,
    "auto_disconnect_on_limit": true
  }
}
```

### Auto-Recharge

Enable automatic top-ups:

```json
{
  "client": {
    "auto_recharge_enabled": true,
    "auto_recharge_threshold": 500,
    "auto_recharge_amount": 10000,
    "auto_recharge_min_balance": 100
  }
}
```

## Configuration

### Client Settings

```json
{
  "client": {
    "interface_name": "bcvpn1",
    "tun_ip": "10.10.0.2",
    "tun_subnet": "24",
    
    // DNS
    "dns_servers": ["1.1.1.1", "8.8.8.8"],
    
    // Security
    "enable_kill_switch": true,
    "strict_verification": true,
    "verify_throughput_after_connect": true,
    
    // Multi-tunnel
    "max_parallel_tunnels": 1,
    
    // Fallback
    "enable_websocket_fallback": false,
    
    // Limits
    "spending_limit_enabled": true,
    "spending_limit_sats": 50000,
    "spending_warning_percent": 80,
    "auto_disconnect_on_limit": true,
    "max_session_spending_sats": 10000,
    
    // Auto-recharge
    "auto_recharge_enabled": true,
    "auto_recharge_threshold": 500,
    "auto_recharge_amount": 10000,
    "auto_recharge_min_balance": 100,
    
    // Monitoring
    "metrics_listen_addr": "127.0.0.1:9091"
  }
}
```

### Save Favorite Providers

Export your current configuration as a profile:

```bash
./bcvpn config export ~/my-vpn-profile.json
```

Import a saved profile:

```bash
./bcvpn config import ~/my-vpn-profile.json
```

### Save Provider Pubkeys

You can manually save trusted provider public keys:

```bash
# Provider key is shown in scan output
# Save to file (one hex pubkey per line)
echo "02abc123..." > ~/.config/BlockchainVPN/trusted_providers.txt
```

## Management Commands

### View Configuration

```bash
# Get current config
./bcvpn config get client

# Get specific value
./bcvpn config get client.dns_servers

# Export full config
./bcvpn config export config.json
./bcvpn config export config.json --json
```

### Update Configuration

```bash
# Set values
./bcvpn config set client.enable_kill_switch true
./bcvpn config set client.max_parallel_tunnels 3
./bcvpn config set client.dns_servers '["1.1.1.1","8.8.8.8"]'

# Validate config
./bcvpn config validate
```

### View Payment History

```bash
./bcvpn history
./bcvpn history --since-last-payment
```

### Check Status

```bash
./bcvpn status
./bcvpn status --json
```

## Troubleshooting

### Connection Issues

```bash
# Check configuration
./bcvpn config validate

# Run diagnostics
./bcvpn doctor
```

### RPC Connection Issues

Verify RPC connection:

```bash
curl -u user:pass http://localhost:25173/getblockchaininfo
```

### Payment Issues

Check wallet balance:

```bash
./bcvpn history
```

### Speed Issues

Run scan with throughput test:

```bash
./bcvpn scan --min-bandwidth-kbps=50000
```

## Metrics

Expose client metrics:

```json
{
  "client": {
    "metrics_listen_addr": "127.0.0.1:9091"
  }
}
```

Access metrics:

```bash
curl http://127.0.0.1:9091/metrics.json
```

With auth token:

```bash
curl -H "X-BCVPN-Metrics-Token: your-token" http://127.0.0.1:9091/metrics.json
```

## Kill Switch

Enable kill switch to block traffic if VPN disconnects:

```json
{
  "client": {
    "enable_kill_switch": true
  }
}
```

This prevents data leaks if the VPN connection drops.

## Multi-Tunnel Support

Connect to multiple providers simultaneously:

```json
{
  "client": {
    "max_parallel_tunnels": 3
  }
}
```

Use cases:
- Load balancing
- Redundant connections
- Geographic diversity

## Summary

As a BlockchainVPN consumer:

1. **Scan**: `./bcvpn scan --country=US --max-price=1000`
2. **Select**: Choose from interactive list
3. **Connect**: Automatic payment and tunnel setup
4. **Verify**: Security checks run automatically
5. **Settle**: Ongoing micropayments as you use data/time

No registration, no subscription, pay only for what you use.
