# BlockchainVPN TODO

## 1. Tunnel/TLS Core
- [x] Remove legacy interface-manager dependency and rely on TLS-over-TUN transport.
- [x] Bind client trust to on-chain provider pubkey during TLS handshake.
- [ ] Add integration tests for provider/client handshake and packet forwarding.
- [ ] Add per-session throughput accounting and enforcement.

## 2. Cross-Platform Networking
- [x] Linux automatic TUN address configuration.
- [x] Linux automatic route and DNS setup/restore.
- [x] Non-Linux stubs with clear runtime warnings.
- [ ] Implement macOS backend for TUN address, routing, and DNS.
- [ ] Implement Windows backend for TUN address, routing, and DNS.

## 3. Provider Networking
- [ ] Implement provider egress/NAT backend by platform.
- [ ] Add optional provider namespace/sandbox isolation mode.
- [ ] Add active health checks for TUN interface and listener.

## 4. Blockchain and Payments
- [x] Service announcement, discovery, payment payloads.
- [x] Price update announcements.
- [ ] Replace naive UTXO selection with deterministic coin selection.
- [ ] Improve payment monitor reorg handling with tx->peer index.
- [ ] Add retry and backoff strategy for RPC connectivity loss.

## 5. Security and Hardening
- [x] Encrypt provider private key at rest.
- [x] Strict OP_RETURN payload decoding.
- [ ] Add cert lifetime tuning and key rotation workflow.
- [ ] Add optional allowlist/denylist for provider access policies.

## 6. Product Surface
- [ ] Implement GUI app based on `docs/GUI.md`.
- [ ] Add machine-readable status output for automation (`--json` mode).
- [ ] Add installation and OS-specific privilege setup guides.
