# Test Coverage Analysis

This document provides a comprehensive analysis of the test coverage for the BlockchainVPN project, identifying gaps and recommendations for improving test completeness.

## Summary

- **Unit tests:** Present for ~60% of packages
- **Integration tests:** Present for network, TLS, client security, and data transfer (platform-specific)
- **Fuzz tests:** Protocol decoders, revocation cache
- **GUI tests:** Basic component creation tests
- **Missing coverage:** Auth, geoip, history, nat, obs, usage metering, spending manager, session management, core payment flow, provider operations, and full E2E scenarios

---

## 1. Unit Test Coverage by Package

### ✅ Well-Covered Packages

| Package | Coverage | Notes |
|---------|----------|-------|
| `internal/config` | High | Validation logic fully tested |
| `internal/crypto` | High | Keystore tests with mocks |
| `internal/protocol` | High | V1/V2 payloads, heartbeats, fuzzing |
| `internal/tunnel` | Very High | **Exceptionally comprehensive**: access policy, revocation cache, TLS policy/lifetime, client security, NAT traversal, multi-tunnel integration |
| `internal/util` | High | Atomic file operations |
| `internal/blockchain` | Medium-High | Scanner merge logic, payment coin selection |
| `cmd/bcvpn` | Medium | Filter endpoints test |
| `cmd/bcvpn-gui` | Medium | GUI tab creation tests (demo mode) |

### ⚠️ Partially Covered (Critical Gaps)

| Package | Coverage | Missing Areas | Priority |
|---------|----------|----------------|----------|
| `internal/blockchain/provider` | Low | Provider announcement creation, reputation store integration | High |
| `internal/blockchain/scanner` | Low | Main scanning loop, V3 payload decoding, endpoint enrichment | High |
| `internal/blockchain/payment` | Low | Transaction building, fee estimation, UTXO selection (beyond coinselect), monitoring | High |
| `internal/tunnel/session` | Low | Session lifetime, authorization renewal, cleanup | High |
| `internal/tunnel/usage` | Basic | ✅ Added unit tests in `usage_test.go` | Complete |
| `internal/tunnel/credit_manager` | Basic | ✅ Added unit tests in `credit_manager_test.go` | Complete |
| `internal/tunnel/multi_tunnel` | Low | Concurrent session management, Add/Cancel/ActiveCount | Medium |
| `internal/tunnel/transfer` | Integration only | Unit tests for packet flow, rate limiting | Medium |
| `internal/tunnel/tunnel` | Integration only | Core tunnel handshake, key derivation, packet pump | High |
| `internal/auth` | None | AuthManager data quotas, session authorization | High |
| `internal/geoip` | None | EnrichEndpoints, country lookup, latency measurement | Medium |
| `internal/history` | None | Payment history persistence, query, export | Low |
| `internal/nat` | None | UPnP/ NAT-PMP mapping, platform implementations | Medium |
| `internal/obs` | None | Logging, metrics collection, event recording | Low |

### ❌ No Test Coverage

- `internal/auth` - Security-critical auth logic
- `internal/geoip` - Geolocation enrichment
- `internal/history` - Payment history storage
- `internal/nat` - NAT traversal
- `internal/obs` - Logging and metrics
- `internal/tunnel/usage` - **New feature, unmetered**
- `internal/tunnel/credit_manager` - **New feature, unmetered**
- `internal/provider` (if separate) - Provider lifecycle
- `internal/reputation` - Reputation store
- `internal/scan_cache` - Scan result caching
- `internal/script_parser` - OP_RETURN parsing (if exists)
- `internal/throughput` - Bandwidth probes
- `internal/tls_rotation` - Certificate rotation
- `internal/transport` - Underlying transport layer
- Platform-specific: `cleanup_marker_*`, `elevation_*`, `isolation_*`, `privilege_*`, `water_params_*`

---

## 2. Integration / E2E / Functional Tests

### Existing Integration Tests (build tag: `integration`)

Located in `internal/tunnel/`:
- `client_security_integration_test.go` - Client security verification (country, bandwidth, throughput probe)
- `network_darwin_integration_test.go` - Network setup on macOS
- `network_linux_integration_test.go` - Network setup on Linux
- `network_windows_integration_test.go` - Network setup on Windows
- `tls_integration_test.go` - TLS handshake with on-chain identity
- `transfer_integration_test.go` - Data transfer through tunnel

### Existing Functional Tests (build tag: `functional`)

None explicitly with `functional` tag currently. `make test-functional` exists but targets integration-like tests.

### Missing E2E Scenarios

| Scenario | Description | Priority |
|----------|-------------|----------|
| **Full client connection flow** | Scan → select → connect → transfer data → disconnect | Critical |
| **Provider lifecycle** | Start provider → announce → heartbeat → rebroadcast → stop | Critical |
| **Payment & authorization** | Create announcement → client pays → provider checks → session grant | Critical |
| **Flexible billing** | Time-based: incremental payments as session runs<br>Data-based: tiered payments as data transfers | Critical |
| **Spending limits** | Configure daily limit → reach warning → auto-disconnect | High |
| **Auto-recharge** | Low credit triggers top-up automatically | High |
| **Multi-tunnel** | Connect to multiple providers simultaneously | High |
| **Demo mode GUI** | Launch GUI with `-demo`, scan/select without RPC | Medium |
| **Cross-platform network** | Test routing/DNS on all 3 platforms end-to-end | Medium |
| **Certificate rotation** | Provider rotates key, announces new cert, clients trust it | Medium |
| **Throughput probes** | Client measures actual bandwidth vs advertised | Low |
| **Geo filtering** | Country filters work correctly | Low |
| **Wallet operations** | Balance check, transaction history, coin selection | High |

---

## 3. Fuzz Testing

Present:
- `internal/protocol/fuzz_test.go` - Fuzz protocol decoders
- `internal/tunnel/revocation_cache_fuzz_test.go` - Fuzz revocation parsing

Missing:
- Fuzz config parsing (JSON/YAML)
- Fuzz payment amount calculations
- Fuzz usage meter logic

---

## 4. Test Quality and Patterns

### Good Practices Observed
- Table-driven tests (`t.Run`) in many places
- Use of test helpers and mocks (e.g., `testClock`, `mockTimer`)
- Platform-specific tests with build tags
- Parallel test execution where safe (`t.Parallel()`)
- Clear failure messages with relevant context
- Integration tests isolated from unit tests via build tags

### Areas for Improvement
- **Mock RPC client**: Create reusable mock for blockchain RPC to avoid needing real `ordexcoind` in most tests
- **Test helpers**: Extract common test setup (e.g., default config, mock wallet) into `testutil` package
- **Golden files**: For complex parser outputs (announcements, transactions)
- **Test data**: Use consistent fixtures for protocol messages
- **Race detection**: Run tests with `-race` more systematically (currently not in Makefile)
- **Coverage tooling**: Add `go tool cover` to CI, track coverage trend

---

## 5. Critical Missing Tests (Action Items)

### Immediate (Block New Features)

1. **`internal/tunnel/usage`** - Complete unit test suite
   - `TestMeterNew`, `TestRecordTime`, `TestRecordData`
   - Threshold crossing logic, timeout calculation

2. **`internal/tunnel/credit_manager`** - Complete unit test suite
   - `TestNew`, `TestRecordPayment`, `TestCheckLimit`, `TestAutoRecharge`
   - Warning thresholds, session max spending

3. **`internal/auth`** - AuthManager tests
   - Data quota tracking, `CanAuthorize` logic, expiration

4. **`internal/tunnel/session`** - Session management tests
   - Authorization renewal, expiration, cleanup

5. **`internal/blockchain/payment`** - Core payment tests (beyond coinselect)
   - `BuildAndSendPayment`, fee estimation, retry logic

6. **`internal/blockchain/scanner`** - Scanner loop tests
   - V3 payload decoding, delta scanning, filter application

7. **`internal/blockchain/provider`** - Provider operations
   - Announcement building, reputation store read/write

8. **`cmd/bcvpn`** - Expand CLI tests
   - Connect flow, provider start, spending manager creation

### Short-term (E2E Coverage)

9. **Functional test: Time-based billing cycle**
   - Provider with `pricing_method=time`, `time_unit_secs=60`
   - Client connects, stays 90s, expect 2 payments (60s + 30s)
   - Verify auth intervals and final disconnect

10. **Functional test: Data-based billing tiers**
    - Provider with `pricing_method=data`, `data_unit_bytes=1000000` (1MB)
    - Transfer 2.5MB, expect 3 payments (1M + 1M + 0.5M)
    - Verify authorization quotas

11. **Functional test: Spending limit enforced**
    - Client has daily limit 10000 sats
    - Connect to provider charging 2000 sats/session
    - After 5 sessions, expect auto-disconnect on 6th attempt

12. **Functional test: Multi-tunnel concurrent**
    - Connect 3 providers simultaneously, verify all active
    - Cancel one, verify others continue
    - Cancel all, verify cleanup

13. **Functional test: Full GUI demo mode**
    - Launch `bcvpn-gui -demo`
    - Scan, filter, select, click Connect (dry-run)
    - Verify no RPC errors in logs

### Medium-term (Quality of Life)

14. **`internal/geoip`** - Enrichment tests with mock GeoIP database
15. **`internal/history`** - CRUD operations, persistence
16. **`internal/nat`** - UPnP/NAT-PMP mock tests (requires platform mocks)
17. **`internal/obs`** - Log and metrics collectors
18. **`cmd/bcvpn-gui`** - Settings tab validation, provider tab controls

---

## 6. Test Infrastructure Improvements

### Add to Makefile

```makefile
test-race:     ## Run tests with race detector
	go test -v -race ./...

test-coverage: ## Generate coverage report
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-unit-only: ## Run only unit tests (no integration)
	go test -v ./... -short

test-integration: ## Run integration tests only
	go test -v -tags integration ./...
```

### CI/CD Enhancements

- Add coverage threshold (e.g., `-coverpkg=./... -cover 80%`)
- Run integration tests on all 3 OS runners (Ubuntu, macOS, Windows)
- Add race detector to CI (may need to reduce parallelism)
- Add fuzz tests to CI (nightly?)

### Test Helpers

Create `internal/testutil/` with:
- `mockrpc.RPCClient` implementation
- `mockgeo.GeoIPDB` stub
- `mockclock.Clock` controllable time source
- `testconfig.Default()` for consistent fixture
- `tempdir` helpers

---

## 7. Coverage Metrics (Current Estimate)

| Component | Unit | Integration | E2E | Fuzz | Status |
|-----------|------|-------------|-----|------|--------|
| Protocol | 90% | - | - | ✓ | ✅ Good |
| Tunnel Core | 40% | 30% | 0% | - | ⚠️ Partial |
| Tunnel Advanced (usage, credit) | 0% | 0% | 0% | - | ❌ Missing |
| Blockchain Scanner | 20% | 0% | 0% | - | ❌ Missing |
| Blockchain Payment | 10% | 0% | 0% | - | ❌ Missing |
| Blockchain Provider | 20% | 0% | 0% | - | ❌ Missing |
| Auth | 0% | 0% | 0% | - | ❌ Missing |
| Config | 100% | - | - | - | ✅ Complete |
| Crypto | 95% | - | - | - | ✅ Complete |
| GeoIP | 0% | 0% | 0% | - | ❌ Missing |
| History | 0% | 0% | 0% | - | ❌ Missing |
| NAT | 0% | 0% | 0% | - | ❌ Missing |
| OBS | 0% | 0% | 0% | - | ❌ Missing |
| GUI | 40% | 0% | 0% | - | ⚠️ Partial |
| CLI | 30% | 0% | 0% | - | ⚠️ Partial |

**Overall rough estimate: ~35% unit test coverage, ~20% integration coverage, <5% E2E coverage**

---

## 8. Recommendations (Prioritized)

### P0 (Critical - Before v1.0 Release)

1. Write unit tests for `usage` and `credit_manager` (core new features)
2. Write tests for payment authorization flow (`auth`, `session`, `payment`)
3. Add E2E test for flexible billing (time and data scenarios)
4. Add E2E test for spending limits enforcement
5. Add mock RPC client to enable testing without real blockchain node

### P1 (High - Improve Reliability)

6. Unit tests for `scanner` (V3 payload, filters, enrichment)
7. Unit tests for `provider` (announcement, reputation interactions)
8. Unit tests for `transfer` (packet flow, rate limiting)
9. Integration test: full client → provider connection (could use mocks for RPC)
10. Race detection in CI

### P2 (Medium - Developer Experience)

11. Test helpers package (`testutil`)
12. Coverage reporting in CI
13. Tests for `geoip`, `history`, `nat`, `obs`
14. GUI component unit tests (settings validation, dialogs)
15. CLI command tests (all subcommands)

### P3 (Low - Polish)

16. Fuzz tests expansion
17. Golden file tests for protocol encodings
18. Performance benchmarks for hot paths (`usage`, `payment`, `transfer`)
19. Property-based testing (if library added)

---

## 9. Running Tests Locally

```bash
# Unit tests (all packages)
make test

# Integration tests (platform-specific, require privileges)
make test-functional

# With race detector
make test-race

# Generate coverage (to be added)
make test-coverage
```

---

## Conclusion

The project has strong foundation in unit testing for core low-level packages (protocol, crypto, config) and excellent integration tests for platform networking. However, **critical gaps exist in the blockchain, tunnel (advanced features), and auth layers** that directly impact the new v0.5.0 features (flexible billing, spending limits, multi-chain). Immediate focus should be on writing comprehensive unit tests for `usage`, `credit_manager`, `auth`, and `payment`, followed by E2E tests covering the billing scenarios.
