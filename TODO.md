# BlockchainVPN Implementation Plan

This document tracks the remaining tasks and improvements for the BlockchainVPN project, ordered from easiest to hardest with logical groupings.

---

## Group 6: Platform-Specific (Hard)

OS-dependent code with risks. Requires testing on multiple platforms.

- [x] **8.1 Cleanup Marker Location**: Already implemented in config.AppConfigDir(). Added pre-restore existence checks for provider IP validity and DNS server verification.

- [x] **8.2 Kill Switch Consolidation**: Already done - common code (resolveProviderIPv4) in killswitch_common.go. Each platform uses different firewall tools (iptables/pfctl/route) - no further consolidation practical.

- [x] **8.3 Privilege Escalation Tests**: Added unit test in privilege_linux_test.go with mockable osGeteuid function. Code only checks euid - no sudo prompting exists.

---

## Group 7: GUI/UX (Medium)

- [x] **5.8 Validation Highlighting**: Add inline error display with red borders for invalid form fields. Replace modal `dialog.ShowError()` popups with per-field error labels and visual highlighting. Affects Provider Tab (~20 fields), Client Tab (~10 fields), Settings Tab (~13 fields) in `cmd/bcvpn-gui/main.go`.

---

## Group 8: Testing (Medium-Hard)

- [x] **7.2 Fuzz Test Coverage**: Expand protocol fuzz testing beyond current `internal/protocol/fuzz_test.go`. Add corpus directory with diverse inputs. Target edge cases: malformed magic bytes, zero-length fields, boundary values, integer overflows, extremely long payloads.

- [x] **7.3 Unit Test Coverage Gaps**:
  - Platform helper functions in `network_*.go` (prefix calculations, route formatting)
  - DNS introspection parsing (malformed resolv.conf, comments, empty lines)
  - `client_security_checks.go` edge cases (strict verification failures, throughput thresholds, country mismatches, DNS leak heuristic)
  - Cancellation scenarios: MultiTunnelManager `CancelAll`, provider goroutine cleanup, `readTunLoop` on interface close
  - Payment coin selection edge cases: insufficient funds, exact match, zero-value UTXOs, duplicate references

- [x] **7.4 Integration Test Reliability**: Current platform integration tests (`*_integration_test.go`) require elevated privileges and specific network setup. Document prerequisites and consider adding a test mode that skips privileged operations.

---

## Group 9: Security (Medium)

- [x] **10.1 Metrics Auth Token Validation**: Require at least 16 chars with mixed character classes. Current minimum is 12 chars with no entropy check.

- [x] **10.2 TLS Cipher Suite Agility**: The `tls_policy.go` cipher suite list for "compat" profile is now configurable via `security.tls_custom_cipher_suites` config field.

- [x] **10.3 Provider Key Rotation Audit**: Documented what happens to old subscriptions when rotating provider key in README.

---

## Group 10: General (Long-term)

- [x] **1.1 Update the readme** with the latest release, and include the latest changes from the changelog in the feature list.

- [x] **1.2 Add the donation/support addresses** and text to the help/about sections of both the cli and gui.

- [ ] **1.3 Start planning a TUI** that has the same functionality as the cli but is run in a terminal with real-time information much like the GUI.

- [x] **1.4 Update docs/TEST_COVERAGE.md** continuously while adding tests for all relevant code paths.

- [x] **1.5 Create a mock rpc server** so that e2e tests can be run as a utility, similar to the rpc-test utility.

- [x] **1.6 Document the rpc-test utility**, add an option to the makefile to compile it, add a help section to it so that it can be used to verify RPC functionality according to the RPC spec.

---

## Group 11: Test Coverage Expansion (Medium-Hard)

Critical packages needing unit tests:

- [x] **11.1 Auth Package**: Add unit tests for `internal/auth` - AuthManager data quotas, session authorization, CanAuthorize logic, expiration handling

- [x] **11.2 GeoIP Package**: Add unit tests for `internal/geoip` - EnrichEndpoints, country lookup, latency measurement with mock GeoIP database

- [x] **11.3 History Package**: Add unit tests for `internal/history` - Payment history persistence, query, export operations

- [x] **11.4 NAT Package**: Add unit tests for `internal/nat` - UPnP/NAT-PMP mapping with platform mocks

- [x] **11.5 OBS Package**: Add unit tests for `internal/obs` - Logging, metrics collection, event recording

- [x] **11.6 Blockchain Scanner**: Add unit tests for `internal/blockchain/scanner` - V3 payload decoding, delta scanning, filter application

- [x] **11.7 Blockchain Payment**: Add unit tests for `internal/blockchain/payment` - BuildAndSendPayment, fee estimation, retry logic

- [x] **11.8 Blockchain Provider**: Add unit tests for `internal/blockchain/provider` - Announcement creation, reputation store integration

- [x] **11.9 Tunnel Session**: Add unit tests for `internal/tunnel/session` - Session lifetime, authorization renewal, cleanup

- [x] **11.10 Tunnel Multi-Tunnel**: Add unit tests for `internal/tunnel/multi_tunnel` - Concurrent session management, Add/Cancel/ActiveCount

E2E/Functional tests:

- [ ] **11.11 Time-based Billing**: Functional test for time-based billing cycle (provider with pricing_method=time)

- [ ] **11.12 Data-based Billing**: Functional test for data-based billing tiers (provider with pricing_method=data)

- [ ] **11.13 Spending Limits**: Functional test for spending limit enforcement

- [ ] **11.14 Multi-tunnel Concurrent**: Functional test for connecting to multiple providers simultaneously

---

## Completed Groups

### Group 6: Platform-Specific (Done)
- **8.1 Cleanup Marker**: Added pre-restore existence checks for provider IP and DNS server validation
- **8.2 Kill Switch Consolidation**: Already done - common code in killswitch_common.go
- **8.3 Privilege Tests**: Added unit test with mockable osGeteuid

### Group 8: Testing (Done)
- **7.2 Fuzz Test Coverage**: Expanded protocol fuzz test corpus with edge cases
- **7.3 Unit Test Coverage Gaps**: Added DNS introspection tests, security check tests exist
- **7.4 Integration Test Reliability**: Added prerequisites documentation

### Group 7: GUI/UX (Done)
- **5.8 Validation Highlighting**: Added inline error display with per-field error labels, replaced modal dialogs for field validation in Provider and Client tabs

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
