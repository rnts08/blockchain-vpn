# BlockchainVPN User Guide

## 1. Overview

BlockchainVPN supports two operating modes:

- Provider mode: announce and sell VPN sessions.
- Client mode: discover providers, pay, and connect.

This guide covers end-to-end flows for both CLI and GUI.

## 2. First Run

1. Start GUI (`bcvpn-gui`) or CLI (`bcvpn`).
2. Generate config if needed:
   - `./bcvpn generate-config`
3. Configure RPC access to `ordexcoind`.
4. Ensure elevated privileges are available (admin/root).

GUI users are guided by the first-run setup wizard.

## 3. Provider Flow

### CLI

1. Set provider settings:
   - `./bcvpn config set provider.listen_port 51820`
   - `./bcvpn config set provider.price_sats_per_session 1000`
   - `./bcvpn config set provider.max_consumers 10`
2. Start provider:
   - `./bcvpn start-provider`
3. Optional runtime actions:
   - `./bcvpn update-price --price 1500`
   - `./bcvpn rebroadcast`
4. Rotate provider identity key:
   - `./bcvpn rotate-provider-key`

### GUI

1. Open **Provider Mode**.
2. Set port, price, max consumers, NAT, cert policy, and access policy fields.
3. Save config and enter provider key password.
4. Click **Start Provider**.

## 4. Client Flow

### CLI

1. Scan:
   - `./bcvpn scan --sort=score --country=US --max-price=2000 --min-bandwidth-kbps=25000 --max-latency-ms=80 --min-available-slots=2`
2. Select provider in prompt.
3. Confirm payment.
4. Tunnel activates and route/DNS are auto-configured.
5. Post-connect security checks run automatically (egress IP transition, DNS checks, country match when available).

### GUI

1. Open **Client Mode**.
2. Scan providers.
3. Select provider and click **Connect Selected**.
4. Optionally enable **Kill Switch** before connecting.

## 5. Status and Validation

- Runtime status:
  - `./bcvpn status`
  - `./bcvpn status --json`
- Config validation:
  - `./bcvpn config validate`

## 6. Common Operations

- Show full config:
  - `./bcvpn config get`
- Update single key:
  - `./bcvpn config set client.enable_kill_switch true`
- Strict verification mode:
  - `./bcvpn config set client.strict_verification true`
- Profile export/import:
  - `./bcvpn config export ./profile.json`
  - `./bcvpn config import ./profile.json --validate`
- View payment history:
  - `./bcvpn history`
- View runtime event timeline:
  - `./bcvpn events --limit=100`
- Export diagnostics bundle:
  - `./bcvpn diagnostics`

## 7. Safety Notes

- Use strong provider key passwords.
- Keep `provider.key` protected.
- Use allowlist/denylist and sandbox mode where possible.
- Verify privilege readiness in status output before connecting.
