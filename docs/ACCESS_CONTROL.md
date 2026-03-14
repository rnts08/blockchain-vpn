# Access Control Mechanisms

This document describes the access control implementation in BlockchainVPN.

## Overview

BlockchainVPN implements access control through allowlists and denylists, allowing providers to restrict who can connect to their VPN service.

## Core Components

### AccessPolicy (`internal/tunnel/access_policy.go`)

Controls client access based on public key allowlists/denylists.

**Structure:**
```go
type accessPolicy struct {
    allow map[string]bool  // Allowed public keys (hex-encoded)
    deny  map[string]bool // Denied public keys (hex-encoded)
}
```

## Access Rules

The access policy enforces these rules in order:

1. **Denylist check first**: If key is in denylist, access is denied
2. **Allowlist check second**: If allowlist is non-empty and key is not in allowlist, access is denied
3. **Default allow**: If both lists are empty, all connections are allowed

## Configuration

### Loading Policy

Load access policy from files:

```go
policy, err := loadAccessPolicy("/path/to/allowlist.txt", "/path/to/denylist.txt")
```

### File Format

Each file contains one public key per line (hex-encoded):

```
# allowlist.txt
02a1b3c5d7e9f...
03b2d4f6a8c0e...
# Comments start with #
```

## Checking Access

```go
err := policy.check(clientPublicKey)
if err != nil {
    // Access denied
}
```

## Implementation Details

### Key Encoding

Public keys are serialized as compressed secp256k1 (33 bytes) and hex-encoded:

```go
key := hex.EncodeToString(peer.SerializeCompressed())
```

### Error Messages

- Denied key: `"peer key denied by denylist"`
- Not in allowlist: `"peer key not present in allowlist"`

## Usage Scenarios

### Allowlist Mode

Only specified clients can connect:

```
allowlist.txt:  [client1_pubkey]
denylist.txt:   (empty)
```

Result: Only client1 can connect.

### Denylist Mode

Block specific clients:

```
allowlist.txt:  (empty)
denylist.txt:   [bad_actor_pubkey]
```

Result: Bad actor is blocked, everyone else allowed.

### Combined Mode

Allow some, deny others:

```
allowlist.txt:  [trusted_client1, trusted_client2]
denylist.txt:   [banned_client]
```

Result:
- trusted_client1: ALLOWED
- trusted_client2: ALLOWED
- banned_client: DENIED
- unknown_client: DENIED (not in allowlist)

## Integration

Access policy is checked during tunnel establishment:

```go
func ConnectToProvider(ctx context.Context, cfg *config.ClientConfig, ...) error {
    // ... TLS handshake ...
    
    // Check access policy
    if err := policy.check(peerPubKey); err != nil {
        return fmt.Errorf("access denied: %w", err)
    }
    
    // ... continue with tunnel setup ...
}
```

## Security Considerations

1. **Key management**: Keep allowlist/denylist files secure
2. **File permissions**: Restrict file read access (chmod 600)
3. **Updates**: Reload policy periodically for changes
4. **Monitoring**: Log access denied events for audit

## Performance

- Access checks are O(1) map lookups
- Policy is loaded once at startup
- No blockchain queries required for access control
