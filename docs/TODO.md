# BlockchainVPN TODO (Next Iteration)

## 1. Click-and-Run UX
- [x] Add GUI auto-elevation/relaunch flow (macOS/Windows/Linux) so users can start networking features without manual terminal steps.
- [x] Add first-run setup wizard in GUI (RPC check, config generation, key creation, privilege check).
- [x] Expose all provider/client settings in a dedicated GUI Settings tab with validation hints and defaults.
- [x] Add CLI `config` subcommands (`get`, `set`, `validate`) for scriptable settings management.

## 2. Cross-Platform Networking Parity
- [x] Implement provider egress NAT backend for macOS.
- [x] Implement provider egress NAT backend for Windows.
- [x] Add cross-platform kill switch mode (block outbound traffic if tunnel drops unexpectedly).
- [ ] Add integration tests that verify route and DNS restore logic on each supported OS backend.

## 3. Reliability and Observability
- [ ] Add provider/client runtime metrics endpoint (session count, throughput, errors, health state).
- [ ] Add structured JSON log mode for both CLI and GUI backend actions.
- [ ] Add persistent crash-safe cleanup markers to restore route/DNS state after abnormal termination.

## 4. Security Hardening
- [ ] Add optional hardware-backed key storage integration where available (Keychain/DPAPI/libsecret).
- [ ] Add mutual TLS certificate revocation cache to immediately drop revoked clients.
- [ ] Add configurable minimum TLS policy and cipher/profile reporting in `status --json`.

## 5. Product and Documentation
- [ ] Add complete user guide with end-to-end flows for provider and client modes.
- [ ] Add troubleshooting guide by OS (permission denied, TUN creation failure, route/DNS conflicts, firewall issues).
- [ ] Add packaging and installer docs for Linux/macOS/Windows releases.
