# Changelog

All notable changes to this project will be documented in this file.

## [0.7.9] - 2026-03-20

### Fix: Provider Key Password Masking
- Provider key password input now masked using terminal echo suppression
- Password characters not echoed to terminal during input

### Fix: NAT Traversal Timeout
- Added 10-second timeout to NAT traversal (UPnP/NAT-PMP discovery)
- Prevents indefinite hang when no NAT/router is available or UPnP is disabled

### Fix: Improved Address Type Detection
- Enhanced `DetectAddressType()` to also check UTXO address string prefix
- Added `GetNewAddress` probing as additional fallback method
- Added "legacy" as candidate address type
- Handles OrdexCoin addresses starting with 'o' correctly

---

## [0.7.8] - 2026-03-20

### Fix: Auto-Detect Wallet Address Type
- Added `DetectAddressType()` function that probes UTXOs to determine wallet address type
- Added `AddressType` field to both `ProviderConfig` and `ClientConfig` ("auto", "p2pkh", "p2sh", "bech32", "bech32m")
- Auto-detects address type from existing UTXOs via scriptPubKey analysis, falls back to probing with getrawchangeaddress
- All wallet operations now use detected/configured address type instead of empty string
- Fixes "unknown address type ''" error on OrdexCoin and other non-bech32 chains

### Fix: Fee Target Clamping
- Added `clampFeeTarget()` to ensure fee target is always between 1-1008
- Defaults to 6 blocks when target is 0 or invalid
- Fixes "estimateSmartFee error -8: invalid config_target" error

### Affected Functions
- `AnnounceService`, `AnnounceHeartbeat`, `AnnouncePriceUpdate`, `AnnounceRating` now accept addressType
- `SendPayment` now accepts addressType
- `GetNewAddress` and `GetRawChangeAddress` now use detected address type
- `NewSpendingManager` now accepts addressType

---

## [0.7.7] - 2026-03-20

### Fix: Scanner Performance and Defaults
- Changed default `--startblock` to -1000 (last 1000 blocks from tip)
- Added support for negative startblock values (relative to tip)
- Removed verbose log spam for non-VPN transactions during scan

---

## [0.7.6] - 2026-03-20

### Feature: Blockchain-Agnostic Support
- Modified `detectChain()` to gracefully handle unknown blockchain genesis hashes
- Added `paramsFromNetwork()` helper function to resolve chain params from network config
- Updated `GetProviderPaymentAddress()` to handle nil chainParams for unknown blockchains
- Added `stringAddress` type implementing `btcutil.Address` interface for unknown chains
- `bcvpn scan` now uses configured `rpc.network` value as fallback when genesis hash is unrecognized
- Enables support for custom Bitcoin-like blockchains (e.g., OrdexCoin)

---

## [0.7.5] - 2026-03-20

### Feature: Provider Bandwidth Auto-Detection
- Implemented `MeasureLocalBandwidthKbps` function for self-contained TCP loopback bandwidth testing
- Wired up `BandwidthAutoTest` config field to run speed test at provider startup
- Updated `buildProviderEndpoint` to accept measured bandwidth parameter
- Added context cancellation support for graceful shutdown
- Added unit tests for bandwidth measurement

---

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
