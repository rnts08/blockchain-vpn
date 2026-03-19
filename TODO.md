# BlockchainVPN Implementation Plan

This document tracks the remaining tasks and improvements for the BlockchainVPN project. Items are added as new improvements are identified.

## Priority: High

- [x] Ensure RPC connection works with local ordexcoind (including regtest mode) - mock-rpc server available for testing
- [x] Add `--dry-run` support to provider commands (start-provider, rebroadcast, generate-provider-key)
- [x] Implement full functionality for `disconnect`, `restart-provider`, `stop-provider` commands (PID file, signal handling)
- [x] Test end-to-end client connection flow in dry-run mode - documented in TESTING.md, works with mock-rpc
- [x] Validate configuration with only required fields for active mode (client vs provider)
- [x] Verify scan command with `--min-score`, `--limit`, and `--rescan` flags match OPERATIONS.md

## Priority: Medium

- [x] Implement rating persistence (ratings.json) - stored in config dir as ratings.json
- [x] Add more scanner filters (min-score, limit, rescan) and sort alias bw
- [x] Simplify default config generation to only include essential fields
- [x] Improve error handling and user feedback for CLI commands (ongoing)
- [x] Add detailed help subcommands for all major commands (generate-send-address, favorite, rate, etc.) - comprehensive help added
- [x] Document demo/testing workflow with regtest ordexcoind - see TESTING.md

## Issues Found During Code Review (Beta Testing Prep)

### Critical - Architecture Gaps (FIXED)

- [x] `SpendingManager.Start()` never called - auto-recharge non-functional
- [x] `UsageMeter` not integrated - payment renewal non-functional
- [x] `ConnectToProvider` didn't use spendingMgr - no ongoing payment monitoring
- [x] No traffic tracking for time/data-based billing renewal

### Critical - Bug Fixes

- [x] `EncodePaymentPayload` in `internal/protocol/vpn_protocol.go:476` - panics if `clientPubKey` is nil (missing nil check, unlike other similar functions)
- [x] `GetConfigField`/`SetConfigField` in `internal/config/config_registry.go:87,96` - missing nil check for `cfg` parameter
- [x] `DecodeReputationPayload` in `internal/protocol/reputation.go:86` - potential out-of-bounds read when reading signature length
- [x] `WriteFileAtomic` in `internal/util/atomic.go` - data loss on rename failure (defer removed temp file before rename could fail)
- [x] Session timer race in `internal/tunnel/tunnel.go:398` - goroutine not synchronized after timer expiration

### Medium

- [x] `EncodeCertFingerprintPayload` in `internal/protocol/vpn_protocol.go:622` - silently zeroes truncated fingerprints instead of returning error
- [x] NAT-PMP goroutine in `internal/nat/nat.go:136` - may send on channel after context cancellation
- [x] Unsafe type assertion in `internal/config/config_registry.go:139` - `.([]string)` could panic
- [x] GeoIP lookup error silently ignored in `internal/geoip/enrich.go:49`

### Low

- [x] `CheckPortAvailable` in `internal/config/port.go:43` - use defer for robustness
- [x] `containsAt` test helper in `internal/crypto/crypto_test.go:106` - use strings.Contains
- [ ] `runPowerShell` in `internal/crypto/keystore.go:360` - inherits full process environment (intentional for PowerShell)
- [ ] Cleanup errors silently ignored in `internal/nat/nat.go:67,107` (acceptable pattern)

---

## Priority: Low

- [ ] Review and optimize test coverage gaps
- [ ] Add more integration tests for edge cases
- [ ] Performance optimization for tunnel establishment
- [ ] Add benchmarks for critical paths
- [ ] Consider UI enhancements (though CLI only)
- [ ] Explore multi-chain support beyond OrdexCoin


