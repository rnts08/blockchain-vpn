# Changelog

All notable changes to this project will be documented in this file.

## [0.3.8] - 2026-03-07

### Features
- Add port conflict detection and auto-rotation for provider
- Add auto-recharge credit system for client (pay-as-you-go)
- Add certificate fingerprint to heartbeat for pinning
- Add certificate rotation announcement on-chain

### Bug Fixes
- Add certificate fingerprint verification for known providers

### Infrastructure
- Version bump to 0.3.8

## [0.3.7] - 2026-03-07

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
