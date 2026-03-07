# BlockchainVPN TODO (Next Iteration)

This list is a new post-MVP backlog based on a full code and docs review.

## 1. Security and Correctness (Highest Priority)
- [ ] Use cryptographically random X.509 serial numbers for generated TLS certificates (replace fixed serial `1`).
- [ ] Remove `log.Fatalf` from internal runtime packages (`internal/tunnel`, `internal/blockchain`) and return errors to callers for controlled shutdown.
- [ ] Add optional authentication for `/metrics.json` endpoint (token/header) and document safe bind defaults.
- [ ] Add explicit key-storage backend health checks and fallback behavior when secure-store commands/tools are missing at runtime.
- [ ] Add validation and normalization for revocation cache entries (reject duplicates, report invalid line numbers).

## 2. Runtime Resilience
- [ ] Add graceful stop/wait lifecycle for provider goroutines (echo server, payment monitor, health checks, listener workers).
- [ ] Add exponential backoff + jitter for provider listener accept errors and metrics server restart loops.
- [ ] Add atomic write helpers for config/history/cleanup-marker updates to reduce corruption risk on crashes.
- [ ] Add startup self-check command (`bcvpn doctor`) to verify TUN, routing tools, key storage backend, and config validity.

## 3. Networking and Platform Quality
- [ ] Add Linux nftables backend option for kill switch/NAT where iptables-legacy is unavailable.
- [ ] Improve Windows routing backend to prefer interface GUID/index stability across reboots/renames.
- [ ] Add macOS service/interface selection fallback logic when default route lookup is ambiguous (multi-homed hosts).
- [ ] Add runtime probes that verify DNS leak posture after connect (and surface result in status/GUI).

## 4. Observability and Operator UX
- [ ] Add log levels (`debug`, `info`, `warn`, `error`) and include them in JSON log output.
- [ ] Expose provider/client runtime health and metrics in GUI Status tab (not only via endpoint).
- [ ] Add per-session timeline/events (connect/auth/revoke/disconnect) to CLI/GUI for troubleshooting.
- [ ] Add alerting hooks/webhook support for provider failures (listener down, TUN down, auth failures burst).

## 5. CLI and GUI Quality of Life
- [ ] Add non-interactive provider key bootstrap/rotate flags for automation (`--password-env`, secure prompt fallback).
- [ ] Add import/export profile support in GUI and CLI (`config export`, `config import --validate`).
- [ ] Add one-click “copy diagnostics bundle” (status JSON + redacted config + recent logs).
- [ ] Add provider/client preset templates for common home-router and VPS deployments.

## 6. Testing and Release Engineering
- [ ] Add integration test matrix in CI for Linux runtime networking paths with privilege-capable test environment.
- [ ] Add deterministic tests for key storage mode resolution and backend fallback behavior.
- [ ] Add fuzz tests for protocol payload parsing and revocation cache parsing.
- [ ] Add release smoke test scripts for CLI and GUI startup on Linux/macOS/Windows artifacts.

## 7. Documentation
- [ ] Add dedicated security model document (`docs/SECURITY.md`) covering trust boundaries and threat model.
- [ ] Add explicit secure-store backend prerequisites by OS (Keychain/libsecret/DPAPI) with troubleshooting.
- [ ] Add operations runbook for providers (rotation, revocation, incident response, upgrade strategy).
- [ ] Add examples for `status --json` and `/metrics.json` fields for automation tooling.
