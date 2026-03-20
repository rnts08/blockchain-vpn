# BlockchainVPN Implementation Plan

This document tracks the remaining tasks and improvements for the BlockchainVPN project. Items are added as new improvements are identified.

## Priority: High

- [x] No automatic reconnection on network disconnect (usability issue, not security)
- [ ] Heartbeat announcements lack cryptographic signature (can be added post-beta)
- [ ] Reputation scores not signature-verified (trust through network effect)
- [ ] Direct connect command is stub-only (must use scan to connect)

## Priority: Medium

- [ ] **Provider bandwidth auto-detection** - Measure actual upload/download bandwidth at startup and advertise accurate speed (currently manual config required)
- [ ] Refund flow integration - client can disconnect and request refund when quality < 75%
- [ ] NAT traversal method selection via config
- [ ] WebSocket origin validation
- [ ] Symmetric NAT detection and handling
- [ ] STUN integration for NAT type detection

### Low

- [x] Test coverage improvements completed (0.7.3)
- [x] Refund flow integration tests added
- [x] Heartbeat signature verification tests added
- [x] Benchmark tests for critical paths added
- [ ] `runPowerShell` in `internal/crypto/keystore.go:360` - inherits full process environment (intentional for PowerShell)
- [ ] Cleanup errors silently ignored in `internal/nat/nat.go:67,107` (acceptable pattern)
- [ ] Performance optimization for tunnel establishment
- [ ] Consider UI enhancements (though CLI only)
- [ ] Explore multi-chain support beyond OrdexCoin


---

### Security Considerations (Acceptable for PoC)

- TLS InsecureSkipVerify with custom certificate verification (by design for P2P auth)
- Authorization extension proportional to payment amount (verified at payment processing)
- Latency echo server unauthenticated (requires VPN port connection first)

---
