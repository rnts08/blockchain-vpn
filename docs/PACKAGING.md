# Packaging and Installer Guide

## 1. Build Artifacts

Build CLI for all platforms:

```bash
make build-cli-all
```

Produces:

- `bcvpn-linux-amd64`
- `bcvpn-darwin-amd64`
- `bcvpn-windows-amd64.exe`

## 2. Linux Packaging

Suggested outputs:

- `.deb` / `.rpm` package
- tarball with binary and docs

Include:

- `bcvpn`
- docs (`INSTALL.md`, `USER_GUIDE.md`, `TROUBLESHOOTING.md`)

Post-install:

- Document privilege model (root/sudo for networking actions).

## 3. macOS Packaging

Suggested outputs:

- CLI binary in `/usr/local/bin/` or via Homebrew
- `.pkg` installer for managed deployments

Post-install:

- Ensure application can request admin privileges for networking setup.

## 4. Windows Packaging

Suggested outputs:

- `.msi` installer (preferred for enterprise deployment)
- signed `.exe` installer for direct distribution

Post-install:

- Document Run as Administrator requirement for networking actions.

## 5. Release Checklist

1. `make test`
2. `make build-cli-all`
3. Run smoke tests
4. Verify `bcvpn status --json` and `bcvpn doctor --json`
5. Publish checksums and release notes
