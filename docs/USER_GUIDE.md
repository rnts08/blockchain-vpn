# BlockchainVPN User Guide

## 1. Overview

BlockchainVPN supports two operating modes:

- Provider mode: announce and sell VPN sessions.
- Client mode: discover providers, pay, and connect.

This guide covers end-to-end CLI flows for both modes.

**Note:** GUI support has been archived. The CLI provides full functionality.

## 2. First Run

### Option A: Guided Setup (Recommended for New Users)

1. Build the CLI:
   ```bash
   go build -o bcvpn ./cmd/bcvpn
   ```
2. Run the interactive setup wizard:
   ```bash
   ./bcvpn setup
   ```
   This will guide you through RPC, logging, security, and provider/client configuration with sensible defaults.

### Option B: Manual Configuration

1. Build the CLI:
   ```bash
   go build -o bcvpn ./cmd/bcvpn
   ```
2. Generate config:
   ```bash
   ./bcvpn generate-config
   ```
3. Configure RPC access to `ordexcoind`:
   ```bash
   ./bcvpn config set rpc.host localhost:25173
   ./bcvpn config set rpc.user youruser
   ./bcvpn config set rpc.pass yourpass
   ```
4. Ensure elevated privileges are available (admin/root).

## 3. Provider Flow

1. Set provider settings:
   ```bash
   ./bcvpn config set provider.listen_port 51820
   ./bcvpn config set provider.price_sats_per_session 1000
   ./bcvpn config set provider.max_consumers 10
   ```
2. Start provider:
   ```bash
   ./bcvpn start-provider
   ```
3. Optional runtime actions:
   ```bash
   ./bcvpn rebroadcast
   ./bcvpn stop-provider && ./bcvpn start-provider
   ```
4. Rotate provider identity key:
   ```bash
   ./bcvpn rotate-provider-key
   ```

## 4. Client Flow

1. Scan for providers:
   ```bash
   ./bcvpn scan --sort=score --country=US --max-price=2000 --min-bandwidth-kbps=25000 --max-latency-ms=80 --min-available-slots=2
   ```
2. Select provider in interactive prompt.
3. Confirm payment.
4. Tunnel activates and route/DNS are auto-configured.
5. Post-connect security checks run automatically (egress IP transition, DNS checks, country match when available).
6. On disconnect, you'll be prompted to rate the provider (1-5 stars).

## 5. Status and Validation

- Runtime status:
  ```bash
  ./bcvpn status
  ./bcvpn status --json
  ```
- Config validation:
  ```bash
  ./bcvpn config validate
  ```

## 6. Common Operations

- Show full config:
  ```bash
  ./bcvpn config get
  ```
- Update single key:
  ```bash
  ./bcvpn config set client.enable_kill_switch true
  ```
- Strict verification mode:
  ```bash
  ./bcvpn config set client.strict_verification true
  ```
- Profile export/import:
  ```bash
  ./bcvpn config export ./profile.json
  ./bcvpn config import ./profile.json --validate
  ```
- View payment history:
  ```bash
  ./bcvpn history
  ```
- View runtime event timeline:
  ```bash
  ./bcvpn events --limit=100
  ```
- Export diagnostics bundle:
  ```bash
  ./bcvpn diagnostics
  ```

## 7. Safety Notes

- Use strong provider key passwords.
- Keep `provider.key` protected.
- Use allowlist/denylist and sandbox mode where possible.
- Verify privilege readiness in status output before connecting.
