# GUI Demo Mode Guide

## Overview

Demo mode allows you to explore and test the BlockchainVPN GUI without needing a running blockchain node, RPC connection, or any backend infrastructure. This is perfect for:

- **UI/UX testing** - Verify interface layouts and workflows
- **Product demos** - Showcase the application to stakeholders
- **Development** - Work on GUI features without setting up the full stack
- **Onboarding** - Help new users understand the workflow before configuration

## What Demo Mode Does

When demo mode is enabled:

- **Scanning** returns simulated provider data instead of querying the blockchain
- **No RPC connections** are attempted (no need for `bitcoind` or similar)
- **Mock providers** appear with realistic pricing, bandwidth, and country information
- **Connection flow** is simulated up to the point where actual networking would begin
- **All UI features** work normally (filtering, sorting, selection, etc.)

## Enabling Demo Mode

### First Time Setup

1. Launch the BlockchainVPN GUI
2. Navigate to the **Settings** tab
3. Scroll to the bottom of the RPC/Global settings section
4. Check the **"Enable Demo Mode (simulate blockchain, no backend needed)"** checkbox
5. Click **Save + Validate**

![Demo Mode Setting](screenshots/demo-mode-setting.png)

> **Note:** You can also manually edit your `config.json` and set `"demo_mode": true`

### Verifying Demo Mode

After enabling demo mode:

1. Go to the **Log** panel at the bottom of the Settings tab
2. You should see: `[DEMO MODE] Using simulated provider data (no blockchain connection required)`
3. If you see RPC connection errors, demo mode is not enabled correctly

## Using Demo Mode

### Scanning Providers

1. Navigate to the **Client** tab
2. Adjust any filters you want (country, max price, minimum bandwidth)
3. Click **Scan**
4. The scan completes **instantly** (no blockchain query delay)
5. Results will show 1 mock provider with:
   - Country: US
   - IP: 198.51.100.x
   - Price: 1000 sats/session
   - Bandwidth: ~10 Mbps
   - Port: 51820

### Filtering and Sorting

All filtering and sorting options work with mock data:

- **Country filter** - Try "US" to always match the mock provider
- **Max price** - Set to e.g., 2000 to filter
- **Min bandwidth** - Set to e.g., 5000 Kbps
- **Sort by** - price, bandwidth, latency, capacity, score

### Connection Flow

When you select and connect to the mock provider:

1. Select the provider from the list
2. Click **Connect**
3. You'll see:
   - Payment confirmation prompt
   - "Sending payment..." message
   - Connection attempt to 198.51.100.1:51820
4. The connection will ultimately fail at the TLS handshake stage (expected)
5. This is normal - the mock provider isn't a real VPN server

## What Works in Demo Mode

✅ Provider scanning with filtering/sorting  
✅ Provider details display (country, price, bandwidth)  
✅ Payment prompts and amount verification  
✅ Wallet balance display (if RPC is configured)  
✅ Settings UI and config validation  
✅ Spending limit warnings (if configured)  
✅ All logging and event display  
✅ Metrics and status panels  

## What Doesn't Work in Demo Mode

❌ Actual network connections (TLS handshake fails)  
❌ TUN interface creation  
❌ Real payment transactions  
❌ Real provider availability checks  
❌ GeoIP latency measurements (may show 0ms)  
❌ Throughput probes  
❌ Certificate validation  

## Demo Mode Limitations

- **Single mock provider**: Only one simulated provider is returned (hardcoded to US, 1000 sats, 10Mbps)
- **No pricing variety**: Always uses session-based pricing (time/data models not showcased)
- **No dynamic behavior**: Provider count, prices, and countries are static
- **Connection fails**: This is expected - the endpoint isn't real

## Tips for Demo/Testing

1. **Combine with a real RPC config** - You can have demo mode enabled but also configure RPC for wallet balance display
2. **Test spending limits** - Enable spending limits in Client config and watch the warnings during connection
3. **Test different pricing methods** - The mock provider uses session pricing, but you can test the UI's ability to display different pricing models by manually editing config to use time/data methods
4. **Quick iteration** - Toggle demo mode off/on to reset to real scanning behavior

## Transitioning to Real Mode

When you're ready to use real providers:

1. Disable the **Demo Mode** checkbox in Settings
2. Configure your RPC connection (host, user, password, TLS)
3. Ensure your blockchain daemon is running and RPC accessible
4. Click **Save + Validate**
5. Navigate to Client tab and click **Scan**
6. The scan will now take a few minutes as it queries the blockchain

## Troubleshooting

### "Scan failed: connection refused"
- Demo mode is **disabled** but no RPC daemon is running
- Enable demo mode or start your blockchain node

### "No providers found"
- Demo mode is disabled and your blockchain has no VPN announcements
- Try scanning from a later block height or ensure providers are announcing

### "TLS handshake failed"
- Normal in demo mode - connection to mock provider fails
- In real mode, this means the provider's certificate is invalid or unreachable

### Settings not persisting
- Ensure you clicked **Save + Validate** after changing settings
- Check the Log panel for validation errors

## Sample Config for Demo

Here's a minimal config that works entirely in demo mode:

```json
{
  "rpc": {
    "host": "localhost:25173",
    "user": "rpcuser",
    "pass": "",
    "enable_tls": false
  },
  "provider": {
    "interface_name": "bcvpn0",
    "listen_port": 51820,
    "price_sats_per_session": 1000,
    "private_key_file": "~/.config/BlockchainVPN/provider.key"
  },
  "client": {
    "interface_name": "bcvpn1",
    "tun_ip": "10.10.0.2",
    "tun_subnet": "24"
  },
  "security": {
    "key_storage_mode": "file"
  },
  "demo_mode": true
}
```

With this config and demo mode enabled, you can scan and explore the UI without any running daemon.

## For Developers

### Extending Demo Mode

You can customize the mock provider data by modifying the `scanProviders` function in `cmd/bcvpn-gui/main.go`. Look for the `if demoMode {` block.

Current mock provider:
- IP: 198.51.100.1
- Port: 51820
- Price: 1000 sats
- Bandwidth: 10,000 Kbps (10 Mbps)
- Country: US
- Consumers: unlimited (0)
- Pricing: session-based

### Testing Different Pricing Models

To test the UI's handling of time/data pricing:

1. In Settings, under **Provider** section, set:
   - Pricing Method: `time` or `data`
   - Billing Time Unit: `minute` (for time) or Billing Data Unit: `MB` (for data)
2. Save and validate
3. Scan in demo mode - the mock provider will still show as session-based in the results, but the **Pricing Method** dropdown in the provider creation UI will reflect your setting

## Feedback

If you encounter issues with demo mode or have suggestions for improving the testing experience, please open an issue on GitHub with details.
