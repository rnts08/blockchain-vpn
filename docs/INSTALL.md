# Installation and Privilege Guide

This guide covers installation and required OS privileges for `bcvpn` (CLI) and `bcvpn-gui` (desktop app).

## 1. Build

Prerequisites:

- Go 1.22+
- A synced `ordexcoind` node with RPC enabled (`server=1`)
- Recommended: `txindex=1` for faster scanning

Build commands:

```bash
go build -o bcvpn ./cmd/bcvpn
go build -o bcvpn-gui ./cmd/bcvpn-gui
```

Cross-platform CLI builds:

```bash
GOOS=linux GOARCH=amd64 go build -o bcvpn-linux-amd64 ./cmd/bcvpn
GOOS=darwin GOARCH=amd64 go build -o bcvpn-darwin-amd64 ./cmd/bcvpn
GOOS=windows GOARCH=amd64 go build -o bcvpn-windows-amd64.exe ./cmd/bcvpn
```

GUI note:

- GUI cross-compilation can require target-specific OpenGL/cgo toolchains.
- Build `cmd/bcvpn-gui` natively on the target OS for the most reliable result.

## 2. Config and Data Paths

BlockchainVPN stores runtime files in the OS user config directory under `BlockchainVPN`:

- Linux: `~/.config/BlockchainVPN/`
- macOS: `~/Library/Application Support/BlockchainVPN/`
- Windows: `%AppData%\BlockchainVPN\`

Files:

- `config.json`
- `provider.key`
- `history.json`

Generate initial config:

```bash
./bcvpn generate-config
```

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

- GUI and CLI now preflight privilege requirements before provider start and before client payment/connection.
- If privileges are missing, the operation is stopped before funds are spent and a clear error is shown.
- GUI first-run wizard performs config, RPC, key, and privilege checks before loading the main tabs.
- GUI includes a platform-specific "Relaunch Elevated" action to request admin/root rights and reopen automatically.
- Configure runtime settings in:
  - GUI provider/client panels (save settings to `config.json`)
  - CLI config subcommands (`bcvpn config get/set/validate`) plus `bcvpn status` / `bcvpn status --json` for verification
  - Security settings (`security.key_storage_mode`, `security.revocation_cache_file`, `security.tls_min_version`, `security.tls_profile`)

## 6. Verify Runtime Status

Use `status` to inspect config/runtime readiness:

```bash
./bcvpn status
./bcvpn status --json
```

`--json` output is intended for automation and CI checks.
