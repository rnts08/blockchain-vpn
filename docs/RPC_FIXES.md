# RPC Implementation Discrepancies

## Summary
The BlockchainVPN application's RPC client does not fully adhere to the Ordexcoin RPC protocol specification as documented in `docs/ordexcoin_rpc_info.md`. The main issues relate to authentication methods, TLS support, and server readiness handling.

## Detailed Discrepancies

### 1. Missing Cookie-Based Authentication
**Specification:** Cookie-based authentication is the default and recommended method. The cookie file is located at `<datadir>/.cookie` and is used automatically.

**Actual Implementation:** The application only supports username/password authentication via HTTP Basic Auth. There is no code to read or use the `.cookie` file.

**Impact:**
- Insecure default configuration (username "rpcuser", empty password)
- Deviates from recommended security practices
- Requires manual credential configuration instead of automatic cookie auth

**Code References:**
- `cmd/bcvpn/main.go:635-648` - `connectRPCWithConfig` uses only `User`/`Pass`
- `internal/config/config.go:22-27` - `RPCConfig` lacks cookie-related fields
- No references to `.cookie` or `rpccookiefile` found in codebase

### 2. Missing HMAC-SHA-256 Authentication Support
**Specification:** Ordexcoin supports HMAC-SHA-256 authentication via the `-rpcauth` option (format `USERNAME:SALT$HASH`).

**Actual Implementation:** No support for HMAC-SHA-256 auth. Only Basic Auth is available.

**Impact:** Cannot use the more secure HMAC-based authentication method.

### 3. Non-Standard TLS Option
**Specification:** "Transport: HTTP/HTTPS (HTTP only, no SSL/TLS encryption)" and "RPC does not use SSL".

**Actual Implementation:** The configuration includes an `EnableTLS` boolean flag, and the client will use TLS if enabled.

**Impact:**
- Misleading option that suggests Ordexcoin Core supports native TLS, which it does not
- May lead users to attempt TLS configuration that will fail with standard Ordexcoin nodes
- Should either be removed or clearly documented as for use with TLS-terminating proxies

**Code References:**
- `internal/config/config.go:26` - `EnableTLS bool` field
- `cmd/bcvpn/main.go:641` - `DisableTLS: !cfg.RPC.EnableTLS`
- `cmd/bcvpn-gui/main.go:389` - same usage

### 4. No Server Warmup Handling
**Specification:** "RPC runs in 'warmup' mode during initial startup - accepts connections but rejects RPC calls until ready. Query warmup status with `getrpcinfo` RPC."

**Actual Implementation:** The application immediately makes RPC calls after connecting without checking if the server is ready.

**Impact:** May experience "Server in warmup" errors during node startup; no automatic retry or wait mechanism.

**Code References:**
- `cmd/bcvpn/main.go:643` - creates client and immediately uses it
- No calls to `getrpcinfo` to check warmup status

### 5. Network-Specific Port Defaults Not Documented
**Specification:** Default RPC ports differ by network:
- Mainnet: 25173
- Testnet: 35173
- Signet: 325173
- Regtest: 18443

**Actual Implementation:** The default configuration uses `localhost:25173` (mainnet). The config requires manual host:port specification with no guidance or network-specific presets.

**Impact:** Users running on testnet/signet/regtest must manually look up and set the correct port; the application does not provide network selection to automatically configure the appropriate default.

**Code References:**
- `internal/config/config.go:182` - default host is `localhost:25173`
- README.md example shows `localhost:18443` (regtest), creating inconsistency

### 6. No Support for RPC Whitelisting Features
**Specification:** The server supports `-rpcwhitelist` and `-rpcwhitelistdefault` for per-user method restrictions.

**Actual Implementation:** Not applicable (client-side only), but the application does not document or configure these server-side requirements for optimal operation.

**Impact:** Users may not be aware of additional server-side access control features that could enhance security.

## Correctly Implemented Features
- HTTP POST mode is correctly enabled (`HTTPPostMode: true`)
- JSON-RPC 2.0 format is handled by the `rpcclient` library
- Content-Type is set to `application/json` by the library

## Recommendations
1. Add cookie-based authentication support (read `.cookie` file, fallback to username/password)
2. Remove or clearly document the `EnableTLS` option as proxy-only
3. Implement warmup detection using `getrpcinfo` with retry/backoff
4. Provide network selection (mainnet/testnet/signet/regtest) to auto-configure default ports
5. Consider adding HMAC-SHA-256 auth if the underlying library supports it
6. Update default configuration to use more secure defaults (e.g., require explicit password or use cookie)
