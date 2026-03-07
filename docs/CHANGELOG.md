# Changelog

All notable changes to this project will be documented in this file.

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
