# Versioning Policy

BlockchainVPN uses semantic-style versioning with current baseline:

- `0.1.0`

## Rules for this project phase

- Patch (`0.1.x`): minor fixes, small QoL, docs-only updates, non-breaking behavior adjustments.
- Minor (`0.x.0`): major fixes/features that materially improve reliability/security/runtime behavior but do not require large-scale refactors.
- Major (`1.0.0` and above): stable production release after major refactor-risk items are closed and no high-priority improvement backlog remains.

## Runtime Version Metadata

CLI exposes version metadata:

```bash
./bcvpn version
./bcvpn version --json
```

Build metadata can be overridden at build time using `-ldflags`:

```bash
go build -ldflags "-X blockchain-vpn/internal/version.GitCommit=$(git rev-parse --short HEAD) -X blockchain-vpn/internal/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o bcvpn ./cmd/bcvpn
```
