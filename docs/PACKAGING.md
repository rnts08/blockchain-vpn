# Packaging and Installer Guide

## 1. Build Artifacts

CLI:

```bash
make build-cli-all
```

Produces:

- `bcvpn-linux-amd64`
- `bcvpn-darwin-amd64`
- `bcvpn-windows-amd64.exe`

GUI:

```bash
make build-gui
```

Build GUI natively on each target OS for reliable OpenGL/cgo linkage.

## 2. Linux Packaging

Suggested outputs:

- `.deb` / `.rpm` package for CLI
- tarball for GUI binary + desktop file

Include:

- `bcvpn` or `bcvpn-gui`
- docs (`INSTALL.md`, `USER_GUIDE.md`, `TROUBLESHOOTING.md`)

Post-install:

- Document privilege model (root/sudo for networking actions).

## 3. macOS Packaging

Suggested outputs:

- Signed `.app` bundle for GUI
- optional CLI in installer payload
- `.pkg` installer for managed deployments

Post-install:

- Ensure application can request admin privileges for networking setup.

## 4. Windows Packaging

Suggested outputs:

- `.msi` installer (preferred for enterprise deployment)
- signed `.exe` installer for direct distribution

Post-install:

- Add Start Menu entries for GUI.
- Document Run as Administrator requirement for networking actions.

## 5. Release Checklist

1. `go test ./...`
2. `make build-cli-all`
3. Native GUI build per OS
4. Smoke test provider/client flows
5. Verify `bcvpn status --json`
6. Publish checksums and release notes
