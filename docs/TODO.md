# BlockchainVPN TODO

## Short-term Targets
- [ ] Move throughput verification to provider-assisted active throughput probes to reduce internet speed test dependency.
- [ ] Add signed provider reputation/quality metadata and weighted selection policy profiles.
- [ ] Implement provider-side session duration limits and auto-disconnect.
- [ ] Add support for multiple TUN interfaces for multi-provider connectivity.

## Medium-term Enhancements
- [ ] **IPv6 Support**: Expand IP pool and tunnel logic to support IPv6 addressing for internal TUN interfaces.
- [ ] **Websocket Transport**: Implement a fallback transport mode using Websockets to bypass restrictive firewalls/proxies.
- [ ] **Delta-Scanning**: Optimize `ScanForVPNs` to use a local cache and only scan new blocks since last run.
- [ ] **Improved Fee Management**: Add dynamic transaction fee adjustments for provider announcements and heartbeats based on mempool congestion.

## Code Quality & Hardening
- [ ] **Session Cleanup**: Refactor `internal/tunnel/tunnel.go` to use a more robust session management system with centralized leak prevention.
- [ ] **Validation Layer**: Implement a comprehensive config validation layer to catch invalid TUN/Subnet ranges and port conflicts early.
- [ ] **Integration Test Coverage**: Expand integration tests for multi-platform NAT traversal and secure key storage backends.
- [ ] **Memory Safety**: Audit packet reading loops in `readTunLoop` for potential buffer overflows or excessive allocation.

