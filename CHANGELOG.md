# Changelog

All notable changes to this project will be documented in this file.

## [0.8.1] - 2026-03-20

### Refactoring: Blockchain-Agnostic Public API

The blockchain package public API has been refactored to eliminate Bitcoin/btcd-specific type leakage, making the code more portable to any UTXO-based blockchain.

#### Internal Types Removed from Public API
- Replaced `btcutil.Amount` with `uint64` throughout (blockchain amounts are always base units)
- Replaced `btcutil.Address` with `string` in function signatures (`SendPayment`, `GetProviderPaymentAddress`)
- Replaced `*chainhash.Hash` with `string` for transaction hashes
- Removed `*chaincfg.Params` from `GetProviderPaymentAddress` (no longer needed)
- `FeeConfig` struct replaces ad-hoc fee parameter passing

#### New Configuration Fields (RPCConfig)
- `CookieDir` — blockchain data directory for cookie auto-detection (default: `.ordexcoin`)
- `CookieDirRegTx` — subdirectory for regtest cookie (default: `regtest`)
- `CookieDirTest3` — subdirectory for testnet3 cookie (default: `testnet3`)
- `CookieDirSignet` — subdirectory for signet cookie (default: `signet`)
- `MinRelayFee` — minimum relay fee in base units (default: 1000)
- `DefaultFeeKb` — default fee per KB when estimation fails (default: 1000)

#### Fee Calculation
- Fee estimation (`estimateDynamicFeePerKb`) now returns `uint64` (sats/KB) instead of `btcutil.Amount`
- All announcement and payment functions now accept a `FeeConfig` parameter for configurable fee floors
- Fee rate conversion: `BTC_per_KB * 1e5 = sats/KB` (correct for Bitcoin Core RPC `estimatesmartfee` output)

#### Remaining btcd Dependencies
All remaining btcd imports (`wire`, `txscript`, `btcjson`, `rpcclient`, `btcec`, `chainhash`) are now confined to the internal implementation layer. The public API of `internal/blockchain` uses only plain Go types, making future blockchain support straightforward.

#### Updated Function Signatures
- `AnnounceService(client, endpoint, feeTarget, feeMode, addressType, feeConfig)`
- `AnnounceHeartbeat(client, pubKey, flags, addressType, feeConfig)`
- `AnnouncePriceUpdate(client, pubKey, price, feeTarget, feeMode, addressType, feeConfig)`
- `AnnounceRating(client, providerPubKey, clientPrivKey, score, source, feeTarget, feeMode, addressType, feeConfig)`
- `SendPayment(client, providerAddress, amount, clientPubKey, addressType, feeConfig)` — providerAddress is now `string`
- `GetProviderPaymentAddress(client, txid)` — returns `string`, no longer needs chainParams
- `WaitForConfirmations(client, txHash, confirmations, interval)` — txHash is now `string`
- `NewSpendingManager(cfg, client, providerAddr, localKey, providerPubKey, addressType, feeConfig)` — providerAddr is now `string`

## [0.8.0] - 2026-03-20

### Breaking Changes
- **Blockchain RPC**: All blockchain announce/send functions now require an `addressType` parameter
- **Breaking change**: `AnnounceService`, `AnnounceHeartbeat`, `AnnouncePriceUpdate`, `AnnounceRating`, and `SendPayment` signatures updated

### Feature: Blockchain-Agnostic Support
- Modified `detectChain()` to gracefully handle unknown blockchain genesis hashes
- Added `paramsFromNetwork()` helper function to resolve chain params from network config
- Updated `GetProviderPaymentAddress()` to handle nil chainParams for unknown blockchains
- Added `stringAddress` type implementing `btcutil.Address` interface for unknown chains
- `bcvpn scan` now uses configured `rpc.network` value as fallback when genesis hash is unrecognized
- Enables support for custom Bitcoin-like blockchains (e.g., OrdexCoin)

### Feature: Auto-Detect Wallet Address Type
- Added `DetectAddressType()` function that probes UTXOs to determine wallet address type
- Added `AddressType` field to both `ProviderConfig` and `ClientConfig` ("auto", "p2pkh", "p2sh", "bech32", "bech32m")
- Auto-detects address type from existing UTXOs via scriptPubKey analysis (76a914...88ac = P2PKH, a914...87 = P2SH, 0014... = bech32, 0020... = bech32m)
- Also inspects UTXO address string prefix ('o' = P2PKH for OrdexCoin)
- Falls back to probing with `getrawchangeaddress` then `getnewaddress`
- Fixes "unknown address type ''" error on OrdexCoin and other non-bech32 chains

### Feature: Automatic Reconnection
- Added automatic reconnection on network disconnect
- New CLI flags: `--auto-reconnect`, `--auto-reconnect-max-attempts`, `--auto-reconnect-interval`, `--auto-reconnect-max-interval`
- Added `AddWithReconnect` method to `MultiTunnelManager`
- Implemented exponential backoff for reconnection attempts

### Feature: Provider Bandwidth Auto-Detection
- Implemented `MeasureLocalBandwidthKbps` function for self-contained TCP loopback bandwidth testing
- Wired up `BandwidthAutoTest` config field to run speed test at provider startup

### Fix: Scanner Performance and Defaults
- Changed default `--startblock` to -1000 (last 1000 blocks from tip)
- Added support for negative startblock values (relative to tip)
- Removed verbose log spam for non-VPN transactions during scan

### Fix: Fee Target Clamping
- Added `clampFeeTarget()` to ensure fee target is always between 1-1008
- Defaults to 6 blocks when target is 0 or invalid
- Fixes "estimateSmartFee error -8: invalid config_target" error

### Fix: Minimum Fee Fallback for New Chains
- `estimateDynamicFeePerKbWithMode` now returns a minimum feerate (1000 sats/KB) when fee estimation fails
- Fallback is triggered when `EstimateSmartFee` and `GetNetworkInfo` return no valid feerate
- Handles fresh chains with no fee history (e.g., OrdexCoin at genesis)

### Fix: Provider Key Password Masking
- Provider key password input now masked using terminal echo suppression
- Password characters not echoed to terminal during input (uses `golang.org/x/term`)

### Fix: NAT Traversal Timeout
- Added 10-second timeout to NAT traversal (UPnP/NAT-PMP discovery)
- Prevents indefinite hang when no NAT/router is available or UPnP is disabled

### Fix: Use RawRequest for sendrawtransaction
- Call `sendrawtransaction` via `RawRequest` with minimal params `[hex]`
- Bypasses btcd/rpcclient `SendRawTransaction` which sends incompatible parameters
- OrdexCoin's modified RPC rejects the standard `[hex, allowHighFees]` signature

### Fix: Use UTXO ScriptPubKey for Change Output
- Change output now uses the first UTXO's `scriptPubKey` directly instead of re-encoding via `PayToAddrScript`
- Added `selectCoinsForTx()` helper that returns UTXOs, total, and change script in one call
- Bypasses address encoding/decoding entirely — works for any blockchain
- Fixes "error creating change script" on unknown chain address formats

### Fix: Skip Initial Heartbeat
- Removed immediate heartbeat broadcast on provider startup
- Heartbeats now start at the configured interval after the service announcement
- Prevents "txn-mempool-conflict" errors when heartbeat competes with announcement for same UTXO

### Fix: Graceful Shutdown
- Both provider and scan commands now handle `SIGTERM` in addition to `SIGINT`
- Client tunnel shutdown now has a 10-second timeout to prevent indefinite hang
- Added `RecoverPendingNetworkStateAndCleanupStaleInterfaces()` to clean up orphaned TUN interfaces on startup
- Cleans up `bcvpn*` interfaces from crashed sessions via `netlink.LinkDel`

### Test Coverage
- Added unit tests for `sendRawTransaction` with mock HTTP RPC server
- Added unit tests for `clampFeeTarget` (9 test cases)
- Added unit tests for script class detection (P2PKH, P2SH, P2WPKH, P2WSH, null data)
- Added unit tests for NAT timeout behavior

---

## [0.7.5] - 2026-03-20

### Fix: Minimum Fee Fallback for New Chains
- `estimateDynamicFeePerKbWithMode` now returns a minimum feerate (1000 sats/KB) when fee estimation fails
- Fallback is triggered when `EstimateSmartFee` and `GetNetworkInfo` return no valid feerate
- Handles fresh chains with no fee history (e.g., OrdexCoin at genesis)
- Heartbeat and other announcements no longer fail with 0-fee transactions

---

## [0.7.11] - 2026-03-20

### Fix: Use RawRequest for sendrawtransaction
- Call `sendrawtransaction` via `RawRequest` with minimal params `[hex]`
- Bypasses btcd/rpcclient `SendRawTransaction` which sends incompatible parameters
- OrdexCoin's modified RPC rejects the standard `[hex, allowHighFees]` signature
- Works across all blockchain announce/send functions

---

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
