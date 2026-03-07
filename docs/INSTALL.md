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

## 5. Verify Runtime Status

Use `status` to inspect config/runtime readiness:

```bash
./bcvpn status
./bcvpn status --json
```

`--json` output is intended for automation and CI checks.
