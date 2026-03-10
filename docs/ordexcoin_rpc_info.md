# Ordexcoin RPC Communication Guide

## Overview

Ordexcoin Core (`ordexcoind`) provides a JSON-RPC interface for programmatic access to the node and wallet functionality. This document describes the communication protocol, configuration, and methods available.

## Protocol

**Transport:** HTTP/HTTPS (HTTP only, no SSL/TLS encryption)  
**Version:** JSON-RPC 2.0  
**Method:** POST only (GET, HEAD, PUT return HTTP 405)  
**Content-Type:** `application/json`  
**Library:** libevent (evhttp)

### Request Format

```http
POST / HTTP/1.1
Host: 127.0.0.1:25173
Content-Type: application/json
Authorization: Basic <base64-encoded-user:pass>

{"method":"getblockchaininfo","params":[],"id":1}
```

### Response Format

```json
{
  "result": {...},
  "error": null,
  "id": 1
}
```

## Network Ports

Default RPC ports by network:

| Network  | Port   |
|----------|--------|
| Mainnet  | 25173  |
| Testnet  | 35173  |
| Signet   | 325173 |
| Regtest  | 18443  |

*Configure with `-rpcport=<port>`*

## Authentication

Ordexcoin supports three authentication methods:

### 1. Cookie-based Authentication (Default, Recommended)

- Random cookie generated at startup
- Cookie file location: `<datadir>/.cookie`
- Configurable via `-rpccookiefile=<path>`
- No passwords stored in configuration
- Used automatically by `ordexcoin-cli` if no `-rpcpassword` provided

### 2. Username/Password (Deprecated but Supported)

```bash
ordexcoind -rpcuser=myuser -rpcpassword=mypassword
```

Uses HTTP Basic Authentication. Password is sent Base64-encoded (not encrypted).

### 3. HMAC-SHA-256 Authentication (`-rpcauth`)

More secure than plain passwords. Format:
```
USERNAME:SALT$HASH
```

Generated using `share/rpcauth/rpcauth.py`:
```bash
python3 share/rpcauth/rpcauth.py
```

Multiple `-rpcauth` entries allowed for different users.

## Access Control

- **Default:** Only localhost (127.0.0.1 and ::1) can connect
- **Allow external IPs:** `-rpcallowip=<ip>` (CIDR notation or netmask, can use multiple times)
- **Warning:** Exposing RPC to external networks is dangerous without firewall/VPN protection

## Server Configuration

### Core Options

- `-server` - Enable RPC server (default: true for `ordexcoind`, false for GUI)
- `-rpcbind=<addr>[:port]` - Bind to specific address(es)
- `-rpcport=<port>` - Port to listen on
- `-rpcthreads=<n>` - Worker threads (default: 4)
- `-rpcworkqueue=<n>` - Work queue depth (default: 16)
- `-rpcservertimeout=<n>` - Request timeout in ms (default: 30000)
- `-rpcserialversion=<0|1>` - Transaction serialization (0=non-segwit, 1=segwit, default: 1)

### Access Control Options

- `-rpcwhitelist=<user:method1,method2,...>` - Per-user method whitelist
- `-rpcwhitelistdefault` - Enable default empty whitelist behavior

### Important Notes

- RPC runs in "warmup" mode during initial startup - accepts connections but rejects RPC calls until ready
- Query warmup status with `getrpcinfo` RPC
- Failed authentication attempts incur 250ms delay (brute-force protection)
- Maximum request size: 32 MB

## RPC Methods

### Blockchain Methods

- `getbestblockhash` - Hash of best (tip) block
- `getblock` - Retrieve block by hash/number
- `getblockchaininfo` - Current blockchain status
- `getblockcount` - Number of blocks in local best chain
- `getblockfilter` - Retrieve BIP157 filter
- `getblockfilterindex` - Return block filter index hash
- `getblockheader` - Retrieve block header by hash
- `getchaintips` - Information about all known tips
- `getdifficulty` - Current difficulty as decimal
- `getmempoolancestors` - Get mempool ancestors
- `getmempooldescendants` - Get mempool descendants
- `getmempoolentry` - Retrieve mempool entry
- `getmempoolinfo` - Information about mempool
- `getrawmempool` - All transaction IDs in mempool
- `gettxout` - Retrieve unspent transaction output
- `gettxoutproof` - Get proof of inclusion/exclusion
- `gettxoutsetinfo` - Statistics about UTXO set
- `preciousblock` - Treat block as if it were received before others
- `pruneblockchain` - Prunblockchain up to specified height
- `savemempool` - Save mempool to disk
- `verifychain` - Verify blockchain database
- `verifytxoutproof` - Verify proof of inclusion/exclusion

### Control Methods

- `getmemoryinfo` - Memory information
- `getrpcinfo` - RPC server information
- `help` - List commands or get help for a command
- `stop` - Stop Ordexcoin server

### Generating Methods

- `generatetoaddress` - Mine blocks to address (regtest only)

### Mining Methods

- `getblocktemplate` - Get block template
- `getmininginfo` - Mining-related information
- `prioritisetransaction` - Prioritize transaction for mining
- `submitblock` - Submit a new block
- `submitheader` - Submit block header

### Network Methods

- `addnode` - Add/remove a persistent peer
- `clearbanned` - Remove all banned IPs
- `disconnectnode` - Disconnect a node by address/id
- `getaddednodeinfo` - Added node information
- `getconnectioncount` - Count current connections
- `getnettotals` - Network traffic totals
- `getnetworkinfo` - Network and peer information
- `getpeerinfo` - Information about connected peers
- `listbanned` - List banned IPs/Subnets
- `ping` - Send ping to all connected nodes
- `setban` - Ban/unban a node by IP/Subnet

### Raw Transactions

- `analyzepsbt` - Analyze a PSBT
- `combinepsbt` - Combine multiple PSBTs
- `combinerawtransaction` - Combine raw transactions
- `converttopsbt` - Convert raw transaction to PSBT
- `createpsbt` - Create a PSBT from inputs/outputs
- `createRawTransaction` - Create raw transaction
- `decodepsbt` - decode a PSBT
- `decoderawtransaction` - Decode raw transaction
- `decodescript` - Decode hex-encoded script
- `finalizepsbt` - Finalize a PSBT
- `fundrawtransaction` - Fund raw transaction
- `getrawtransaction` - Retrieve raw transaction
- `joinpsbts` - Join two PSBTs
- `merkleblock` - Retrieve a merkle block
- `sendrawtransaction` - Submit raw transaction
- `signpsbt` - Sign inputs in a PSBT
- `siginrawtransactionwithwallet` - Sign raw transaction with wallet
- `testmempoolaccept` - Check if raw transaction would be accepted
- `unlockunspent` - Move locked UTXO back to spendable

### Util Methods

- `deriveaddresses` - Derive addresses for a descriptor
- `estimatesmartfee` - Estimate fee for priority confirmation
- `getdescriptorinfo` - Compute descriptor checksum
- `signmessagewithprivkey` - Sign message with private key
- `signmessage` - Sign message with wallet key
- `verifymessage` - Verify a signed message

### Wallet Methods (if wallet enabled)

- `abandontransaction` - Mark transaction and outputs as abandoned
- `abortrescan` - Abort current wallet rescan
- `addmultisigaddress` - Add multisig address to wallet
- `addwitnessaddress` - DEPRECATED, use `addmultisigaddress`
- `backupwallet` - Backup wallet to external file
- `bumpfee` - Bump transaction fee
- `createwallet` - Create a new wallet
- `dumpwallet` - Dump wallet keys to file
- `dumpprivkey` - Export private key
- `encryptwallet` - Encrypt wallet
- `getaddressesbylabel` - Get addresses by label
- `getaddressinfo` - Get address information
- `getbalance` - Get total available balance
- `getnewaddress` - Get new receiving address
- `getrawchangeaddress` - Get address for change
- `getreceivedbyaddress` - Get total amount received
- `gettributableinfo` - Get information about a tributable
- `gettxoutsetinfo` - Get UTXO set statistics
- `getunconfirmedbalance` - Get unconfirmed balance
- `getwalletinfo` - Get wallet information
- `importaddress` - Import address to wallet
- `importmulti` - Import multiple addresses/privkeys
- `importprivkey` - Import private key
- `importprunedfunds` - Import funds from pruned wallet
- `importpubkey` - Import public key
- `importwallet` - Import wallet dump file
- `keypoolrefill` - Refill keypool
- `listaddressgroupings` - List address groups
- `listlabels` - List all labels
- `listlockunspent` - List locked unspent outputs
- `listreceivedbyaddress` - List received by address
- `listsinceblock` - List transactions since block
- `listtransactions` - List wallet transactions
- `listunspent` - List unspent outputs
- `loadwallet` - Load wallet from file
- `lockunspent` - Lock/unlock UTXO
- `removeprunedfunds` - Remove pruned UTXO
- `rescanblockchain` - Rescan blockchain for wallet txs
- `sendmany` - Send to multiple addresses
- `sendtoaddress` - Send to address
- `sethdseed` - Set HD seed
- `setlabel` - Set label for address
- `settxfee` - Set transaction fee
- `settxfee` - DEPRECATED, use `settxfee`
- `walletlock` - Lock wallet
- `walletpassphrase` - temporarily unlock wallet
- `walletpassphrasechange` - Change wallet passphrase

## Using `ordexcoin-cli`

`ordexcoin-cli` is the command-line RPC client that communicates with `ordexcoind`.

### Basic Usage

```bash
# Using default localhost:25173
ordexcoin-cli getblockcount

# Specify RPC connection
ordexcoin-cli -rpcport=35173 -testnet getblockcount

# Use cookie authentication (automatic if no password set)
ordexcoin-cli getwalletinfo

# Specify wallet for multi-wallet setups
ordexcoin-cli -rpcwallet=wallet1.dat getbalance

# Use named arguments
ordexcoin-cli -named getbalance account=""
```

### Common Options

- `-rpcconnect=<ip>` - RPC server address (default: 127.0.0.1)
- `-rpcport=<port>` - RPC server port
- `-rpcuser=<user>` / `-rpcpassword=<pw>` - Credentials (if not using cookie)
- `-rpcwallet=<name>` - Target specific wallet endpoint
- `-rpcwait` - Wait for RPC server to start
- `-rpcwaittimeout=<seconds>` - Timeout for `-rpcwait`
- `-stdinrpcpass` - Read RPC password from stdin
- `-datadir=<dir>` - Data directory
- `-testnet` / `-signet` / `-regtest` - Network selection

### Examples

```bash
# Get blockchain info
ordexcoin-cli getblockchaininfo

# Get network status
ordexcoin-cli getnetworkinfo

# Get peer information
ordexcoin-cli getpeerinfo

# Generate blocks (regtest only)
ordexcoin-cli -regtest generatetoaddress 100 $(ordexcoin-cli -regtest getnewaddress)

# Send transaction
ordexcoin-cli sendtoaddress "address" 1.0

# List transactions
ordexcoin-cli listtransactions "*" 10
```

## Using `ordexcoin-wallet`

`ordexcoin-wallet` is an offline tool for wallet file management. It does NOT communicate with `ordexcoind`. It directly manipulates wallet files.

### Commands

- `info` - Get wallet info
- `create` - Create new wallet file
- `salvage` - Attempt recovery from corrupt wallet
- `dump` - Print all key-value records
- `createfromdump` - Create wallet from dump file

### Example

```bash
# Create a new wallet
ordexcoin-wallet -wallet=mywallet.dat create

# Get wallet info
ordexcoin-wallet -wallet=mywallet.dat info

# Dump wallet contents
ordexcoin-wallet -wallet=mywallet.dat dump > wallet_dump.txt
```

**Note:** `ordexcoin-wallet` operates on local wallet files and does not connect to any RPC server.

## Does `ordexcoin-qt` Run `ordexcoind` Internally?

**Yes.** The `ordexcoin-qt` GUI includes the full node functionality and does NOT require a separate `ordexcoind` process. When you enable "Accept incoming connections" (Options → Network → "Map port using UPnP" and/or "Allow incoming connections"), the GUI starts the P2P network listener and the RPC server internally.

### How It Works

- `ordexcoin-qt` uses the same codebase as `ordexcoind`
- The `-server` option is controlled by the "Enable server" checkbox in the GUI (Options → Network)
- When `-server=1` (enabled), the RPC server starts on the configured port
- The node runs in the same process as the GUI
- You can connect to the RPC interface on `127.0.0.1:25173` (or testnet/signet/regtest ports)
- The GUI itself uses RPC calls internally to interact with the node

### Configuration

In `ordexcoin-qt`:
1. Go to **Settings → Options → Network**
2. Check **"Enable server"** to accept incoming P2P connections and enable RPC
3. Optionally enable **"Map port using UPnP"** for automatic port forwarding
4. Click **OK** and restart the application

The configured RPC port can be changed via command line when launching `ordexcoin-qt`:
```bash
ordexcoin-qt -rpcport=25173 -server
```

### Accessing the RPC Server from `ordexcoin-cli`

When `ordexcoin-qt` is running with server enabled:

```bash
# Connect to the GUI's internal node
ordexcoin-cli getblockchaininfo

# Or specify port if non-default
ordexcoin-cli -rpcport=25173 getnetworkinfo
```

Authentication uses the cookie file in the data directory (default location varies by OS).

### Important Differences from Standalone `ordexcoind`

- `ordexcoin-qt` runs all components (node, wallet, GUI) in a single process
- No separate daemon process to manage
- All RPC methods available in `ordexcoind` are also available when `ordexcoin-qt` is running with `-server`
- The GUI itself is the node; "accepting incoming connections" means the P2P network stack is active

## Security Considerations

1. **Never expose RPC to public internet** - Use SSH tunnels or VPNs for remote access
2. **Use cookie authentication** - More secure than username/password
3. **Set strong `-rpcpassword`** if using password auth (but cookie auth is preferred)
4. **Firewall the RPC port** - Bind to localhost only unless necessary
5. **Use `-rpcallowip` carefully** - Only whitelist trusted IPs
6. **Consider `-rpcwhitelist`** to restrict available methods per user
7. **RPC does not use SSL** - All traffic is unencrypted; use VPN/TLS tunnel for remote access

## Example: Secure Remote Access via SSH Tunnel

```bash
# On local machine (client)
ssh -L 25173:127.0.0.1:25173 user@remote-server

# Then on local machine, connect to local port
ordexcoin-cli -rpcport=25173 getblockchaininfo
```

## Troubleshooting

### "Could not connect to the server"

- Ensure `ordexcoind` or `ordexcoin-qt` is running
- Verify `-server=1` is set for `ordexcoin-qt`
- Check RPC port with `netstat -tlnp | grep 25173`
- Verify `-rpcport` matches the server's listening port

### "401 Unauthorized"

- Cookie auth failed: verify datadir permissions
- If using `-rpcuser`/`-rpcpassword`, check credentials are correct
- Check that `ordexcoin-cli` is using the same datadir as the server

### RPC server not starting

- Check debug.log in datadir for errors
- Ensure port is not in use by another process
- Verify `-server=1` is set
- Check that data directory is writable

### "Server in warmup"

- Node is still starting up; wait a few moments and retry
- Use `-rpcwait` flag with `ordexcoin-cli` to wait automatically

## Appendix: Source Code References

- RPC server: `src/httprpc.cpp`, `src/httpserver.cpp`
- Server args: `src/init.cpp` (function `SetupServerArgs`)
- Node interface: `src/node/interfaces.cpp`
- CLI client: `src/ordexcoin-cli.cpp`
- Daemon: `src/ordexcoind.cpp`
- Wallet tool: `src/ordexcoin-wallet.cpp`
- GUI init: `src/init/ordexcoin-qt.cpp`
- Default ports: `src/chainparamsbase.cpp` (lines 47-53)

## License

Ordexcoin Core is MIT licensed. See COPYING for details.
