# Networking Automation Guide

This document describes what BlockchainVPN configures automatically at runtime so users do not need to apply manual route/DNS/TUN steps.

## 1. Client Networking (Automatic)

When a client connects to a provider, BlockchainVPN automatically:

1. Creates and configures the client TUN interface.
2. Adds split default routes (`0.0.0.0/1` and `128.0.0.0/1`) through the TUN interface.
3. Adds a direct host route for the provider endpoint outside the tunnel to prevent control-channel loops.
4. Applies DNS servers for leak-resistant resolution.
5. Restores previous route and DNS state when the session ends.

Platform backends:

- Linux: `netlink` + `/etc/resolv.conf` backup/restore.
- macOS: `ifconfig`, `route`, `networksetup`.
- Windows: `netsh`, `route`, PowerShell DNS APIs.

## 2. Provider Networking (Automatic)

When provider mode starts, BlockchainVPN automatically:

1. Creates/configures provider TUN interface.
2. Starts TLS listener and UDP echo service.
3. Optionally enables NAT traversal (UPnP/NAT-PMP) for home routers.
4. Optionally enables provider egress NAT backend (Linux/macOS/Windows backends).
5. Runs active health checks for TUN and listener.

## 3. Privilege Requirements

Automatic networking changes require elevated privileges:

- Linux: root (or equivalent networking capabilities)
- macOS: administrator/sudo
- Windows: elevated Administrator terminal

The application now preflights privileges before provider start and before client payment/connection. If elevation is missing, the operation fails early with a clear error.

## 4. No-Manual-Step Operation

For Linux, macOS, and Windows client mode, route/DNS/TUN changes are automated by runtime backends.

For provider mode:

- Core provider networking is automated across Linux/macOS/Windows.
- Provider egress NAT backend is available on Linux/macOS/Windows.

## 5. Verification

Use:

```bash
./bcvpn status
./bcvpn status --json
```

`status` reports privilege readiness and key runtime settings required for automated networking.
