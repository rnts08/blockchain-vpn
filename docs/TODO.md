# BlockchainVPN TODO (Open Items)

This list contains only remaining work after the latest parity and runtime pass.

## 1. Marketplace Protocol Completeness
- [ ] Add a v2 on-chain provider metadata payload that includes:
  bandwidth offer, max consumers, provider-declared origin/country, and availability flags.
- [ ] Add scanner support for v2 payloads while keeping backward compatibility with v1 announcements.
- [ ] Add provider heartbeat/availability broadcasts so clients can prefer currently-online providers.

## 2. Discovery and Selection UX
- [ ] Add CLI/GUI filtering and sorting by advertised bandwidth and capacity once v2 metadata is available.
- [ ] Add richer search filters (country, max price, min bandwidth, max latency, available slots).
- [ ] Show effective provider score/ranking inputs directly in scan results (price, latency, country confidence, capacity).

## 3. Client Security Verification
- [ ] Add OS-native DNS server introspection checks (not only DNS query heuristic) to improve leak detection confidence.
- [ ] Add optional strict mode to fail connection when country verification or IP verification checks fail.
- [ ] Add active throughput verification against advertised provider bandwidth after connect.

## 4. Provider Runtime and Lifecycle
- [ ] Add graceful shutdown wait-groups for provider goroutines (listener, echo, payment monitor, health checks).
- [ ] Add accept-loop backoff + jitter for repeated listener errors.
- [ ] Add atomic file writes for config/history/cleanup marker updates.

## 5. UX and Operations
- [ ] Add per-session event timeline (connect/auth/revoke/disconnect/errors) in both CLI and GUI.
- [ ] Add import/export profile commands and GUI actions.
- [ ] Add one-click diagnostics bundle export (redacted config + status JSON + recent logs).

## 6. Tests and Hardening
- [ ] Add integration tests for post-connect security checks with mocked DNS/IP services.
- [ ] Add cross-platform integration coverage for cleanup-marker recovery and restore behavior.
- [ ] Add protocol/property tests for future metadata v2 encoding/decoding and scanner merge logic.
