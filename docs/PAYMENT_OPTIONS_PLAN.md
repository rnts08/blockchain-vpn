# Payment Options Enhancement Plan

## Overview

This document outlines the plan to extend BlockchainVPN's pricing and payment model to support multiple pricing methods, usage-based billing, spending limits, and multi-blockchain/network support.

---

## Current State

### Existing Pricing Model
- **Single method**: Flat fee per session (price_sats_per_session)
- Payment is one-time upfront before session starts
- Provider monitors blockchain for transactions meeting or exceeding the advertised price
- Client authorization lasts for 24 hours after payment confirmation

### Limitations
- No support for time-based or data-based billing
- No granular usage tracking during session
- No client-side spending controls or warnings
- Blockchain/network configuration is hardcoded to RPC only
- Token symbol not configurable for display purposes

---

## Proposed Enhancements

### 1. Multiple Pricing Methods

Support three distinct billing models:

| Method | Description | Use Case |
|--------|-------------|----------|
| **Session** | Flat fee per session (current behavior) | Traditional VPN per-connection billing |
| **Time** | Billing per time unit (minute/hour) | Pay-as-you-go time-based subscriptions |
| **Data** | Billing per data unit (MB/GB) | Volume-based bandwidth billing |

#### Implementation Details

##### Protocol Extension

**File**: `internal/protocol/vpn_protocol.go`

Add new constants and struct:

```go
// Pricing method types
const (
    PricingMethodSession = 0  // Flat fee per session
    PricingMethodTime    = 1  // Price per time unit
    PricingMethodData    = 2  // Price per data unit
)

type PricingModel struct {
    Method     uint8   // Session/Time/Data
    Price      uint64  // Price in token's smallest unit (e.g., satoshis)
    TimeUnit   uint32  // For time-based: seconds per billing cycle (e.g., 60 for per-minute)
    DataUnit   uint32  // For data-based: bytes per billing unit (e.g., 1_000_000 for MB)
}
```

Extend `VPNEndpoint`:

```go
type VPNEndpoint struct {
    IP                    net.IP
    Port                  uint16
    Price                 uint64           // Satoshis per session (legacy, kept for v1 compatibility)
    PublicKey             *btcec.PublicKey
    AdvertisedBandwidthKB uint32
    MaxConsumers          uint16
    CountryCode           string
    AvailabilityFlags     uint8
    ThroughputProbePort   uint16
    CertFingerprint       []byte

    // New fields (v2 extended)
    PricingModel          // Embedded (24 bytes)
    SessionTimeout        uint32  // Max session duration in seconds (0 = unlimited)
}
```

Update `EncodePayloadV2()` and `DecodePayloadV2()` to include new fields. To maintain backward compatibility:
- Use a new magic byte `0x56504E03` (VPN v3) for announcements with extended pricing
- Alternatively, extend v2 by checking remaining buffer length (length-prefixed optional fields)

**Recommended**: Create v3 protocol with explicit feature flags.

##### Configuration Extension

**File**: `internal/config/config.go`

Add to `ProviderConfig`:

```go
type ProviderConfig struct {
    InterfaceName               string `json:"interface_name"`
    ListenPort                  int    `json:"listen_port"`
    AutoRotatePort              bool   `json:"auto_rotate_port"`
    AnnounceIP                  string `json:"announce_ip"`
    Country                     string `json:"country"`
    Price                       uint64 `json:"price_sats_per_session"` // legacy, keep
    MaxConsumers                int    `json:"max_consumers"`
    PrivateKeyFile              string `json:"private_key_file"`
    BandwidthLimit              string `json:"bandwidth_limit"`
    EnableNAT                   bool   `json:"enable_nat"`
    EnableEgressNAT             bool   `json:"enable_egress_nat"`
    NATOutboundInterface        string `json:"nat_outbound_interface"`
    IsolationMode               string `json:"isolation_mode"`
    AllowlistFile               string `json:"allowlist_file"`
    DenylistFile                string `json:"denylist_file"`
    CertLifetimeHours           int    `json:"cert_lifetime_hours"`
    CertRotateBeforeHours       int    `json:"cert_rotate_before_hours"`
    HealthCheckEnabled          bool   `json:"health_check_enabled"`
    // ... existing fields ...

    // NEW fields for flexible pricing:
    PricingMethod               string `json:"pricing_method"`   // "session", "time", "data"
    BillingTimeUnit             string `json:"billing_time_unit"` // "minute", "hour"
    BillingDataUnit             string `json:"billing_data_unit"` // "MB", "GB"
    SessionTimeoutSec           int    `json:"session_timeout_sec"` // 0 = no timeout
}
```

**Default config** should set `pricing_method` to `"session"` to preserve existing behavior.

---

### 2. Usage Metering and Incremental Payments

#### Provider Side

**Current**: `MonitorPayments` authorizes a client for 24 hours upon receiving a qualifying payment. Expiration is fixed.

**Changes**:
- Track session usage (time and data) per connected client
- When a client's authorization is nearing expiration due to billing cycles, provider expects another payment
- Provider continues to monitor blockchain; each additional payment extends authorization by another billing cycle

**Implementation**: Extend `auth.AuthManager` to support multiple authorization entries per peer with different expiration times, or store usage metrics in session context and check against cumulative paid amounts.

**Recommended approach**: Keep simple - each payment grants a fixed duration (e.g., next 5 minutes for time-based, or next 100 MB for data-based). Provider's `processTxForPayment` calculates billing cycle based on payment amount and provider's pricing model:

```go
// For time-based: amount / price_per_minute = minutes_granted
// For data-based: amount / price_per_mb = mb_granted
// Store in auth manager as "remaining_quota" with timestamp
```

#### Client Side

**New component**: `internal/tunnel/meter.go`

```go
type UsageMeter struct {
    sessionStart    time.Time
    bytesSent       uint64
    bytesReceived   uint64
    pricingModel    *PricingModel
    totalBudget     uint64 // total prepaid amount in sats
    spentAmount     uint64 // cumulative amount charged
    mu              sync.RWMutex
}

func (m *UsageMeter) AddTraffic(sent, received uint64) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.bytesSent += sent
    m.bytesReceived += received
}

func (m *UsageMeter) CurrentCost() uint64 {
    m.mu.RLock()
    defer m.mu.RUnlock()
    // Calculate based on pricing model:
    // - Session: fixed price (already charged)
    // - Time: elapsed_seconds / time_unit * price
    // - Data: (bytes_sent + bytes_received) / data_unit * price
}

func (m *UsageMeter) NeedsPayment(thresholdPercent uint32) bool {
    // True when spent >= threshold% of totalBudget
}
```

The client's payment flow:

1. Before connecting, client estimates required prepayment:
   - For session: exactly advertised price
   - For time: e.g., 1 hour worth = price_per_hour
   - For data: e.g., 1 GB worth = price_per_gb

2. Client sends payment transaction with appropriate amount.

3. During session, `UsageMeter` tracks consumption.

4. Periodic check: if remaining budget < warning threshold, log warning.

5. If `AutoPay` enabled and remaining budget < auto_pay threshold:
   - Automatically send additional payment
   - Extend session (provider authorizes based on new payment)

6. If budget exhausted:
   - If `AutoDisconnectOnLimit`, gracefully terminate session
   - Otherwise, allow continuation until provider revokes (after current billing cycle)

---

### 3. Client Spending Limits and Controls

#### Configuration

Add to `ClientConfig` in `internal/config/config.go`:

```go
type ClientConfig struct {
    InterfaceName              string `json:"interface_name"`
    TunIP                      string `json:"tun_ip"`
    TunSubnet                  string `json:"tun_subnet"`
    EnableKillSwitch           bool   `json:"enable_kill_switch"`
    MetricsListenAddr          string `json:"metrics_listen_addr"`
    StrictVerification         bool   `json:"strict_verification"`
    VerifyThroughputAfterSetup bool   `json:"verify_throughput_after_connect"`
    MaxParallelTunnels         int    `json:"max_parallel_tunnels"`
    EnableWebSocketFallback    bool   `json:"enable_websocket_fallback"`

    // NEW: Spending management
    SpendingLimitEnabled       bool   `json:"spending_limit_enabled"`
    SpendingLimitSats          uint64 `json:"spending_limit_sats"`         // Total cap per period (e.g., daily)
    SpendingWarningPercent     uint32 `json:"spending_warning_percent"`    // Warning when spent >= X% (e.g., 80)
    AutoDisconnectOnLimit      bool   `json:"auto_disconnect_on_limit"`    // Auto-disconnect at limit
    AutoRechargeEnabled        bool   `json:"auto_recharge_enabled"`       // Existing
    AutoRechargeThreshold      uint64 `json:"auto_recharge_threshold"`     // Existing
    AutoRechargeAmount         uint64 `json:"auto_recharge_amount"`        // Existing
    AutoRechargeMinBalance     uint64 `json:"auto_recharge_min_balance"`   // Existing

    // NEW: Per-session limits
    MaxSessionSpendingSats     uint64 `json:"max_session_spending_sats"`   // Max per connection (0 = unlimited)
}
```

#### Spending Manager

**File**: `internal/tunnel/spending_manager.go` (new)

Extract and enhance existing `CreditManager`:

```go
type SpendingManager struct {
    mu                    sync.RWMutex
    totalSpentToday       uint64 // or per-configurable period
    dailyResetTime        time.Time
    limitEnabled          bool
    totalLimit            uint64
    sessionStartSpent     uint64 // snapshot at session start
    sessionMax            uint64
    autoDisconnect        bool
    warningPercent        uint32
    warningIssued         bool

    // Auto-recharge fields (from CreditManager)
    rechargeEnabled       bool
    rechargeThreshold     uint64
    rechargeAmount        uint64
    minBalance            uint64
    lastRecharge          time.Time

    client                *rpcclient.Client
    providerAddr          btcutil.Address
    localKey              *btcec.PrivateKey
}

func (sm *SpendingManager) RecordPayment(amount uint64) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    sm.totalSpentToday += amount
    sm.sessionStartSpent += amount // track both

    // Check limits
    if sm.limitEnabled && sm.totalSpentToday >= sm.totalLimit {
        return fmt.Errorf("spending limit reached: %d >= %d", sm.totalSpentToday, sm.totalLimit)
    }
    if sm.sessionMax > 0 && (sm.sessionStartSpent + amount) > sm.sessionMax {
        return fmt.Errorf("session spending limit reached: %d >= %d", sm.sessionStartSpent+amount, sm.sessionMax)
    }
    // Check warning threshold
    if !sm.warningIssued {
        spentPercent := uint32(float64(sm.totalSpentToday) / float64(sm.totalLimit) * 100)
        if spentPercent >= sm.warningPercent {
            log.Printf("WARNING: Spending at %d%% of limit (%d/%d sats)", spentPercent, sm.totalSpentToday, sm.totalLimit)
            sm.warningIssued = true
        }
    }
    return nil
}

func (sm *SpendingManager) ShouldDisconnect() bool {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    if sm.autoDisconnect && sm.limitEnabled && sm.totalSpentToday >= sm.totalLimit {
        return true
    }
    return false
}

func (sm *SpendingManager) RemainingBudget() uint64 {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    if !sm.limitEnabled {
        return ^uint64(0) // unlimited
    }
    if sm.totalSpentToday >= sm.totalLimit {
        return 0
    }
    return sm.totalLimit - sm.totalSpentToday
}
```

Integrate with payment flow:
- Before sending payment, check `RecordPayment` - if would exceed limit, abort or require user confirmation.
- After successful payment, call `RecordPayment` to update spent total.
- During session, periodically check `ShouldDisconnect()`.

---

### 4. Multi-Blockchain and Network Support

#### Problem

The application currently uses hardcoded RPC connection details and assumes Bitcoin-like blockchain with satoshis. Need to support:
- Multiple networks (mainnet/testnet/regtest/simnet/custom)
- Different RPC ports and configurations
- Different token symbols and decimal places for display

#### Configuration Changes

**File**: `internal/config/config.go`

Modify `RPCConfig`:

```go
type RPCConfig struct {
    Host        string `json:"host"`        // host:port (e.g., "localhost:18443")
    User        string `json:"user"`
    Pass        string `json:"pass"`
    EnableTLS   bool   `json:"enable_tls"`
    Network     string `json:"network"`      // "mainnet", "testnet", "regtest", "simnet", "custom"
    TokenSymbol string `json:"token_symbol"` // Display symbol: "BTC", "LTC", "ORDEX", etc.
    TokenDecimals int  `json:"token_decimals"` // Number of decimal places (default: 8)
}
```

Add new config section for network parameters (if custom):

```go
type NetworkConfig struct {
    Name          string `json:"name"`           // human-readable
    MagicBytes    string `json:"magic_bytes"`    // hex, for blockchain P2P (not VPN protocol)
    AddressHrp    string `json:"address_hrp"`    // bech32 human-readable part
    BIP44CoinType  int   `json:"bip44_coin_type"` // e.g., 0 for BTC, 1 for testnet, etc.
}
```

But simpler: use `chaincfg.Params` from btcd suite and auto-detect from RPC.

#### Network Detection

**File**: `internal/blockchain/network.go` (new)

```go
package blockchain

import (
    "github.com/btcsuite/btcd/chaincfg"
    "github.com/btcsuite/btcd/chaincfg/chainhash"
    "github.com/btcsuite/btcd/rpcclient"
)

var KnownNetworks = map[string]*chaincfg.Params{
    "mainnet":  &chaincfg.MainNetParams,
    "testnet":  &chaincfg.TestNet3Params,
    "regtest":  &chaincfg.RegressionNetParams,
    "simnet":   &chaincfg.SimNetParams,
}

func DetectNetwork(client *rpcclient.Client) (*chaincfg.Params, error) {
    info, err := client.GetBlockChainInfo()
    if err != nil {
        return nil, err
    }
    genesis, err := client.GetBlockHash(0)
    if err != nil {
        return nil, err
    }
    // Compare genesis hash to known networks
    for name, params := range KnownNetworks {
        if genesis.IsEqual(params.GenesisHash) {
            return params, nil
        }
    }
    // Unknown network - require explicit config
    return nil, fmt.Errorf("unknown network, set config.rpc.network explicitly")
}

func GetTokenInfo(network string) (symbol string, decimals int) {
    switch network {
    case "mainnet":
        return "ORDEX", 8 // Or detect from coin?
    case "testnet":
        return "ORDEX-TEST", 8
    case "regtest":
        return "ORDEX-RT", 8
    default:
        return "COIN", 8
    }
}
```

#### RPC Connection

**File**: `cmd/bcvpn/main.go` (or create `internal/blockchain/rpc.go`)

Update `connectRPCWithConfig` to use detected network parameters:

```go
func connectRPCWithConfig(cfg *config.Config) *rpcclient.Client {
    // Determine chain parameters
    var params *chaincfg.Params
    network := strings.ToLower(strings.TrimSpace(cfg.RPC.Network))
    if network != "" && network != "auto" {
        if p, ok := blockchain.KnownNetworks[network]; ok {
            params = p
        } else {
            log.Fatalf("Unknown network: %s", network)
        }
    }

    // Connect
    connCfg := &rpcclient.ConnConfig{
        Host:         cfg.RPC.Host,
        User:         cfg.RPC.User,
        Pass:         cfg.RPC.Pass,
        HTTPPostMode: true,
        EnableTLS:    cfg.RPC.EnableTLS,
        // ChainParams needed for address decoding/creating
        ChainParams:  params or chaincfg.MainNetParams, // fallback
    }
    return rpcclient.New(connCfg, nil)
}
```

#### Display Formatting

Create utility: `internal/util/format.go`

```go
func FormatAmount(amount uint64, symbol string, decimals int) string {
    if decimals == 0 {
        return fmt.Sprintf("%d %s", amount, symbol)
    }
    integer := amount / uint64(10^decimals)
    fractional := amount % uint64(10^decimals)
    return fmt.Sprintf("%d.%0*d %s", integer, decimals, fractional, symbol)
}
```

Use throughout CLI/GUI for consistent display.

---

### 5. Filtering Enhancements

#### Scanner Filters

**File**: `cmd/bcvpn/main.go` (scan command)

Add flags:

```go
scanPricingMethod := scanCmd.String("pricing-method", "", "Filter by pricing method: session, time, data")
scanMaxPricePerUnit := scanCmd.Float64("max-price-per-unit", 0, "Max price per billing unit (e.g., sats/min or sats/GB)")
scanPricingUnit := scanCmd.String("pricing-unit", "", "Unit for max-price-per-unit: 'min', 'hour', 'MB', 'GB'")
```

Update `filterEndpoints()` to accept pricing criteria.

**File**: `internal/blockchain/scanner.go` → add `PricingMethod` and `UnitPrice` to `ProviderAnnouncement`.

---

### 6. Validation

**File**: `internal/config/validate.go`

Add validation:

```go
// Provider pricing validation
pricingMethod := strings.ToLower(strings.TrimSpace(cfg.Provider.PricingMethod))
switch pricingMethod {
case "", "session", "time", "data":
    // valid
default:
    errs = append(errs, fmt.Errorf("provider.pricing_method must be one of: session, time, data"))
}

if pricingMethod == "time" {
    if cfg.Provider.BillingTimeUnit == "" {
        errs = append(errs, fmt.Errorf("provider.billing_time_unit required when pricing_method=time"))
    }
}
if pricingMethod == "data" {
    if cfg.Provider.BillingDataUnit == "" {
        errs = append(errs, fmt.Errorf("provider.billing_data_unit required when pricing_method=data"))
    }
}

// Validate spending limits
if cfg.Client.SpendingLimitEnabled {
    if cfg.Client.SpendingLimitSats == 0 {
        errs = append(errs, fmt.Errorf("client.spending_limit_sats required when spending_limit_enabled=true"))
    }
    if cfg.Client.SpendingWarningPercent > 100 {
        errs = append(errs, fmt.Errorf("client.spending_warning_percent must be 0-100"))
    }
}
```

---

### 7. Backward Compatibility Strategy

- **Protocol**: Keep v1 and v2 support. New v3 format with extended fields. Clients should decode v1/v2 as `PricingMethodSession` (legacy).
- **Config**: New fields optional; defaults preserve current behavior.
- **Provider**: Old providers only announce session pricing → new clients treat as session method.
- **Client**: Old clients ignore providers with time/data pricing (filter them out by default, or show incompatibility warning).
- **GUI**: Hide/show fields based on pricing method selection.

---

### 8. Implementation Phases

#### Phase 1: Core Protocol and Config (Week 1-2)
- [ ] Extend `VPNEndpoint` with `PricingModel` and `SessionTimeout`
- [ ] Update `EncodePayloadV2`/`DecodePayloadV2` (or create v3)
- [ ] Add new config fields with defaults
- [ ] Update validation
- [ ] Update `buildProviderEndpoint` to use new fields
- [ ] Update `ScanForVPNs` and `ProviderAnnouncement`
- [ ] Add unit tests for protocol encoding/decoding

#### Phase 2: Client Filtering and UI (Week 3)
- [ ] Add CLI scan flags for pricing method and unit price
- [ ] Implement filtering logic
- [ ] Update GUI scan dialog (filter panel)
- [ ] Update provider settings UI in GUI
- [ ] Add display formatting utility
- [ ] Update status output to show pricing model

#### Phase 3: Usage Metering (Week 4-5)
- [ ] Implement `UsageMeter` in new file
- [ ] Integrate with session traffic counters
- [ ] For time-based: periodic cost calculation
- [ ] For data-based: byte-based cost calculation
- [ ] Extend `AuthManager` to track remaining quota per peer (or use separate store)
- [ ] Modify provider payment monitor to grant quota based on payment amount
- [ ] Implement client-side auto-pay (reuse/rename CreditManager → SpendingManager)
- [ ] Tests for metering calculations

#### Phase 4: Spending Limits (Week 6)
- [ ] Create `SpendingManager` (split from CreditManager)
- [ ] Integrate with payment flow (pre-spend checks)
- [ ] Implement daily spending tracking (persist to disk?)
- [ ] Warning and auto-disconnect logic
- [ ] Update history recording to track spending per session
- [ ] CLI/GUI settings for spending limits
- [ ] Tests for limit enforcement

#### Phase 5: Multi-Blockchain (Week 7)
- [ ] Add `Network` and `TokenSymbol`/`TokenDecimals` to RPC config
- [ ] Create `internal/blockchain/network.go` with detection
- [ ] Update RPC connection to use appropriate chaincfg.Params
- [ ] Update fee estimation if needed (should be generic)
- [ ] Update token symbol display throughout app
- [ ] Tests for network detection and parameter handling
- [ ] Documentation for multi-chain setup

#### Phase 6: Integration and Testing (Week 8)
- [ ] End-to-end tests: time-based billing roundtrip
- [ ] End-to-end tests: data-based billing
- [ ] End-to-end tests: spending limits
- [ ] End-to-end tests: network switching
- [ ] Update README with new config options
- [ ] Update sample config.json
- [ ] Migration guide for existing users

---

### 9. Data Model Changes Summary

#### Protocol Buffer (OP_RETURN) - v3 proposal

| Field | Size (bytes) | Description |
|-------|--------------|-------------|
| Magic | 4 | `0x56504E03` |
| IP Type | 1 | 0x04/0x06 |
| IP Address | 4 or 16 |
| Port | 2 |
| Pricing Method | 1 | 0=Session, 1=Time, 2=Data |
| Price | 8 | Price per billing unit (sats) |
| Time/Data Unit | 4 | Seconds for time, bytes for data |
| Bandwidth Limit KB | 4 | (existing AdvertisedBandwidthKB) |
| Max Consumers | 2 |
| Country Code | 2 |
| Availability Flags | 1 |
| Throughput Probe Port | 2 |
| Session Timeout | 4 | Max session seconds |
| Public Key | 33 |

Total: ~78 bytes for v3 (vs ~66 for v2)

#### Config Struct Changes

**ProviderConfig**: +6 fields
**ClientConfig**: +5 fields
**RPCConfig**: +3 fields

---

### 10. Testing Strategy

**Unit Tests**:
- Protocol encode/decode for v1, v2, v3
- Pricing cost calculation (time/data)
- Spending limit arithmetic
- Network detection logic

**Integration Tests**:
- Time-based: provider with per-minute pricing, client connects, uses 2 minutes → payment amount correct
- Data-based: provider with per-GB pricing, transfer 100MB → partial payment triggered
- Spending limit: set limit 1000 sats, session uses 800 → warning; next payment would exceed → disconnect
- Multi-chain: connect to regtest vs mainnet RPC

**Manual QA**:
- GUI: all new fields visible and editable
- Scan: filtering by pricing method works
- Status: token symbol displayed correctly

---

### 11. Open Questions

1. **Payment granularity**: Should clients send many small payments (risk: high fees) or larger prepayments? → Prefer block prepayment (e.g., 1 hour or 1 GB chunks).
2. **Provider reorg handling**: Quota-based auth may be lost on reorg; need to persist quota per pubkey with block height? → Keep 24h rule, but recalc expired peers periodically.
3. **Multi-token**: If supporting different chains with different decimals, do we need to handle fee estimation in different units? → Keep fees in smallest unit.
4. **Client-side wallet balance check**: Should we warn if wallet balance < required prepayment? → Nice to have.
5. **Spending period**: Daily limit implies daily reset. What defines "day"? UTC midnight or rolling 24h? → Implement both options? Simpler: session-based limits only initially.

---

### 12. Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Complex billing logic introduces bugs | Financial loss, poor UX | Extensive unit tests; start with conservative defaults; logging |
| High on-chain fee burden for micro-payments | Users overpay in fees | Recommend larger billing chunks; fee estimation already dynamic |
| Reorg causes quota loss | Unauthorized disconnects | Keep 24h max cap; payments with >6 confirmations are secure |
| Multi-chain config confusion | Users misconfigure RPC | Validate RPC network against GetBlockChainInfo; provide clear errors |
| Provider overhead for continuous monitoring | CPU/memory usage | Quota checking is cheap; keep simple data structures |
| Client disconnect happens mid-session | Data loss | Allow grace period; warn before disconnect |
| Backward compat issues | Old clients can't use new providers | Default to session-based; filter unknown methods by default |

---

### 13. Success Criteria

- [ ] Provider can set pricing method to session/time/data with appropriate units
- [ ] Client can filter providers by pricing method and unit price
- [ ] Client can set spending limits (total and per-session) with warnings and auto-disconnect
- [ ] Token symbol displays correctly in CLI/GUI based on config
- [ ] Application can connect to different blockchain networks (testnet/regtest)
- [ ] All existing functionality remains intact (backward compatibility)
- [ ] Comprehensive test coverage for new features

---

## Appendix: Example Config Snippets

### Provider with Time-Based Pricing (per minute)

```json
{
  "provider": {
    "pricing_method": "time",
    "price": 50,
    "billing_time_unit": "minute",
    "session_timeout_sec": 3600,
    "price_sats_per_session": 0  // ignored when pricing_method != "session"
  }
}
```

### Provider with Data-Based Pricing (per GB)

```json
{
  "provider": {
    "pricing_method": "data",
    "price": 10000,
    "billing_data_unit": "GB",
    "bandwidth_limit": "100mbit"
  }
}
```

### Client with Spending Limits

```json
{
  "client": {
    "spending_limit_enabled": true,
    "spending_limit_sats": 10000,
    "spending_warning_percent": 80,
    "auto_disconnect_on_limit": true,
    "auto_recharge_enabled": false
  }
}
```

### RPC for Different Network

```json
{
  "rpc": {
    "host": "localhost:18443",
    "user": "rpcuser",
    "pass": "rpcpass",
    "network": "testnet",
    "token_symbol": "ORDEX-TEST"
  }
}
```

---

*Document version: 1.0*  
*Last updated: 2026-03-11*
