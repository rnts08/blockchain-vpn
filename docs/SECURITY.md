# Security Model

This document summarizes trust boundaries and core security controls in BlockchainVPN.

## Trust Boundaries

- Blockchain/RPC boundary: provider discovery, payment settlement, authorization signals.
- Provider runtime boundary: local host networking, TLS listener, payment monitor.
- Client runtime boundary: local routing/DNS controls, TLS identity checks.
- Local secret boundary: provider private key material (`file` or secure-store backend).

## Identity and Transport

- TLS-over-TUN is used for transport.
- Peer identity is bound to a secp256k1 public key embedded in certificate extension.
- Client verifies provider key against on-chain endpoint identity.
- Provider verifies client certificate identity key against payment authorization + access policy.

## Key Material Protection

- File mode: encrypted private key with password-derived AES-GCM.
- Optional secure-store backends:
  - macOS Keychain
  - Linux libsecret
  - Windows DPAPI

## Access Control

- On-chain payment authorization controls session admission.
- Optional provider allowlist/denylist adds operator policy control.
- Optional revocation cache can immediately reject revoked cert identity keys.

## Networking Safety Controls

- Route/DNS setup/restore automation with per-OS backends.
- Optional kill switch to reduce traffic leakage when tunnel drops.
- Crash-safe cleanup markers restore pending route/DNS state on restart.

## Observability and Exposure

- Runtime status available via `bcvpn status --json`.
- Optional metrics endpoint (`/metrics.json`), with optional token auth.
- Recommended: bind metrics to loopback unless protected by host firewall/proxy.

## Threat Notes

- Compromised local host can still exfiltrate in-memory secrets or traffic.
- Misconfigured RPC endpoint or exposed wallet node weakens payment/auth assumptions.
- Metrics endpoints without loopback/auth can leak operational metadata.
