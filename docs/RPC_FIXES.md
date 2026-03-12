# RPC Implementation Status

> **Note**: This document was previously tracking RPC implementation discrepancies. Many issues have been resolved. This document now serves as a record of what was fixed.

## Resolved Issues

### 1. Cookie-Based Authentication - RESOLVED
The application now supports cookie-based authentication via the `rpc.cookie_file` config field.

**Code References:**
- `internal/config/config.go:32` - `CookieFile` field in RPCConfig
- `cmd/bcvpn/main.go:743` - Uses cookie file if provided

### 2. Insecure Default Configuration - RESOLVED
The default config now generates a secure random RPC password instead of using an empty password.

**Code References:**
- `internal/config/config.go:185-195` - `GenerateRandomRPCPassword()` function
- Default config now includes a 64-character hex password

### 3. Network-Specific Configuration - RESOLVED
The application now supports network selection (mainnet, testnet, regtest, signet) which automatically configures appropriate defaults.

**Code References:**
- `internal/config/config.go:28` - `Network` field in RPCConfig
- `internal/config/config.go:207-208` - Default network is "mainnet"

### 4. Server Warmup Handling - RESOLVED
The RPC client now includes warmup detection and retry logic.

**Code References:**
- `cmd/bcvpn/main.go` - `waitForServerReady` function
- Uses retry logic with exponential backoff

## Remaining Considerations

### HMAC-SHA-256 Authentication
Not implemented. The application uses HTTP Basic Auth or cookie-based authentication.

### TLS Option
The `EnableTLS` option remains for use with TLS-terminating proxies. Ordexcoin Core itself does not support native TLS for RPC.

## Historical

See git history for previous state of this document (commit messages reference "RPC fixes").
