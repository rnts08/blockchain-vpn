# Installation and Privilege Guide

This guide covers installation and required OS privileges for the BlockchainVPN CLI.

## 1. Build

Prerequisites:

- Go 1.25+
- A synced `ordexcoind` node with RPC enabled (`server=1`)
- Recommended: `txindex=1` for faster scanning

Build commands:

```bash
go build -o bcvpn ./cmd/bcvpn
```

Cross-platform CLI builds:

```bash
make build-cli-all
# Or manually:
GOOS=linux GOARCH=amd64 go build -o bcvpn-linux-amd64 ./cmd/bcvpn
GOOS=darwin GOARCH=amd64 go build -o bcvpn-darwin-amd64 ./cmd/bcvpn
GOOS=windows GOARCH=amd64 go build -o bcvpn-windows-amd64.exe ./cmd/bcvpn
```

## 2. Config and Data Paths

BlockchainVPN stores runtime files in the OS user config directory under `BlockchainVPN`:

- Linux: `~/.config/BlockchainVPN/`
- macOS: `~/Library/Application Support/BlockchainVPN/`
- Windows: `%AppData%\BlockchainVPN\`

Files:

- `config.json`
- `provider.key`
- `history.json`
- `session.json` (last session info for rating)
- `ratings.json` (local ratings cache)

Generate initial config:

```bash
./bcvpn generate-config
```

## 2.1 RPC Connection Security

BlockchainVPN connects to `ordexcoind` via RPC. There are two recommended configurations:

### Option A: Localhost Only (Recommended for Desktop)

Run your `ordexcoind` with RPC bound to localhost only. No password required:

```bash
# ordexcoind configuration (bitcoin.conf or ordexcoind.conf)
server=1
txindex=1
rpcbind=127.0.0.1
rpcallowip=127.0.0.1
# No rpcpassword needed when bound to localhost
```

BlockchainVPN config for localhost:

```json
{
  "rpc": {
    "host": "127.0.0.1:25173",
    "user": "youruser",
    "pass": "",
    "enable_tls": false
  }
}
```

**Security Note:** When RPC is only accessible on localhost, the password is not strictly required since only local processes can access the RPC interface.

### Option B: Remote Node (Requires Password)

If connecting to a remote `ordexcoind` node, you must use authentication:

```bash
# ordexcoind configuration (bitcoin.conf or ordexcoind.conf)
server=1
txindex=1
rpcbind=0.0.0.0
rpcallowip=your-vpn-server-ip/32
rpcpassword=your-secure-random-password
```

BlockchainVPN config for remote node:

```json
{
  "rpc": {
    "host": "your-ordexcoind-host:25173",
    "user": "rpcuser",
    "pass": "your-secure-random-password",
    "enable_tls": true
  }
}
```

**Security Warning:** Storing RPC passwords in `config.json` is a security risk. Consider:

1. Using localhost-only connections when possible
2. Using OS-specific secure credential storage (set `key_storage_mode` to `keychain` (macOS), `libsecret` (Linux), or `dpapi` (Windows))
3. Restricting RPC access with `rpcallowip` to specific IPs only
4. Enabling TLS (`enable_tls: true`) when using remote connections

### Setting Up Secure Remote Connections

For remote connections, enable TLS in BlockchainVPN:

```json
{
  "rpc": {
    "host": "your-remote-host:25173",
    "user": "rpcuser",
    "pass": "your-password",
    "enable_tls": true
  }
}
```

TLS ensures RPC credentials are encrypted in transit.

## 2.2 Secure Key Storage Backend Prerequisites

If `security.key_storage_mode` is not `file`, ensure backend tooling exists:

- macOS `keychain`: `security` CLI (built-in on standard macOS installs)
- Linux `libsecret`: `secret-tool` from libsecret (`libsecret-tools` package on many distros)
- Windows `dpapi`: `powershell` or `pwsh`

If backend prerequisites are missing, use `security.key_storage_mode=file` or `auto` (falls back to file mode).

For metrics endpoint protection, set `security.metrics_auth_token` and call `/metrics.json` with `X-BCVPN-Metrics-Token`.

## 3. Required Privileges by OS

### Linux

Required for full runtime networking features:

- Create/configure TUN device
- Add/remove routes
- Set/restore DNS resolver settings
- Provider egress NAT backend setup

Run as root or with `CAP_NET_ADMIN` capability:

```bash
sudo ./bcvpn start-provider
sudo ./bcvpn scan
```

### macOS

Required for full runtime networking features:

- Create/configure TUN interface
- Apply route changes
- Apply DNS changes (via `networksetup`)
- Provider egress NAT backend setup

Run from an administrator context when connecting or providing service:

```bash
sudo ./bcvpn start-provider
sudo ./bcvpn scan
```

### Windows

Required for full runtime networking features:

- Create/configure TUN interface
- Apply route changes
- Apply DNS settings
- Provider egress NAT backend setup

Run in an elevated terminal ("Run as Administrator"):

```powershell
.\bcvpn.exe start-provider
.\bcvpn.exe scan
```

## 4. Firewall and Router Requirements

- Provider must accept inbound TCP traffic on `provider.listen_port`.
- Provider UDP echo server uses the same configured listen port.
- If behind a home router, enable `provider.enable_nat` to use UPnP/NAT-PMP mapping.
- If running provider egress NAT, set `provider.enable_egress_nat=true` and configure `provider.nat_outbound_interface`.
- To enforce client traffic blocking outside the tunnel during a session, set `client.enable_kill_switch=true`.

## 5. Click-and-Run Behavior

CLI preflights privilege requirements before provider start and before client payment/connection. If privileges are missing, the operation is stopped before funds are spent and a clear error is shown.

Configure runtime settings using:

- CLI config subcommands: `bcvpn config get/set/validate`
- Status verification: `bcvpn status` / `bcvpn status --json`
- Logging settings: `logging.format`, `logging.level` and overrides (`BCVPN_LOG_FORMAT`, `BCVPN_LOG_LEVEL`)
- Security settings: `security.key_storage_mode`, `security.revocation_cache_file`, `security.tls_min_version`, `security.tls_profile`, `security.metrics_auth_token`

## 6. Verify Runtime Status

Use `status` to inspect config/runtime readiness:

```bash
./bcvpn status
./bcvpn status --json
./bcvpn version
./bcvpn version --json
./bcvpn doctor
./bcvpn doctor --json
```

`--json` output is intended for automation and CI checks.
