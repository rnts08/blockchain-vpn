#!/usr/bin/env bash
set -euo pipefail

echo "[smoke] building CLI for Linux/macOS/Windows"
GOOS=linux GOARCH=amd64 go build -o dist/bcvpn-linux-amd64 ./cmd/bcvpn
GOOS=darwin GOARCH=amd64 go build -o dist/bcvpn-darwin-amd64 ./cmd/bcvpn
GOOS=windows GOARCH=amd64 go build -o dist/bcvpn-windows-amd64.exe ./cmd/bcvpn

echo "[smoke] running local CLI metadata checks"
go build -o dist/bcvpn ./cmd/bcvpn
./dist/bcvpn version
./dist/bcvpn version --json
./dist/bcvpn doctor --json >/dev/null

echo "[smoke] building GUI (native host only)"
go build -o dist/bcvpn-gui ./cmd/bcvpn-gui

echo "[smoke] completed"
