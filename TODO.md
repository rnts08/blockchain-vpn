# BlockchainVPN Implementation Plan

This document outlines the prioritized implementation plan based on code analysis.

---

## PHASE 1: CRITICAL (Fundamental Safety & Financial Issues) ✅ DONE

### 1.1 Payment Before Verification ✅ DONE
**Location:** `cmd/bcvpn-gui/main.go:1588-1615`  
**Issue:** Client sends payment BEFORE verifying provider is still online. Risk: fund loss if provider goes offline between scan and payment.  
**Fix:** Add liveness check (ping/echo) before payment, or implement reservation system.

### 1.2 Self-Connection Not Prevented ✅ DONE
**Location:** `cmd/bcvpn-gui/main.go:1560-1616`  
**Issue:** No check to prevent connecting to own provider (same IP:port). Creates routing loop.  
**Fix:** Add self-connection validation comparing provider endpoint to local announced IP.

### 1.3 RPC TLS Disabled ✅ DONE
**Location:** `cmd/bcvpn-gui/main.go:362`, `internal/blockchain/provider.go:362`  
**Issue:** RPC credentials sent in plaintext (DisableTLS: true).  
**Fix:** Enable TLS by default, or document that RPC must be on localhost/secured network.

---

## PHASE 2: HIGH PRIORITY SECURITY ✅ DONE

### 2.1 RPC Password Stored Plaintext ⚠️ PARTIAL
**Location:** `internal/config/config.go:25`  
**Issue:** Password in plaintext JSON config file.  
**Fix:** Use OS keychain (keychain/libsecret/dpapi) by default, warn if file mode used. (GUI now defaults to TLS which helps, but password storage still plaintext - needs keychain integration)

### 2.2 Metrics Endpoint Unauthenticated ✅ DONE
**Location:** `cmd/bcvpn-gui/main.go:946-950`  
**Issue:** Doctor warns but allows metrics without auth token.  
**Fix:** Require auth token when enabling metrics, block save without token.

### 2.3 Logging Sensitive Data ✅ DONE
**Location:** `internal/blockchain/provider.go:34`  
**Issue:** `log.Printf("Created payload: %x\n", payload)` exposes raw tx data.  
**Fix:** Remove or redact payload from logs.

---

## PHASE 3: HIGH PRIORITY BUGS (Core Functionality) ✅ DONE

### 3.1 IPv6 Packets Silently Dropped ✅ DONE
**Location:** `internal/tunnel/tunnel.go:436-453`  
**Issue:** readTunLoop only handles IPv4, IPv6 packets dropped silently.  
**Fix:** Add IPv6 handling in readTunLoop (check version 6, parse destination IP).

### 3.2 Race Condition in Capacity Check ✅ DONE
**Location:** `internal/tunnel/tunnel.go:330-340`  
**Issue:** MaxConsumers check has race window allowing over-limit connections.  
**Fix:** Use atomic counter or move check+accept inside locked section.

### 3.3 DNS Leak Check Doesn't Block ✅ DONE
**Location:** `internal/tunnel/client_security_checks.go:129-137`  
**Issue:** strict=true doesn't actually block on DNS leak detection.  
**Fix:** Return error when strict=true and leak detected.

### 3.4 Payment Amount Not Verified ⚠️ PARTIAL
**Location:** `internal/blockchain/payment.go:146-234`  
**Issue:** No check that paid amount >= provider's advertised price before sending.  
**Fix:** Added zero price check in GUI before payment, but payment.go itself doesn't verify

### 3.5 Error Handling Ignored ✅ DONE
**Location:** `internal/blockchain/payment.go:206-207`  
**Issue:** `changeAddr, _` and `changeScript, _` ignore errors.  
**Fix:** Handle errors properly.

### 3.6 IPv4 Packet Bounds Check Missing ✅ DONE
**Location:** `internal/tunnel/tunnel.go:437`  
**Issue:** Reads packet[16:20] without verifying length >= 20. Can panic on short packets.  
**Fix:** Add `if n < 20` check before parsing.

---

## PHASE 4: MEDIUM PRIORITY

### 4.1 Port Conflict Detection
**Issue:** No warning when provider+client ports conflict on same machine.  
**Fix:** Add detection and modal warning.

### 4.2 Provider Offline Mid-Session
**Issue:** No handling when provider disconnects unexpectedly.  
**Fix:** Implement reconnection logic with exponential backoff.

### 4.3 Session Timer Leak ✅ DONE
**Location:** `internal/tunnel/tunnel.go:399-412`  
**Issue:** Timer not stopped in all code paths.  
**Fix:** Use `defer timer.Stop()` or `case` with `default: timer.Stop()`.

### 4.4 TLS Handshake Before State Access ✅ DONE
**Location:** `internal/tunnel/tunnel.go:271`  
**Issue:** Accessing ConnectionState before handshake completes.  
**Fix:** Call Handshake() or defer state access.

### 4.5 Certificate Pinning Missing
**Issue:** No pinning between sessions for known providers.  
**Fix:** Store first-seen cert hash, warn on subsequent changes.

### 4.6 Zero Price Provider Handling ✅ DONE
**Issue:** No policy for free (price=0) providers.  
**Fix:** Added zero price check in GUI - rejects connection with error.

---

## PHASE 5: GUI/UX IMPROVEMENTS

### 5.1 Status Label Not Reflecting Actual State
**Location:** `cmd/bcvpn-gui/main.go:425,520,525`  
**Issue:** Shows "running" immediately without verification.  
**Fix:** Subscribe to provider state events, update label accordingly.

### 5.2 No Progress Indicators
**Issue:** Long operations (scan, connect) show no feedback.  
**Fix:** Add loading spinners/progress bars.

### 5.3 Log Panel Improvements
**Issue:** No auto-scroll, search, or export.  
**Fix:** Add auto-scroll toggle, search, export button.

### 5.4 Confirmation for Destructive Actions
**Issue:** Stop Provider, Disconnect All have no confirmation.  
**Fix:** Add confirmation dialogs.

### 5.5 Real-time Metrics
**Issue:** Manual refresh only.  
**Fix:** Add auto-refresh (5s interval) or live charts.

### 5.6 Wallet Balance Display
**Issue:** History shown but not current balance.  
**Fix:** Add balance query and display.

### 5.7 Country Filter Dropdown
**Issue:** Free-text entry instead of dropdown.  
**Fix:** Add country code dropdown with search.

### 5.8 Input Validation Highlighting
**Issue:** Error dialogs don't highlight invalid fields.  
**Fix:** Add field border color on validation failure.

---

## PHASE 6: LOW PRIORITY / NICE TO HAVE

### 6.1 Wallet Transaction Detection
**Location:** `internal/blockchain/provider.go:310-313`  
**Issue:** Doesn't verify tx is to wallet's own addresses.  
**Fix:** Check against wallet addresses.

### 6.2 Concurrent Provider Start Race
**Location:** `cmd/bcvpn-gui/main.go:1255-1355`  
**Issue:** Potential race on provider start/stop.  
**Fix:** Additional state locking.

### 6.3 Throughput Port Hardcoded
**Location:** `internal/tunnel/client_security_checks.go:185`  
**Issue:** Port 51821 hardcoded.  
**Fix:** Extract from provider endpoint data.

### 6.4 Heartbeat Interval
**Location:** `cmd/bcvpn-gui/main.go:1338`  
**Issue:** 5 minutes may be too long.  
**Fix:** Make configurable, consider 1 minute default.

### 6.5 Provider Scan Efficiency
**Location:** `cmd/bcvpn-gui/main.go:1427-1450`  
**Issue:** Loads all then filters in-memory.  
**Fix:** Server-side filtering if supported.

### 6.6 Payment Monitor Polling
**Location:** `internal/blockchain/provider.go:274`  
**Issue:** 1-minute interval slow.  
**Fix:** Make configurable, use wallet notifications.

---

## PHASE 7: TESTING

### 7.1 GUI Integration Tests
**Issue:** No UI automation tests.  
**Fix:** Add fyne-test or similar framework.

### 7.2 Fuzz Test Coverage
**Issue:** Protocol fuzz tests may miss edge cases.  
**Fix:** Expand coverage, add corpus.

---

## IMPLEMENTATION ORDER SUMMARY

| Phase | Items | Status |
|-------|-------|--------|
| 1 | 1.1-1.3 | ✅ DONE |
| 2 | 2.1-2.3 | ✅ DONE (2.1 partial) |
| 3 | 3.1-3.6 | ✅ DONE (3.4 partial) |
| 4 | 4.1-4.6 | ⚠️ 4.3, 4.4, 4.6 DONE |
| 5 | 5.1-5.8 | ❌ Not started |
| 6 | 6.1-6.6 | ❌ Not started |
| 7 | 7.1-7.2 | ❌ Not started |

**Completed:** ~17 issues fixed
**Remaining:** ~20 issues
