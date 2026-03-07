# BlockchainVPN Implementation Plan

This document outlines the prioritized implementation plan based on code analysis.

---

## PHASE 2: HIGH PRIORITY SECURITY ✅ DONE

### 2.1 RPC Password Stored Plaintext ✅ DONE
**Location:** `internal/config/config.go:25`  
**Issue:** Password in plaintext JSON config file.  
**Fix:** 
- Document localhost-only RPC setup (no password required)
- Add security warnings to INSTALL.md
- Update default config to localhost:25173
- Auto-detect localhost and disable TLS/password

---

## PHASE 3: HIGH PRIORITY BUGS

### 3.4 Payment Amount Not Verified ✅ DONE
**Location:** `internal/blockchain/payment.go:146-234`  
**Issue:** No check that paid amount >= provider's advertised price before sending.  
**Fix:** Added client-side verification and provider-side verification of actual transaction outputs.

---

## PHASE 4: MEDIUM PRIORITY ✅ DONE

### 4.1 Port Conflict Detection ✅ DONE
**Issue:** No warning when provider+client ports conflict on same machine.  
**Fix:** Added port conflict detection and auto-rotation utility functions.

### 4.2 Provider Offline Mid-Session ✅ DONE
**Issue:** No handling when provider disconnects unexpectedly.  
**Fix:** Added CreditManager for pay-as-you-go auto-recharge system.

### 4.5 Certificate Pinning Missing ✅ DONE
**Issue:** No pinning between sessions for known providers.  
**Fix:** Added certificate fingerprint to heartbeat and on-chain announcement.

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

## REMAINING SUMMARY

| Phase | Items | Status |
|-------|-------|--------|
| 2 | 2.1 | ✅ DONE |
| 3 | 3.4 | ✅ DONE |
| 4 | 4.1, 4.2, 4.5 | ✅ DONE |
| 5 | 5.1-5.8 | ❌ Not started |
| 6 | 6.1-6.6 | ❌ Not started |
| 7 | 7.1-7.2 | ❌ Not started |

**Remaining Issues:** 16
