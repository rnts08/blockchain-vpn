# BlockchainVPN TODO List

This document tracks the remaining tasks and improvements for the BlockchainVPN project.

---

## GUI/UX Improvements

- [ ] **5.8 Validation Highlighting**: Add inline error display with red borders for invalid form fields. Replace modal `dialog.ShowError()` popups with per-field error labels and visual highlighting. Affects Provider Tab (~20 fields), Client Tab (~10 fields), Settings Tab (~13 fields) in `cmd/bcvpn-gui/main.go`.

---

## Error Handling & Reliability

### CLI Error Returns (Critical for Testing)
- [ ] **2.1 Replace log.Fatalf**: Refactor command handlers in `cmd/bcvpn/main.go` to return errors instead of calling `log.Fatalf` directly. Allow `main()` to handle exit codes. This makes commands testable and enables programmatic use. Affects ~20 command handlers (lines 47, 53, 64, 67, 133, 144, 149, 267, 287, 333, etc.).

### Missing Error Checks
- [ ] **2.2 Scanner Silent Failures**: Add debug logging for `hex.DecodeString` failures in `internal/blockchain/scanner.go:83-85` and `protocol.ExtractScriptPayload` errors (line 88). Currently these are silently ignored, masking malformed data.
- [ ] **2.3 Payment Hash errors**: Handle error from `chainhash.NewHashFromStr` in `internal/blockchain/payment.go:210` (error is ignored). Should never fail with RPC data but defensive programming needed.
- [ ] **2.4 Crypto Key Validation**: Validate `btcec.PrivKeyFromBytes` result is non-nil in `internal/crypto/crypto.go:121`. Currently returns nil error but invalid key if decryption yields invalid bytes.

---

## Configuration

### Validation Gaps
- [ ] **3.1 Duration Validations**: Add minimum value checks for time duration fields:
  - `provider.health_check_interval` (currently no minimum, could be 0)
  - `provider.bandwidth_monitor_interval` (not validated at all in `Validate()`)
  - Consider maximum bounds to prevent excessive intervals
- [ ] **3.2 Cross-Field Validations**: Add checks in `internal/config/validate.go`:
  - `cert_lifetime_hours` > `cert_rotate_before_hours` (prevent rotate window larger than lifetime)
  - `max_session_duration_secs` should be <= cert lifetime (or 0 for unlimited)
  - Client metrics port collision with provider listen port (like existing check for provider metrics)
- [ ] **3.3 Default Application**: Ensure `config.LoadConfig` calls `applyConfigDefaults()` after loading to apply all defaults consistently (only `MaxParallelTunnels` gets a default currently).

---

## Goroutine & Resource Management

- [ ] **4.1 MultiTunnel Leak**: Add timeout to context in `internal/tunnel/multi_tunnel.go:70-93` (`Add()` method). The spawned goroutine may block indefinitely if `ConnectToProvider` hangs. Use `context.WithTimeout(parent, 30s)` and ensure `done` channel closes even on panic.
- [ ] **4.2 Retry Logging**: Add debug logs in `internal/blockchain/retry.go:27-46` to make retry attempts visible. Log: "Retry X/Y for operation Z after Wms". Critical for debugging RPC issues.
- [ ] **4.3 Provider Shutdown Timeout**: The 5-second timeout in `cmd/bcvpn/main.go:236-240` may be too short if goroutines hang. Consider making it configurable or dynamic based on number of running goroutines. Ensure all provider goroutines respect `ctx.Done()`.

---

## Code Quality & Maintainability

### Code Duplication
- [ ] **5.1 DNS Introspection Abstraction**: Factor out common parsing logic duplicated in `internal/tunnel/dns_introspection_linux.go`, `dns_introspection_darwin.go`, `dns_introspection_windows.go`. Create a shared helper for parsing `resolv.conf` format; platform files only implement command execution.
- [ ] **5.2 Network Configuration Abstraction**: Extract common route setup and DNS configuration logic from `internal/tunnel/network_linux.go`, `network_darwin.go`, `network_windows.go`. These files share significant code for route manipulation, DNS writes, and cleanup.
- [ ] **5.3 Config Get/Set Refactoring**: Replace massive 200+ line switch statements in `cmd/bcvpn/main.go:1305-1522` (`getConfigField`/`setConfigField`) with a reflection-based field registry or map. Reduces maintenance burden when adding new config fields.

### Hardcoded Values → Configuration
- [ ] **6.1 Announcement Interval**: The 24-hour reannouncement interval in `cmd/bcvpn/main.go:161` is hardcoded. Add `provider.announcement_interval` config field with sensible default.
- [ ] **6.2 Default DNS Servers**: The hardcoded DNS servers (`1.1.1.1`, `8.8.8.8`) in `internal/tunnel/network_linux.go:16` (and equivalents) should be configurable or at least package-level constants. Users may want custom DNS.
- [ ] **6.3 Default RPC Credentials**: `internal/config/config.go:182-184` generates default RPC user "rpcuser" with empty password. Generate random secure credentials or require explicit setup.

---

## Testing

- [ ] **7.2 Fuzz Test Coverage**: Expand protocol fuzz testing beyond current `internal/protocol/fuzz_test.go`. Add corpus directory with diverse inputs. Target edge cases: malformed magic bytes, zero-length fields, boundary values, integer overflows, extremely long payloads.
- [ ] **7.3 Unit Test Coverage Gaps**:
  - Platform helper functions in `network_*.go` (prefix calculations, route formatting)
  - DNS introspection parsing (malformed resolv.conf, comments, empty lines)
  - `client_security_checks.go` edge cases (strict verification failures, throughput thresholds, country mismatches, DNS leak heuristic)
  - Cancellation scenarios: MultiTunnelManager `CancelAll`, provider goroutine cleanup, `readTunLoop` on interface close
  - Payment coin selection edge cases: insufficient funds, exact match, zero-value UTXOs, duplicate references
- [ ] **7.4 Integration Test Reliability**: Current platform integration tests (`*_integration_test.go`) require elevated privileges and specific network setup. Document prerequisites and consider adding a test mode that skips privileged operations.

---

## Platform-Specific Improvements

- [ ] **8.1 Cleanup Marker Location**: The marker files for network state recovery (`internal/tunnel/cleanup_marker_*.go`) use hardcoded paths (e.g., `/etc/blockchain-vpn-network-marker`). Move to application config directory for consistency and user-writable locations. Also add pre-restore existence checks.
- [ ] **8.2 Kill Switch Consolidation**: Review `killswitch_linux.go`, `killswitch_darwin.go`, `killswitch_windows.go` for duplication. Extract common rule management logic where possible. Audit for resource leaks to ensure iptables/network rules are restored even on partial failure.
- [ ] **8.3 Privilege Escalation Tests**: Add tests for `privilege_linux.go` (and equivalents) covering failure modes: sudo prompts without TTY, missing sudoers config, user rejection. Ensure clear error messages.

---

## Observability & Diagnostics

- [ ] **9.1 Retry Operation Metrics**: Instrument `retry.go` to expose retry count, backoff durations, and failure reasons via the metrics endpoint (`/metrics.json`). Helps diagnose connectivity issues in production.
- [ ] **9.2 Goroutine Leak Detection**: Add runtime goroutine count monitoring to `providerStatus` output. Alert if goroutine count grows unexpectedly over time (indicative of leaks in multi_tunnel or rotation loops).
- [ ] **9.3 Enhanced Logging Context**: Add structured logging (with fields) to key operations: payment monitor found payment, provider server accepting connection, scanner block progress, config validation failures. Use `log.Printf` with key=value pairs or consider structured logging library.

---

## Security

- [ ] **10.1 Metrics Auth Token Validation**: Currently `security.metrics_auth_token` has a minimum length check (12 chars) but no entropy validation. Consider requiring at least 16 chars with mixed character classes, or generate a secure token if left empty and metrics are enabled.
- [ ] **10.2 TLS Cipher Suite Agility**: The `tls_policy.go` cipher suite list for "compat" profile is static. Consider updating periodically as old ciphers are deprecated. Add a way to override via config without code change.
- [ ] **10.3 Provider Key Rotation Audit**: The `rotate-provider-key` command generates new key but what happens to old subscriptions? Document whether old payments remain valid, or if rotation invalidates existing client certificates. Add explicit revocation of old key if needed.

---

## Questions for User:

1. Which priorities should I focus on first? (The plan proposes starting with GUI validation highlighting)
2. Are there any constraints? (e.g., "don't touch scanner.go", "keep changes localized")
3. Should I preserve backward compatibility for any of these changes? (config defaults, DNS server changes)
4. Do any items depend on others? (e.g., validation highlighting depends on no breaking changes to GUI)
5. Are there any existing tests I might break with these changes?
