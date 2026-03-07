# Changelog

All notable changes to this project will be documented in this file.
 
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
