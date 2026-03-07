# BlockchainVPN Troubleshooting

## 1. Permission Denied / Not Elevated

Symptoms:

- Provider start fails with privilege error.
- Client connect fails before payment with privilege warning.

Fix:

- Linux: run with `sudo` or root-equivalent privileges.
- macOS: run from admin context (`sudo`).
- Windows: run terminal as Administrator.
- GUI: use **Relaunch Elevated** in setup wizard.

## 2. TUN Interface Creation Failure

Symptoms:

- `failed to create TUN interface`

Fix:

- Ensure OS TUN driver support is present.
- Ensure elevated privileges.
- Verify interface name is not already in use.

## 3. Route/DNS Configuration Errors

Symptoms:

- Connect succeeds but traffic does not flow.
- DNS leaks or no DNS resolution.

Fix:

- Check `bcvpn status` warnings.
- Verify default route and DNS service state on host.
- Retry connection (cleanup hooks restore and reapply state).
- Disable conflicting third-party VPN/firewall tooling during diagnosis.

## 4. Provider NAT/Egress Issues

Symptoms:

- Clients connect but no internet egress.

Fix:

- Set `provider.enable_egress_nat=true`.
- Set `provider.nat_outbound_interface` explicitly if autodetect is wrong.
- Confirm host firewall/router allows forwarding/NAT behavior.

## 5. RPC Connectivity Failures

Symptoms:

- Scan/start fails with RPC errors.

Fix:

- Verify `rpc.host`, `rpc.user`, and `rpc.pass`.
- Confirm `ordexcoind` is running and synced.
- Confirm RPC server is enabled (`server=1`).

## 6. GUI Build/Run Issues

Symptoms:

- GUI fails to build cross-platform.

Fix:

- Build GUI natively on target OS.
- Keep Go/Fyne dependencies current.

## 7. Kill Switch Behavior Questions

Notes:

- Kill switch applies while session is active.
- Cleanup runs on normal disconnect.
- If host crashes, reconnect/restart to reapply normal networking cleanup.

## 8. Secure Key Storage Backend Errors

Symptoms:

- Provider key setup fails in `keychain`, `libsecret`, or `dpapi` modes.
- Status shows key storage mode unsupported.

Fix:

- Confirm backend prerequisites are installed:
  - macOS: `security`
  - Linux: `secret-tool` (libsecret tools)
  - Windows: `powershell` or `pwsh`
- Use `security.key_storage_mode=auto` for fallback behavior.
- Use `security.key_storage_mode=file` if secure-store backend is unavailable.
