# Provider Lifecycle Management

This document describes how providers are discovered, managed, and their lifecycle within BlockchainVPN.

## Overview

The provider lifecycle encompasses:
1. Announcement on blockchain
2. Discovery by clients
3. Connection establishment
4. Service delivery
5. Session termination

## Core Components

### Provider Discovery

Providers announce their services on the OrdexCoin blockchain using OP_RETURN transactions with VPN endpoint data.

**Announcement Format:**
- Magic bytes: `0x56504E03` (V3)
- IP address (IPv4/IPv6)
- Port number
- Price in satoshis
- Public key (secp256k1)
- Pricing method (session/time/data)
- Time/data unit specifications

### Scanner (`internal/blockchain/scanner.go`)

Continuously scans blockchain for provider announcements.

**Functions:**
- `StartScanner()`: Begins block scanning
- `StopScanner()`: Halts scanning
- `GetProviders()`: Returns list of discovered providers

### Provider Store (`internal/blockchain/provider.go`)

Maintains provider information and reputation.

**Key Functions:**
- `GetProvider()`: Retrieves provider by public key
- `UpdateProvider()`: Updates provider metadata
- `GetProvidersByRegion()`: Filters providers by location

## Lifecycle Stages

### 1. Announcement

Provider broadcasts announcement transaction:
```
OP_RETURN <VPN_ENDPOINT_DATA>
```

Client software monitors blockchain and decodes announcements.

### 2. Discovery

Client scanner detects new announcements:
1. Fetch transaction details
2. Decode OP_RETURN payload
3. Verify provider public key
4. Add to provider list
5. Start reputation tracking

### 3. Selection

Client selects provider based on:
- Price
- Bandwidth (from throughput probe)
- Latency (from probe)
- Reputation score
- Geographic location
- Availability

### 4. Connection

Client initiates connection:
1. Parse provider endpoint
2. Establish TLS connection
3. Send payment
4. Provider verifies payment
5. Configure TUN interface
6. Begin packet forwarding

### 5. Service

Provider delivers VPN service:
- Route client traffic
- Apply NAT
- Track usage (time/data)
- Request periodic payments

### 6. Termination

Session ends via:
- Client disconnect
- Provider shutdown
- Payment failure
- Network error
- Timeout

## Reputation System

Providers build reputation over time:

**Factors:**
- Uptime percentage
- Payment success rate
- Average bandwidth delivered
- Latency measurements
- User ratings

**Storage:** `internal/blockchain/reputation_store.go`

## Payment Verification

Providers verify payments on-chain:

1. Monitor mempool for payments to their address
2. Decode OP_RETURN to extract client public key
3. Verify payment amount meets price
4. Grant/deny service access

## Health Monitoring

Providers report health status:

- `ProviderHealth` struct tracks:
  - Current connections
  - Available bandwidth
  - Uptime
  - Last heartbeat

Clients can query health before connecting.

## Configuration

Provider settings in `config.go`:
- `Price`: Service price in satoshis
- `PricingMethod`: 0=session, 1=time, 2=data
- `TimeUnitSecs`: Billing interval for time-based
- `DataUnitBytes`: Billing interval for data-based
- `MaxConsumers`: Maximum concurrent clients
- `SessionTimeoutSecs`: Max session duration
