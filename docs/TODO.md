# BlockchainVPN TODO

## Completed In 0.3.0

### Marketplace Protocol Completeness
- [x] Added v2 on-chain provider metadata payload with bandwidth, max consumers, country, and availability flags.
- [x] Added scanner support for v1 and v2 payloads with merged provider state.
- [x] Added provider heartbeat/availability broadcasts.

### Discovery and Selection UX
- [x] Added CLI/GUI sorting by bandwidth and capacity.
- [x] Added richer search filters (country, max price, min bandwidth, max latency, available slots).
- [x] Added score/ranking output in CLI/GUI provider listings.

### Client Security Verification
- [x] Added OS-native DNS server introspection checks.
- [x] Added optional strict verification mode for country/IP checks.
- [x] Added throughput verification against advertised provider bandwidth after connect.

### Provider Runtime and Lifecycle
- [x] Added graceful shutdown wait handling for provider goroutine groups in CLI/GUI control flow.
- [x] Added accept-loop backoff + jitter on provider listener errors.
- [x] Added atomic file writes for config/history/cleanup marker updates.

### UX and Operations
- [x] Added per-session event timeline for CLI (`events`) and GUI (Status tab).
- [x] Added config import/export support in CLI and GUI.
- [x] Added diagnostics bundle export in CLI (`diagnostics`) and GUI.

### Tests and Hardening
- [x] Added post-connect security verification tests with mocked checks and throughput feature test.
- [x] Added cleanup-marker recovery coverage on Linux/macOS/Windows integration tests.
- [x] Added protocol tests/fuzz coverage for v2 metadata and heartbeat decoding.

## Future Iteration Candidates
- [ ] Move throughput verification to provider-assisted active throughput probes to reduce internet speed test dependency.
- [ ] Add signed provider reputation/quality metadata and weighted selection policy profiles.
