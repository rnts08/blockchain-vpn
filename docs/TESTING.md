# Testing and Demo Workflow

This document describes how to set up a local regtest OrdexCoin environment for testing BlockchainVPN in dry-run mode without real funds.

## Overview

BlockchainVPN includes a mock RPC server that simulates a blockchain node for testing. This allows you to:

- Test provider announcements without broadcasting real transactions
- Test client scanning without scanning the real blockchain
- Test configuration validation and command parsing
- Debug tunnel setup without network changes

## Starting the Mock RPC Server

The mock RPC server is located at `cmd/mock-rpc/main.go`. Build and run it:

```bash
go build -o mock-rpc ./cmd/mock-rpc
./mock-rpc --listen localhost:18443 --network regtest --verbose
```

Options:
- `--listen`: Address to listen on (default: `localhost:18443`)
- `--network`: Network name (default: `regtest`)
- `--v`: Enable verbose logging

The server will:
- Start with a pre-mined block height of 100
- Support standard RPC methods (getblockcount, getblockhash, getblock, sendrawtransaction, etc.)
- Generates deterministic test addresses and transactions
- Simulates blockchain state in memory (no persistence)

## Configuring BlockchainVPN for Dry-Run Mode

Create a config that points to the mock RPC:

```bash
bcvpn generate-config
bcvpn config set rpc.host localhost:18443
bcvpn config set rpc.user any
bcvpn config set rpc.pass any
bcvpn config set rpc.network regtest
```

### Client Mode

```bash
bcvpn config set client.interface_name bcvpn1
bcvpn config set client.tun_ip 10.10.0.2
bcvpn config set client.tun_subnet 24
```

### Provider Mode

```bash
bcvpn config set provider.interface_name bcvpn0
bcvpn config set provider.listen_port 51820
bcvpn config set provider.announce_ip 127.0.0.1
bcvpn config set provider.price 1000
bcvpn config set provider.tun_ip 10.10.0.1
bcvpn config set provider.tun_subnet 24
bcvpn config set provider.private_key_file ~/.config/blockchain-vpn/provider.key
```

## Testing with Dry-Run Mode

The `--dry-run` flag allows you to simulate operations without making changes:

```bash
# Test provider announcement (no transaction sent)
bcvpn start-provider --dry-run

# Test rebroadcast
bcvpn rebroadcast --dry-run

# Test key generation (no files written)
bcvpn generate-provider-key --dry-run

# Test scanning (will use mock blockchain data)
bcvpn scan --dry-run
```

## Full End-to-End Test in Regtest

For a full integration test (without dry-run), you can:

1. Start the mock RPC
2. Generate provider key: `bcvpn generate-provider-key`
3. Start provider: `bcvpn start-provider`
   - Provider will broadcast a service announcement to the mock chain
4. In another terminal, scan: `bcvpn scan`
   - You should see the provider you just announced
5. Connect (as client): `bcvpn connect <provider-signature>`
   - This will handle payment negotiation and tunnel setup (but won't actually create a working tunnel without elevated privileges and proper network setup)

## Expected Behavior

- The mock RPC returns synthetic block hashes and heights
- GetNewAddress returns test addresses like `regtest:q...`
- SendRawTransaction simulates acceptance and returns a fake txid
- Scanning will "find" announcements from the in-memory blocks

## Notes

- The mock RPC does not validate signatures or scripts; it simply echoes back transactions
- No actual coins exist; all balance checks are bypassed
- The server is single-purpose for testing and should not be used on mainnet

## Troubleshooting

If you get connection errors:
- Ensure the mock RPC is running and listening on the configured port
- Check that `rpc.host` in config matches the mock server address
- Use `--verbose` on mock-rpc to see incoming requests
- Check firewall rules that might block localhost connections
