# Provider Operations Runbook

## 1. Startup Checklist

1. Validate config:
   - `./bcvpn config validate`
2. Check runtime readiness:
   - `./bcvpn status --json`
3. Confirm privileges/elevation are available.
4. Confirm RPC node health and wallet balance.

## 2. Key Rotation Workflow

1. Rotate provider key:
   - `./bcvpn rotate-provider-key`
2. Re-broadcast service announcement:
   - `./bcvpn rebroadcast`
3. Confirm new provider identity is visible in scans.
4. Archive old backups and enforce secure retention policy.

## 3. Revocation Workflow

1. Add revoked pubkeys (hex compressed key, one per line) to `security.revocation_cache_file`.
2. Save config and verify:
   - `./bcvpn config validate`
3. Confirm rejected connections in logs/status metrics.
4. Keep revocation file under controlled permissions.

## 4. Incident Response

When compromise is suspected:

1. Stop provider immediately.
2. Rotate provider key.
3. Revoke suspicious client keys.
4. Review payment history and auth logs.
5. Re-announce service with fresh identity.

## 5. Upgrade Strategy

1. Capture pre-upgrade diagnostics:
   - `./bcvpn status --json`
2. Backup config directory (`config.json`, key material, history).
3. Upgrade binaries.
4. Run validation + status checks.
5. Re-enable provider and verify latency/traffic path.
