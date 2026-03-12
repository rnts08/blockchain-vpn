# BlockchainVPN Implementation Plan

This document tracks the remaining tasks and improvements for the BlockchainVPN project, ordered from easiest to hardest with logical groupings.

---

## Group 6: Platform-Specific (Hard)

OS-dependent code with risks. Requires testing on multiple platforms.

- [ ] **8.1 Cleanup Marker Location**: The marker files for network state recovery (`internal/tunnel/cleanup_marker_*.go`) use hardcoded paths (e.g., `/etc/blockchain-vpn-network-marker`). Move to application config directory for consistency and user-writable locations. Add pre-restore existence checks.

- [ ] **8.2 Kill Switch Consolidation**: Review `killswitch_linux.go`, `killswitch_darwin.go`, `killswitch_windows.go` for duplication. Extract common rule management logic where possible. Audit for resource leaks to ensure iptables/network rules are restored even on partial failure.

- [ ] **8.3 Privilege Escalation Tests**: Add tests for `privilege_linux.go` (and equivalents) covering failure modes: sudo prompts without TTY, missing sudoers config, user rejection. Ensure clear error messages.

---

## Group 7: GUI/UX (Medium)

- [ ] **5.8 Validation Highlighting**: Add inline error display with red borders for invalid form fields. Replace modal `dialog.ShowError()` popups with per-field error labels and visual highlighting. Affects Provider Tab (~20 fields), Client Tab (~10 fields), Settings Tab (~13 fields) in `cmd/bcvpn-gui/main.go`.

---

## Group 8: Testing (Medium-Hard)

- [ ] **7.2 Fuzz Test Coverage**: Expand protocol fuzz testing beyond current `internal/protocol/fuzz_test.go`. Add corpus directory with diverse inputs. Target edge cases: malformed magic bytes, zero-length fields, boundary values, integer overflows, extremely long payloads.

- [ ] **7.3 Unit Test Coverage Gaps**:
  - Platform helper functions in `network_*.go` (prefix calculations, route formatting)
  - DNS introspection parsing (malformed resolv.conf, comments, empty lines)
  - `client_security_checks.go` edge cases (strict verification failures, throughput thresholds, country mismatches, DNS leak heuristic)
  - Cancellation scenarios: MultiTunnelManager `CancelAll`, provider goroutine cleanup, `readTunLoop` on interface close
  - Payment coin selection edge cases: insufficient funds, exact match, zero-value UTXOs, duplicate references

- [ ] **7.4 Integration Test Reliability**: Current platform integration tests (`*_integration_test.go`) require elevated privileges and specific network setup. Document prerequisites and consider adding a test mode that skips privileged operations.

---

## Group 9: Security (Medium)

- [ ] **10.1 Metrics Auth Token Validation**: Require at least 16 chars with mixed character classes, or generate a secure token if left empty and metrics are enabled. Current minimum is 12 chars with no entropy check.

- [ ] **10.2 TLS Cipher Suite Agility**: The `tls_policy.go` cipher suite list for "compat" profile is static. Add a way to override via config without code change.

- [ ] **10.3 Provider Key Rotation Audit**: Document what happens to old subscriptions when rotating provider key. Add explicit revocation of old key if needed.

---

## Group 10: General (Long-term)

- [ ] **1.1 Update the readme** with the latest release, and include the latest changes from the changelog in the feature list.

- [ ] **1.2 Add the donation/support addresses** and text to the help/about sections of both the cli and gui.

- [ ] **1.3 Start planning a TUI** that has the same functionality as the cli but is run in a terminal with real-time information much like the GUI.

- [ ] **1.4 Update docs/TEST_COVERAGE.md** continuously while adding tests for all relevant code paths.

- [ ] **1.5 Create a mock rpc server** so that e2e tests can be run as a utility, similar to the rpc-test utility.

- [ ] **1.6 Document the rpc-test utility**, add an option to the makefile to compile it, add a help section to it so that it can be used to verify RPC functionality according to the RPC spec.

---

## Completed Groups

### Group 5: Observability & Diagnostics (Done)
- **9.1 Retry Operation Metrics**: Added retry metrics to /metrics.json (total_retries, total_failures, last_retry_op, retries_by_operation)
- **9.2 Goroutine Leak Detection**: Added goroutine count to /metrics.json endpoint

### Group 4: Code Quality & Abstraction (Done)
- **5.3 Config Get/Set Refactoring**: Create reflection-based config registry (5.3)
- DNS introspection (5.1) - common logic already extracted
- Network abstraction (5.2) - skipped due to platform-specific complexity

### Group 3: Error Handling & Reliability (Done)
- Add 30-second connection timeout to MultiTunnelManager.Add() (4.1)
- Add configurable provider.shutdown_timeout (4.3)
- Add handleError helper functions for CLI error handling (2.1)

### Group 2: Configuration & Defaults (Done)
- Add min/max bounds for duration fields (3.1)
- Add cross-field validation (3.2)
- Add applyConfigDefaults() (3.3)
- Add provider.announcement_interval (6.1)
- Add provider.dns_servers and client.dns_servers (6.2)
- Generate random RPC password (6.3)

### Group 1: Quick Wins (Done)
- Crypto key validation (2.4)
- Payment hash error handling (2.3)
- Scanner debug logging (2.2)
- Retry logging (4.2)
- Enhanced logging context (9.3)
