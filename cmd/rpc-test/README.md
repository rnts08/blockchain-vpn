# RPC Test Script

Simple utility to test connectivity to an OrdexCoin RPC server.

## Usage

```bash
# Build and run
go run ./cmd/rpc-test/main.go -host localhost:25174 -user rpcuser -pass rpcpass

# Build binary
go build -o rpc-test ./cmd/rpc-test

# Run with specific command
./rpc-test -host localhost:25174 -user rpcuser -pass rpcpass -cmd getblockcount
./rpc-test -host localhost:25174 -user rpcuser -pass rpcpass -cmd getblockhash 1000
./rpc-test -host localhost:25174 -user rpcuser -pass rpcpass -cmd getnetworkinfo
```

## Flags

- `-host`: RPC server address (default: `localhost:25174`)
- `-user`: RPC username (required)
- `-pass`: RPC password (required)
- `-tls`: Enable TLS (default: false)
- `-cmd`: RPC method to call (default: `getblockcount`)
- `-v`: Enable ultra verbose output - logs all actions, requests, and raw responses (default: false)
- Additional arguments are passed as parameters to the RPC call

## Verbose Mode

When `-v` is set, the script outputs detailed diagnostic information to stderr:
- Connection parameters (host, TLS status)
- Authentication details (password is masked)
- RPC method being called
- Parameters being sent
- Raw responses from the server
- Any errors with full context

Example:
```bash
./rpc-test -user rpcuser -pass rpcpass -cmd getblockcount -v
```

Output (stderr):
```
[VERBOSE] Connecting to RPC server at localhost:25174
[VERBOSE] TLS enabled: false
[VERBOSE] User: rpcuser
[VERBOSE] Password: ********
[VERBOSE] RPC client created successfully
[VERBOSE] Executing command: getblockcount
[VERBOSE] Using specialized GetBlockCount method
[VERBOSE] Received result: map[blockcount:12345]
[VERBOSE] Formatting output
[VERBOSE] Done
```

## Examples

```bash
# Basic connectivity test
./rpc-test -user yourrpcuser -pass yourrpcpass

# Verbose mode with detailed logging
./rpc-test -user rpcuser -pass rpcpass -cmd getblockcount -v

# Get network info
./rpc-test -cmd getnetworkinfo -user rpcuser -pass rpcpass

# Get block hash for height 100
./rpc-test -cmd getblockhash 100 -user rpcuser -pass rpcpass

# Custom RPC call with parameters (verbosely)
./rpc-test -cmd getrawtransaction txid true -user rpcuser -pass rpcpass -v
```

The script outputs JSON-formatted results to stdout. Verbose logging goes to stderr.
