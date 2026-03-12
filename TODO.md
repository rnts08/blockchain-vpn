# BlockchainVPN Implementation Plan

This document tracks the remaining tasks and improvements for the BlockchainVPN project, ordered from easiest to hardest with logical groupings.

---

## Group 1: Quick Wins (Easy, Low Risk)

Isolated fixes that improve debuggability and robustness.

### Error Handling Fixes

- [ ] **2.4 Crypto Key Validation**: Validate `btcec.PrivKeyFromBytes` result is non-nil in `internal/crypto/crypto.go:127`. Currently returns nil error but invalid key if decryption yields invalid bytes.

- [ ] **2.3 Payment Hash errors**: Handle error from `chainhash.NewHashFromStr` in `internal/blockchain/payment.go:210` (error is ignored). Should never fail with RPC data but defensive programming needed.

- [ ] **2.2 Scanner Silent Failures**: Add debug logging for `hex.DecodeString` failures in `internal/blockchain/scanner.go:88` and `protocol.ExtractScriptPayload` errors. Currently these are silently ignored, masking malformed data.

### Logging Improvements

- [ ] **4.2 Retry Logging**: Add debug logs in `internal/blockchain/retry.go:27-46` to make retry attempts visible. Log: "Retry X/Y for operation Z after Wms". Critical for debugging RPC issues.

- [ ] **9.3 Enhanced Logging Context**: Add structured logging (with fields) to key operations: payment monitor found payment, provider server accepting connection, scanner block progress, config validation failures. Use `log.Printf` with key=value pairs.

---

## Group 2: Configuration & Defaults (Medium)

Improvements to config validation and removing hardcoded values.

### Validation Gaps

- [ ] **3.1 Duration Validations**: Add minimum value checks for time duration fields:
  - `provider.health_check_interval` (currently no minimum, could be 0) - minimum 1s
  - `provider.bandwidth_monitor_interval` (not validated at all in `Validate()`) - minimum 1s
  - Add maximum bounds to prevent excessive intervals

- [ ] **3.2 Cross-Field Validations**: Add checks in `internal/config/validate.go`:
  - `cert_lifetime_hours` > `cert_rotate_before_hours` (prevent rotate window larger than lifetime)
  - `max_session_duration_secs` should be <= cert lifetime (or 0 for unlimited)
  - Client metrics port collision with provider listen port

- [ ] **3.3 Default Application**: Ensure `config.LoadConfig` applies all defaults consistently. Currently only `MaxParallelTunnels` gets a default. Create `applyConfigDefaults()` function to handle all defaults.

### Hardcoded Values → Configuration

- [ ] **6.1 Announcement Interval**: The 24-hour reannouncement interval in `cmd/bcvpn/main.go` is hardcoded. Add `provider.announcement_interval` config field with 24h default.

- [ ] **6.2 Default DNS Servers**: The hardcoded DNS servers (`1.1.1.1`, `8.8.8.8`) in `internal/tunnel/network_*.go` should be configurable via config. Add `client.dns_servers` and `provider.dns_servers` fields.

- [ ] **6.3 Default RPC Credentials**: `internal/config/config.go` generates default RPC user "rpcuser" with empty password. Generate random secure credentials on first run OR require explicit setup (remove defaults).

---

## Group 3: Error Handling & Reliability (Medium)

Critical for testing and robustness. These make the CLI testable.

### CLI Error Returns

- [ ] **2.1 Replace log.Fatalf**: Refactor command handlers in `cmd/bcvpn/main.go` to return errors instead of calling `log.Fatalf` directly. Allow `main()` to handle exit codes. This makes commands testable and enables programmatic use. Affects ~20 command handlers.

### Goroutine & Resource Management

- [ ] **4.1 MultiTunnel Leak**: Add timeout to context in `internal/tunnel/multi_tunnel.go` (`Add()` method). The spawned goroutine may block indefinitely if `ConnectToProvider` hangs. Use `context.WithTimeout(parent, 30s)` and ensure `done` channel closes even on panic.

- [ ] **4.3 Provider Shutdown Timeout**: The 5-second timeout in `cmd/bcvpn/main.go` may be too short if goroutines hang. Make it dynamic based on number of running goroutines. Ensure all provider goroutines respect `ctx.Done()`.

---

## Group 4: Code Quality & Abstraction (Hard)

Refactoring to reduce duplication. Higher risk of breaking changes.

### Code Duplication

- [ ] **5.3 Config Get/Set Refactoring**: Replace massive 200+ line switch statements in `cmd/bcvpn/main.go` (`getConfigField`/`setConfigField`) with a reflection-based field registry or map. Reduces maintenance burden when adding new config fields.

- [ ] **5.1 DNS Introspection Abstraction**: Factor out common parsing logic duplicated in `internal/tunnel/dns_introspection_linux.go`, `dns_introspection_darwin.go`, `dns_introspection_windows.go`. Create a shared helper for parsing `resolv.conf` format; platform files only implement command execution.

- [ ] **5.2 Network Configuration Abstraction**: Extract common route setup and DNS configuration logic from `internal/tunnel/network_linux.go`, `network_darwin.go`, `network_windows.go`. These files share significant code for route manipulation, DNS writes, and cleanup.

---

## Group 5: Observability & Diagnostics (Medium-Hard)

Metrics and monitoring improvements.

- [ ] **9.1 Retry Operation Metrics**: Instrument `retry.go` to expose retry count, backoff durations, and failure reasons via the metrics endpoint (`/metrics.json`). Helps diagnose connectivity issues in production.

- [ ] **9.2 Goroutine Leak Detection**: Add runtime goroutine count monitoring to `providerStatus` output. Alert if goroutine count grows unexpectedly over time (indicative of leaks in multi_tunnel or rotation loops).

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

## Implementation Order

**Start here (Group 1)**:
1. 2.4 Crypto Key Validation
2. 2.3 Payment Hash errors
3. 2.2 Scanner Silent Failures
4. 4.2 Retry Logging
5. 9.3 Enhanced Logging Context

**Then (Group 2)**:
1. 3.1 Duration Validations
2. 3.2 Cross-Field Validations
3. 3.3 Default Application
4. 6.1 Announcement Interval
5. 6.2 Default DNS Servers
6. 6.3 Default RPC Credentials

**Then (Group 3)**:
1. 2.1 Replace log.Fatalf
2. 4.1 MultiTunnel Leak
3. 4.3 Provider Shutdown Timeout

**Then (Group 4-10)** as time permits.
