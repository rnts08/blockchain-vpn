# Changelog

All notable changes to this project will be documented in this file.

## [0.7.1] - 2026-03-19

### Bug Fixes
- Fixed session spending limits ineffective - SetSessionStart() now called after payment
- Session limits will now properly enforce per-session spending caps

### Version Bump
- Bumped patch version to 0.7.1.

## [0.7.0] - 2026-03-19

### Critical Bug Fixes (Pre-Beta Release)
- Fixed connection leak on WebSocket fallback - original connection now closed before retry
- Fixed TOCTOU race condition in readTunLoop - session pointer used under lock
- Fixed nil pointer panic in RPC client creation - added nil check for client
- Added recover() to handleClient goroutines - prevents crashes from panics
- Fixed goroutine leaks in provider server - TUN reader and WS server now tracked with WaitGroup
- Added graceful shutdown for background goroutines

### Version Bump
- Bumped minor version to 0.7.0 for beta release.

## [0.6.16] - 2026-03-19

### Rating System
- Updated scan output to display on-chain accumulated rating (ReputationScore)
- Rating shown in scan output when provider has blockchain ratings

### Version Bump
- Bumped patch version to 0.6.16.

## [0.6.15] - 2026-03-19

### Rating System
- Added `sessionInfo` struct with local storage for session persistence
- Added rating prompt on disconnect (1-5 stars or skip)
- Integrated `AnnounceRating` into disconnect flow when RPC configured
- Added client key generation/loading for signing ratings
- Rating saved locally in ratings.json and broadcast to blockchain if RPC configured

### Version Bump
- Bumped patch version to 0.6.15.

## [0.6.14] - 2026-03-19

### Quality Assurance & Trust Features
- Added `ConnectionQuality` struct for detailed quality reporting
- Added `CheckConnectionQuality()` function for comprehensive connection verification
- Changed bandwidth threshold from 50% to 75% (as per P2P trust requirements)
- Quality checks now cover: egress IP, DNS leak, country, bandwidth
- Quality score (0-100%) with warnings logged for failed checks
- Strict verification mode fails on any quality warning

### Version Bump
- Bumped patch version to 0.6.14.

## [0.6.13] - 2026-03-19

### Provider & Rating Improvements
- Added auto country detection for providers when config country is empty (uses ip-api.com)
- Added `AnnounceRating` function for blockchain reputation broadcast
- Added `--broadcast` flag to `rate` command (TODO: full blockchain integration)
- Added `EncodeReputationPayloadWithoutSignature` for signing ratings

### Version Bump
- Bumped patch version to 0.6.13.

## [0.6.12] - 2026-03-19

### Critical Security & Architecture Fixes
- Added confirmation requirement in provider payment processing - providers now wait for minimum confirmations before authorizing clients (configurable via `payment_required_confirmations`)
- Fixed data quota enforcement - provider now consumes data quota during sessions and disconnects clients when quota is exhausted
- Added `WaitForConfirmations` function and integrated into client flow - client now waits for payment confirmation before connecting

### Version Bump
- Bumped patch version to 0.6.12.

## [0.6.11] - 2026-03-19

### Architecture Fixes - Auto-Settlement System
- Added `PricingParams` struct and `NewPricingParamsFromEndpoint()` for time/data-based billing
- Integrated `UsageMeter` in `ConnectToProvider` to track traffic for billing renewal
- Added payment renewal goroutine that checks `ShouldRenewPayment()` every 5 seconds
- Added `spendingMgr.ShouldDisconnect()` check to disconnect when spending limit reached
- Added `spendingMgr.Start()` call after payment confirmation for auto-recharge support
- Traffic callbacks now call `usageMeter.AddTraffic()` to track bytes for data-based billing

### Version Bump
- Bumped patch version to 0.6.11.

## [0.6.10] - 2026-03-19

### Bug Fixes
- Fixed data loss in `WriteFileAtomic` - now only removes temp file on error after rename fails
- Fixed session timer race condition in tunnel - now waits for goroutine to finish after timer expiration
- Added logging for GeoIP lookup errors instead of silently ignoring them

### Version Bump
- Bumped patch version to 0.6.10.

## [0.6.9] - 2026-03-19

### Bug Fixes
- `EncodeCertFingerprintPayload` now returns error for truncated fingerprints instead of silently zeroing
- NAT-PMP goroutine now checks context cancellation before sending result to prevent potential leaks
- `CheckPortAvailable` uses defer for robust resource cleanup
- Replaced custom `containsAt` test helper with `strings.Contains`

### Version Bump
- Bumped patch version to 0.6.9.

## [0.6.8] - 2026-03-19

### Security Fixes
- Added nil check in `EncodePaymentPayload` to prevent panic on nil public key
- Added nil check in `GetConfigField`/`SetConfigField` to prevent panic on nil config
- Fixed bounds check in `DecodeReputationPayload` to prevent potential out-of-bounds read
- Added error return for unsupported slice types in config registry

### Version Bump
- Bumped patch version to 0.6.8.

## [0.6.7] - 2026-03-19

### Testing
- Added comprehensive unit tests for CLI filtering and sorting functions
- Added benchmarks for critical path operations (filter, sort, score computation)
- Fixed test for effectiveCountry to handle empty declared country

### Version Bump
- Bumped patch version to 0.6.7.

## [0.6.6] - 2026-03-19

### CLI
- Added `-a`/`--about` flag for quick access to about info
- Added `broadcast` as alias for `rebroacast` command (as per OPERATIONS.md)
- Added `--from`, `--to` (RFC3339 format), `--json`, and `--table` flags to history command
- Updated main help to include all available detailed help commands

### Version Bump
- Bumped patch version to 0.6.6.

## [0.6.5] - 2026-03-19

### CLI
- Improved error handling and user feedback for CLI commands
- Added helpful hints and guidance in error messages for RPC connection failures
- Added guidance when no VPN providers are found during scan
- Added actionable guidance when no provider or client connection is active
- Improved connect command output with better usage examples

### Version Bump
- Bumped patch version to 0.6.5.

## [0.6.4] - 2026-03-18

### CLI
- Added detailed help functions for all commands (`bcvpn help <command>`)
- Help system now covers all subcommands with consistent formatting
- Improved command documentation and examples

### Infrastructure
- Added comprehensive test coverage for configuration validation
- Code formatting and linting improvements

### Version Bump
- Bumped patch version to 0.6.4.

## [0.6.3] - 2026-03-18

### CLI
- Added `--dry-run` flag to `start-provider`, `rebroadcast`, and `generate-provider-key` commands
- Implemented `disconnect` command using PID file management
- Implemented `stop-provider` and `restart-provider` commands with proper PID file handling
- Added `--min-score`, `--limit` (max 100), and `--rescan` flags to `scan` command
- Added `bw` as an alias for `--sort` (bandwidth)
- Updated help text to match OPERATIONS.md specifications
- Removed obsolete `update-price` command

### Configuration
- Configuration validation now only requires fields for active mode (provider or client)
- Added `pid_file` field to provider configuration (default: config dir/provider.pid)

### Bug Fixes
- Fixed duplicate validation code in internal/config/validate.go

### Version Bump
- Bumped patch version to 0.6.3.


## [0.6.1] - 2026-03-17

### CLI
- Help and version flags now work without requiring config.json
- Added 'bcvpn help config' subcommand for config management help
- Main help now shows 'see bcvpn help config' for config command

### Version Bump
- Bumped patch version to 0.6.1.

## [0.6.0] - 2026-03-17

### Configuration
- Changed default token symbol from ORDEX to OXC
- Simplified default config to only include essential fields
- Changed default config storage directory from ~/.config/BlockchainVPN/ to ~/.config/blockchain-vpn/
- Added detection for legacy config directory variants on startup

### CLI
- Added nice-looking help flag (-h, --help) with usage information
- Added privilege warning for provider mode when running without elevated privileges

### Version Bump
- Bumped minor version to 0.6.0.

## [0.5.21] - 2026-03-14

### Documentation
- Added CONSUMER.md - comprehensive consumer guide with scan, connect, payment, and security features
- Added PROVIDER.md reference in CONSUMER.md for provider details

### Configuration
- Added `favorite_providers` field to client config for saving trusted providers

### Version Bump
- Bumped patch version to 0.5.21.

## [0.5.20] - 2026-03-14

### Configuration
- Added `bandwidth_auto_test` field for automatic speed testing
- Added `nat_traversal_method` field for explicit NAT traversal control ("auto", "upnp", "natpmp", "none")

### Documentation
- Added PROVIDER.md - comprehensive provider guide with setup, configuration, and troubleshooting
- Added config.provider.example.json - example provider configuration

### Version Bump
- Bumped patch version to 0.5.20.

## [0.5.19] - 2026-03-14

### Documentation
- Added CI_CD.md - CI/CD pipeline configuration documentation
- Added MONITORING.md - monitoring system health documentation
- Added CONTRIBUTING.md - contributor onboarding documentation
- Added COMMUNITY.md - community engagement strategies documentation

### Functional Tests
- Added deployment automation functional tests
- Added observability/monitoring functional tests
- Added contributor workflow functional tests
- Added community feedback functional tests

### Version Bump
- Bumped patch version to 0.5.19.

## [0.5.18] - 2026-03-14

### Documentation
- Added BILLING.md - billing system architecture documentation
- Added MULTI_TUNNEL.md - multi-tunnel concurrency documentation
- Added PROVIDER_LIFECYCLE.md - provider lifecycle management documentation
- Added ACCESS_CONTROL.md - access control mechanisms documentation

### Version Bump
- Bumped patch version to 0.5.18.

## [0.5.17] - 2026-03-14

### Functional Tests - Performance
- Added throughput measurement calculation tests
- Added zero duration edge case tests
- Added large payload throughput tests
- Added context timeout handling tests
- Added calculation precision tests

### Version Bump
- Bumped patch version to 0.5.17.

## [0.5.16] - 2026-03-14

### Functional Tests - Tunnel Lifecycle
- Added IP pool allocation and release tests (IPv4)
- Added IPv6 pool allocation tests
- Added IP pool exhaustion handling tests
- Added session stats tracking tests
- Added concurrent session stats tests
- Added rate enforcer throttling tests
- Added rate enforcer edge case tests

### Functional Tests - Access Control
- Added allowlist-only access control tests
- Added denylist-only access control tests
- Added empty policy access control tests
- Added combined allowlist/denylist tests

### Unit Tests - Utilities
- Added formatBytes unit tests
- Added large number format tests

### Version Bump
- Bumped patch version to 0.5.16.

## [0.5.15] - 2026-03-14

### Functional Tests - Billing System
- Added time-based billing functional tests (TestFunctional_TimeBasedBilling)
- Added time-based billing payment renewal tests
- Added time-based billing threshold behavior tests
- Added data-based billing functional tests (TestFunctional_DataBasedBilling)
- Added data-based billing payment renewal tests
- Added data-based billing tiers tests (1KB, 1MB, 10MB)
- Added spending limit enforcement functional tests
- Added session spending limit functional tests
- Added spending warning threshold functional tests

### Functional Tests - Multi-Tunnel
- Added multi-tunnel concurrent connection tests (TestFunctional_MultiTunnelConcurrent_Connection)
- Added multiple providers concurrent test (5 providers)
- Added duplicate tunnel ID rejection test
- Added specific tunnel cancel test
- Added concurrent tunnel add test (10 concurrent)
- Added list interfaces mapping test

### Version Bump
- Bumped patch version to 0.5.15.

## [0.5.10] - 2026-03-14

### Tunnel Session Management
- Added Close() method to clientSession to properly shut down connections and release resources.
- Ensured MultiTunnelManager CancelAll properly cancels and waits for tunnels (already existed).

### Testing
- Added unit tests for clientSession.Close and sessionStats concurrency.
- Added tests for MultiTunnelManager CancelAll concurrency.

### Version Bump
- Bumped patch version to 0.5.10.

## [0.5.9] - 2026-03-12

### Platform-Specific Improvements
- Added pre-restore existence checks in cleanup_marker_linux.go - validates provider IP and DNS server before restore
- Added mockable `osGeteuid` variable in privilege_linux.go for unit testing
- Added unit test for privilege_linux.go with mock support

### GUI/UX Improvements
- Added inline validation error display for form fields in Provider and Client tabs
- Replaced modal dialog popups with per-field error labels for validation errors
- Fields with inline validation: price, max consumers, listen port, cert lifetime, rotate window, announce IP, TUN IP, TUN subnet, health check interval, max price, min bandwidth, max latency, min slots
- Added `validatedField` helper for wrapping form entries with error labels

### Testing & Quality
- Expanded fuzz test corpus in `internal/protocol/fuzz_test.go` with edge cases:
  - Malformed magic bytes
  - Zero-length fields
  - Boundary values
  - Integer overflows
  - Extremely long payloads
  - PAY/PRICE/HEARTBEAT variations
- Added comprehensive unit tests for DNS introspection (`hasExpectedSecureDNS`)
- Added integration test prerequisites documentation to TEST_COVERAGE.md

---

## [0.5.8] - 2026-03-12

## [0.5.7] - 2026-03-12

### Configuration & Validation Improvements
- Added minimum/maximum bounds validation for duration fields:
  - `provider.health_check_interval` (minimum 1s, maximum 24h)
  - `provider.bandwidth_monitor_interval` (minimum 1s, maximum 24h)
  - `provider.announcement_interval` (minimum 1h, maximum 7d)
- Added cross-field validation: `cert_lifetime_hours` > `cert_rotate_before_hours`
- Added cross-field validation: `max_session_duration_secs` <= `cert_lifetime_hours`
- Added `applyConfigDefaults()` to ensure all config defaults are applied consistently
- Added configurable `provider.announcement_interval` (default: 24h)
- Added configurable `provider.dns_servers` and `client.dns_servers` arrays
- Default config now generates secure random RPC password instead of empty
- Added `GenerateRandomRPCPassword()` function for secure credential generation
- Added configurable `provider.shutdown_timeout` (default: 10s)

### Error Handling & Logging
- Added debug logging for scanner hex.Decode failures
- Added debug logging for protocol.ExtractScriptPayload errors
- Added retry attempt logging in blockchain operations
- Added error handling for chainhash.NewHashFromStr in payment.go
- Added validation for btcec.PrivKeyFromBytes result in crypto.go

### Goroutine & Resource Management
- Added 30-second connection timeout to `MultiTunnelManager.Add()` to prevent indefinite blocking
- Provider shutdown timeout now configurable via `provider.shutdown_timeout` config field

### CLI Improvements
- Added `handleError` and `handleErrorFn` helper functions for consistent error handling in command handlers

### Code Quality
- **Config Get/Set Refactoring**: Replaced massive 200+ line switch statements with reflection-based field registry in `internal/config/config_registry.go`. Reduces maintenance burden when adding new config fields. File reduced by ~200 lines.

### Observability & Diagnostics
- Added retry operation metrics to `/metrics.json` endpoint: total_retries, total_failures, last_retry_op, retries_by_operation
- Added goroutine count tracking to `/metrics.json` endpoint for leak detection

### Tests
- Added comprehensive unit tests for retry logic
- Added unit tests for crypto error paths
- Added unit tests for config validation bounds
- Added unit tests for cross-field validation
- Added unit tests for applyConfigDefaults
- Added unit tests for config registry (GetConfigField, SetConfigField, ListConfigFields)
- Added unit tests for retry metrics recording

---

## [0.5.4] - 2026-03-12

### Configuration & Validation Improvements
- Added minimum/maximum bounds validation for duration fields:
  - `provider.health_check_interval` (minimum 1s, maximum 24h)
  - `provider.bandwidth_monitor_interval` (minimum 1s, maximum 24h)
  - `provider.announcement_interval` (minimum 1h, maximum 7d)
- Added cross-field validation: `cert_lifetime_hours` > `cert_rotate_before_hours`
- Added cross-field validation: `max_session_duration_secs` <= `cert_lifetime_hours`
- Added `applyConfigDefaults()` to ensure all config defaults are applied consistently
- Added configurable `provider.announcement_interval` (default: 24h)
- Added configurable `provider.dns_servers` and `client.dns_servers` arrays
- Default config now generates secure random RPC password instead of empty
- Added `GenerateRandomRPCPassword()` function for secure credential generation
- Added configurable `provider.shutdown_timeout` (default: 10s)

### Error Handling & Logging
- Added debug logging for scanner hex.Decode failures
- Added debug logging for protocol.ExtractScriptPayload errors
- Added retry attempt logging in blockchain operations
- Added error handling for chainhash.NewHashFromStr in payment.go
- Added validation for btcec.PrivKeyFromBytes result in crypto.go

### Goroutine & Resource Management
- Added 30-second connection timeout to `MultiTunnelManager.Add()` to prevent indefinite blocking
- Provider shutdown timeout now configurable via `provider.shutdown_timeout` config field

### CLI Improvements
- Added `handleError` and `handleErrorFn` helper functions for consistent error handling in command handlers

### Tests
- Added comprehensive unit tests for retry logic
- Added unit tests for crypto error paths
- Added unit tests for config validation bounds
- Added unit tests for cross-field validation
- Added unit tests for applyConfigDefaults

---

## [0.5.2] - 2026-03-12

### Testing & Reliability
- Added comprehensive unit tests for `UsageMeter` (time/data billing metering)
- Added comprehensive unit tests for `SpendingManager` (spending limits, auto-recharge, session tracking)
- Added complete test suite for `AuthManager` (authorization, data quotas, expiration, consumption)
- Added unit tests for `session` package (sessionStats, rateEnforcer, bandwidth parsing)
- Added unit tests for payment verification functions
- Fixed bug in `AuthManager.AuthorizePeer` where extending a peer with additional data quota failed to increment `dataQuota`
- Updated GUI tests to use demo mode to avoid RPC timeouts
- Added `docs/TEST_COVERAGE.md` with detailed test gap analysis and roadmap

### Infrastructure
- Improved test coverage for critical billing and auth components ahead of v1.0 release

---

## [0.5.0] - 2026-03-11

### Features: Flexible Pricing Models
- Added support for multiple pricing methods: session-based, time-based (per minute/hour), and data-based (per MB/GB)
- Extended on-chain protocol to V3 with new fields: pricing method, time/data units, session timeout
- Provider can configure pricing method, billing units, and session timeouts in config
- Client automatically interprets provider's pricing model and handles appropriate payment amounts

### Features: Usage Metering and Incremental Payments
- Implemented client-side usage meter to track time and data consumption
- For time-based pricing: periodic payments based on elapsed time
- For data-based pricing: tiered payments as data thresholds are crossed
- Provider now grants authorization based on payment amount (duration or data quota)
- Sessions can be extended automatically through continuous payments (auto-pay)

### Features: Client Spending Limits and Controls
- Added comprehensive spending management with configurable limits
- Total daily spending cap with warning thresholds
- Per-session spending maximums
- Auto-disconnect on limit exhaustion
- Prepaid credit balance system with auto-recharge
- All spending tracked and persisted in history

### Features: Multi-Blockchain and Network Support
- Added configurable RPC network (mainnet, testnet, regtest, simnet, custom)
- Token symbol and display unit configuration
- Automatic network detection from blockchain info
- Display amounts using correct token symbol (e.g., BTC, LTC, ORDEX)
- Fee estimation works across supported networks

### Features: Enhanced Filtering
- Scan command now supports filtering by pricing method (`--pricing-method`)
- Filters apply to all pricing models appropriately
- GUI scan dialog updated with pricing method filter

### Features: Demo/Simulation Mode (GUI)
- Added `demo_mode` configuration option to bypass backend requirements
- Generates mock provider data for UI/UX testing without blockchain daemon
- Enables quick testing of scanning, connection flow, and provider selection
- Toggle in settings tab; when enabled, skip RPC connections and use simulated data
- New `-demo` command-line flag to launch directly in demo mode and skip setup wizard

### Command Line and RPC Improvements
- GUI now supports `-demo` / `--demo` flag for QA/UX testing without a backend
- RPC client now supports cookie-based authentication automatically
- Improved RPC connection warmup handling with `waitForServerReady`
- Default RPC ports configured per network (mainnet/testnet/signet/regtest)
- Configurable RPC network selection and token symbol display

### Security and Bug Fixes
- Provider authorization now supports data quotas and dynamic expiration based on pricing model
- AuthManager extended to track remaining data quota per peer
- Payment monitor computes authorization duration/data based on payment amount and provider's pricing
- Client spending manager validates limits before payments
- Graceful disconnect when spending limits are reached

### Infrastructure
- Added `internal/tunnel/usage.go` for usage metering
- Renamed `CreditManager` to `SpendingManager` with expanded functionality
- Extended protocol V3 in `internal/protocol/vpn_protocol.go`
- Updated config validation for new fields
- Updated scanner to decode V3 payloads and populate pricing fields
- All tests passing; backward compatibility maintained for V1/V2 providers

---

## [0.4.10] - 2026-03-07

### GUI/UX Improvements
- **Item 5.7 Country Filter Dropdown**: Replaced the free-text country entry in the "Client Mode" tab with a searchable `SelectEntry` dropdown. The dropdown is pre-populated with common country codes and dynamically updated with unique codes discovered during provider scans.

## [0.4.9] - 2026-03-07

### GUI/UX Improvements
- **Item 5.6 Wallet Balance Display**: Added real-time wallet balance display to the "Wallet" tab. The balance is fetched via RPC and integrated into the auto-refresh cycle for continuous updates.

## [0.4.8] - 2026-03-07

### GUI/UX Improvements
- **Item 5.5 Real-time Metrics**: Overhauled the "Network Status" tab with reactive data bindings. Added a robust auto-refresh mechanism (5s interval) for connection stats, provider/client metrics, and detailed runtime events.

## [0.4.7] - 2026-03-07

### GUI/UX Improvements
- **Item 5.4 Confirmation Dialogs**: Added confirmation dialog to "Disconnect All" in client mode to prevent accidental disconnection.

## [0.4.6] - 2026-03-07

### GUI/UX Improvements
- **Item 5.3 Log Panel Enhancements**: Added toolbar to log panel with auto-scroll toggle, search/filter entry, and an export button. Switched to high-performance log storage with background filtering to support large log volumes.

## [0.4.5] - 2026-03-07

### GUI/UX Improvements
- **Item 5.2 Progress Indicators**: Added infinite progress bars for scanning and connecting operations in the Client Tab. Scanning and connecting buttons now correctly disable while operations are active.

## [0.4.4] - 2026-03-07

### GUI/UX Improvements
- **Item 5.1 Status Label Accuracy**: Implemented reactive data binding for provider status. Status now updates accurately across all tabs (Starting, Running, Stopping, Stopped) and automatically manages start/stop button availability.

## [0.4.3] - 2026-03-07

### Infrastructure
- Updated GitHub Actions permissions to allow automated releases
- Version bump to 0.4.3
 
## [0.4.2] - 2026-03-07

### Features & Protocol
- Added `ThroughputProbePort` to V2 service announcement payload for dynamic discovery
- Added configurable `HeartbeatInterval` and `PaymentMonitorInterval` in provider settings
- Updated blockchain scanner to handle new V2 payload fields

### Security & Bug Fixes
- Improved wallet address detection to check all controlled addresses instead of a single change address
- Fixed concurrent provider start race condition in GUI by decoupling blocking I/O from state locks
- Updated client security checks to use dynamically determined throughput probe ports
- Fixed potential deadlock in GUI log updates

### Tests
- Added GUI unit tests for main tabs, settings, and wallet components
- Refactored Makefile with `test-unit` for better CI/CD integration

### Infrastructure
- Version bump to 0.4.2

## [0.4.1] - 2026-03-07

### Tests & CI/CD
- Split tests into unit and functional suites using build tags
- Updated Makefile with `test-unit`, `test-functional`, and automated `release` targets
- Synced GitHub CI/CD workflows to Go 1.25.x and improved build isolation

## [0.4.0] - 2026-03-07

### Security & Protocol
- Add port conflict detection and auto-rotation for provider
- Add auto-recharge credit system for client (pay-as-you-go)
- Add certificate fingerprint to heartbeat for pinning
- Add certificate rotation announcement on-chain
- Add certificate fingerprint verification for known providers

### Features
- Add CreditManager for automatic payment replenishment
- Add port detection utilities (FindAvailablePort, CheckPortAvailable)
- Add certificate fingerprint payload to on-chain announcements
- Add client auto-recharge configuration options

### Infrastructure
- Version bump to 0.4.0

## [0.3.8] - 2026-03-07

### Security
- Add RPC security documentation in INSTALL.md
- Document localhost-only RPC setup (no password required)
- Document remote RPC with TLS and password
- Document security risks of plaintext password storage
- Update default config to localhost:25173 (ordexcoind default)
- Auto-disable TLS when connecting to localhost

### Infrastructure
- Version bump to 0.3.7

## [0.3.6] - 2026-03-07

### Security
- Add payment amount verification on client side before sending payment
- Provider now verifies actual transaction outputs instead of wallet balance change
- Client and provider both verify payment meets or exceeds advertised price

### Bug Fixes
- Fix payment verification to check actual amounts paid to provider address
- Provider now properly validates incoming payment amounts from transaction vouts

### Infrastructure
- Version bump to 0.3.6

## [0.3.5] - 2026-03-07

### Security
- Add provider liveness check (UDP echo) before payment to prevent fund loss
- Add self-connection detection to prevent routing loops
- Add RPC TLS option (enabled by default) for secure RPC communication
- Enforce metrics auth token when endpoints are enabled
- Remove sensitive payload data from logs

### Bug Fixes
- Add IPv6 packet handling in TUN read loop
- Add bounds checking to prevent panic on short/truncated packets
- Fix DNS leak check to actually block in strict mode
- Fix race condition in provider capacity check
- Add TLS handshake verification before reading cert state
- Add proper timer cleanup to prevent resource leak
- Add proper error handling for change address in payments

### Infrastructure
- Version bump to 0.3.5

## [0.3.4] - 2026-03-07

### Networking & Transport
- Added WebSocket fallback transport support using WSS to bypass restrictive firewalls/proxies.
- Added IPv6 support to internal TUN interface IP pool and cross-platform OS routing layers.

### Multi-Tunneling & Marketplace
- Added support for multiple concurrent tunnels via `MultiTunnelManager`.
- Added provider-assisted throughput probes to replace external internet speed test dependency.
- Added signed provider reputation store and weighted selection algorithm.
- Improved blockchain scanner with delta-scanning (cache based) and mempool-aware fee management.

### Security & Validation
- Added provider-side session duration limits (`max_session_duration_secs`) and automatic disconnect enforcement.
- Added comprehensive validation layer for TUN subnet overlaps and port conflicts.
- Refactored session management in `tunnel.go` with centralized cleanup and leak prevention.
- Hardened memory safety in packet reading loops with buffer pools and enlarged TUN buffers.

### Infrastructure & CI/CD
- Added GitHub Action workflow for automated testing, linting, and cross-platform releases.
- Cleaned up redundant documentation and fully updated `README.md` with current feature set and build guides.
- Refactored `go.mod` to ensure `github.com/btcsuite/websocket` is a direct dependency.
- Added cross-platform CLI build targets and Linux GUI build support in CI.

### Tests & Infrastructure
- Added integration tests for multi-platform NAT traversal and secure key storage.
- Refactored keystore for full platform mockability and added cross-platform test coverage.

## [0.3.0] - 2026-03-07

### Marketplace Protocol
- Added v2 on-chain provider metadata payload with bandwidth, max consumers, country, and availability flags.
- Added scanner support for v1 and v2 payloads with merged provider state.
- Added provider heartbeat/availability broadcasts.

### Discovery and Selection UX
- Added CLI/GUI sorting by bandwidth and capacity.
- Added richer search filters (country, max price, min bandwidth, max latency, available slots).
- Added score/ranking output in CLI/GUI provider listings.

### Client Security Verification
- Added OS-native DNS server introspection checks.
- Added optional strict verification mode for country/IP checks.
- Added throughput verification against advertised provider bandwidth after connect.

### Provider Runtime and Lifecycle
- Added graceful shutdown wait handling for provider goroutine groups in CLI/GUI control flow.
- Added accept-loop backoff + jitter on provider listener errors.
- Added atomic file writes for config/history/cleanup marker updates.

### UX and Operations
- Added per-session event timeline for CLI (`events`) and GUI (Status tab).
- Added config import/export support in CLI and GUI.
- Added diagnostics bundle export in CLI (`diagnostics`) and GUI.

### Tests and Hardening
- Added post-connect security verification tests with mocked checks and throughput feature test.
- Added cleanup-marker recovery coverage on Linux/macOS/Windows integration tests.
- Added protocol tests/fuzz coverage for v2 metadata and heartbeat decoding.

## [0.2.0] - 2026-03-07

### Features
- Added GUI parity for provider controls and diagnostics (rebroadcast, price update, metrics snapshots).
- Added `bcvpn doctor` diagnostics command for environment and privilege verification.
- Added non-interactive key password environment variable support for automation.
- Added structured JSON log mode for CLI/GUI backend actions.
- Added CI/CD matrix and release smoke workflow tooling.
- Added security and operations documentation with automation examples.

### Security & Hardening
- Hardened TLS, metrics authentication, and revocation cache enforcement.
- Integrated hardware-backed provider key storage (macOS Keychain, Windows DPAPI, Linux libsecret).
- Improved crash-safe route and DNS configuration recovery.
- Removed fatal runtime exits to improve TUI/GUI stability.

## [0.1.0] - 2026-03-06

### Core Protocol & Transport
- Replaced WireGuard with custom TLS-over-TUN protocol using keys derived from the blockchain wallet.
- Implemented `OP_RETURN` based on-chain service announcement and discovery protocol.
- Implemented payment flow with deterministic UTXO selection and dynamic fee estimation.
- Implemented payment monitor with reorg handling.

### GUI & Platforms
- Initial implementation of cross-platform GUI using the Fyne toolkit.
- Added support for cross-platform client routing and DNS automation (Linux, macOS, Windows).
- Added provider egress NAT backend support across all supported platforms.
- Integrated GeoIP and latency-based provider enrichment.

### Project Setup
- Initial project structure and internal package organization.
- Implemented core VPN logic (TUN, TLS, Networking) and authorization management.
