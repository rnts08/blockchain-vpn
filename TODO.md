# BlockchainVPN Implementation Plan

This document tracks remaining tasks and improvements for BlockchainVPN. Items are organized by priority and category.

## Priority: High

- [ ] Heartbeat announcements lack cryptographic signature (can be added post-beta)
- [ ] Reputation scores not signature-verified (trust through network effect)
- [ ] Direct connect command is stub-only (must use scan to connect)
- [ ] Refund flow not implemented

## Priority: Medium

- [ ] NAT traversal method selection via config (currently auto only)
- [ ] WebSocket origin validation
- [ ] Symmetric NAT detection and handling
- [ ] STUN integration for NAT type detection

### Low

- [ ] `runPowerShell` in `internal/crypto/keystore.go:360` - inherits full process environment (intentional for PowerShell)
- [ ] Cleanup errors silently ignored in `internal/nat/nat.go:67,107` (acceptable pattern)
- [ ] Performance optimization for tunnel establishment
- [ ] GeoIP database bundling or auto-download (currently manual MaxMind download)
- [ ] Configurable GeoIP database path (hardcoded to `GeoLite2-Country.mmdb`)

---

## Completed Features (Post-Beta)

The following items were identified as post-beta limitations but are now fully implemented:

- [x] Automatic reconnection on network disconnect with exponential backoff
- [x] Provider bandwidth auto-detection at startup
- [x] Blockchain-agnostic RPC support (unknown chains, flexible fee estimation)
- [x] Wallet address type auto-detection
- [x] NAT traversal timeout (10s)
- [x] Provider key password masking
- [x] Graceful shutdown (SIGTERM/SIGINT) with cleanup (TUN, NAT, routes, DNS)
- [x] Blockchain-agnostic public API (plain Go types, no btcd leakage)
- [x] Configurable RPC cookie directories per network
- [x] Minimum relay fee and default fee per KB configuration

---

## Testing Requirements

**Important**: All new features and changes must include corresponding unit tests. Existing code undergoing modification should have test coverage improved where feasible.

- Test coverage for blockchain-agnostic public API (completed v0.8.1)
- Test coverage for fee estimation edge cases (completed v0.8.0)
- Test coverage for address type detection (completed v0.8.0)
- Test coverage for NAT timeout handling (completed v0.8.0)

---

## Security Considerations (Acceptable for PoC)

- TLS InsecureSkipVerify with custom certificate verification (by design for P2P auth)
- Authorization extension proportional to payment amount (verified at payment processing)
- Latency echo server unauthenticated (requires VPN port connection first)

---

## Documentation Notes

- Always update README.md, CHANGELOG.md, and AGENTS.md when adding features or changing workflows.
- Keep the feature checklist in README.md in sync with actual implementation status.
- When adding new configuration fields, update the sample config in README.md with sensible defaults.
