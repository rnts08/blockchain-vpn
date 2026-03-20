# Changelog

All notable changes to this project will be documented in this file.

## [0.7.4] - 2026-03-20

### Feature: Automatic Reconnection
- Added automatic reconnection on network disconnect
- New CLI flags: `--auto-reconnect`, `--auto-reconnect-max-attempts`, `--auto-reconnect-interval`, `--auto-reconnect-max-interval`
- Added `AddWithReconnect` method to `MultiTunnelManager`
- Added `ReconnectConfig` and `tunnelParams` types for managing reconnection state
- Implemented exponential backoff for reconnection attempts
- Configuration fields added: `AutoReconnectEnabled`, `AutoReconnectMaxAttempts`, `AutoReconnectInterval`, `AutoReconnectMaxInterval`

### Unit Tests
- Added `TestParseAutoReconnectInterval` test
- Added `TestMultiTunnelManager_ReconnectInfoStored` test
- Added `TestMultiTunnelManager_CancelClearsReconnectInfo` test

---

## [0.7.3] - 2026-03-20

### Test Coverage Improvements
- Fixed failing `TestSignWithSecp256k1_Randomized` test by adding signature verification
- Added `verifyASN1Signature` function for ASN.1 encoded ECDSA signature verification
- Improved `internal/transport` package tests (0% → 6.1% coverage)
- Improved `internal/version` package tests (0% → 100% coverage)
- Improved `internal/geoip` package tests (18.2% → 29.1% coverage)
- Improved `internal/history` package tests (15.2% → ~20% coverage)
- Improved `internal/nat` package tests (15.4% → ~25% coverage)
- Improved `internal/blockchain` package tests (22.7% → 23.1% coverage)
- Improved `internal/tunnel` package tests (33.9% → 34.1% coverage)

### Integration Tests
- Added refund flow integration tests for connection quality checks
- Added heartbeat signature verification integration tests
- Added benchmark tests for transport package

### Bug Fixes
- Fixed `TestRunClientPostConnectChecks_StrictFailsOnCountryMismatch` test
- Fixed `TestCheckConnectionQuality_QualityScoreClamped` test
- Fixed `multi_tunnel_functional_test.go` to use correct function signatures

### Refund Flow
- Added `TestFunctional_RefundFlow_LowQualityDisconnection` test
- Added `TestFunctional_RefundFlow_HighQualityNoRefund` test
- Added `TestFunctional_RefundFlow_QualityThresholdEdgeCases` test
- Added `TestFunctional_RefundFlow_DNSCausesRefund` test

### Heartbeat Signature Verification
- Added `TestFunctional_HeartbeatSignatureVerification` test
- Added `TestFunctional_HeartbeatSignatureDifferentPubKeyFails` test
- Added `TestFunctional_HeartbeatSignatureTamperedFails` test
- Added `TestFunctional_HeartbeatSignatureEmptyMessageFails` test
- Added `TestFunctional_VerifyASN1SignatureInvalidASN1` test

---

## [0.7.2] - 2026-03-17

### Previous Release
- Previous patch release
