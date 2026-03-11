# BlockchainVPN: Decentralized VPN Marketplace

BlockchainVPN is a peer-to-peer VPN marketplace built on top of the OrdexCoin blockchain. It allows anyone to become a VPN provider by announcing their service on-chain, and allows clients to discover, pay for, and connect to these services in a decentralized, permissionless manner.

Current version: `0.4.9`

## 0. Support the development

BTC: bc1qkmzc6d49fl0edyeynezwlrfqv486nmk6p5pmta
ETH/ERC-20: bc1qkmzc6d49fl0edyeynezwlrfqv486nmk6p5pmta
LTC: bc1qkmzc6d49fl0edyeynezwlrfqv486nmk6p5pmta
SOL: HB2o6q6vsW5796U5y7NxNqA7vYZW1vuQjpAHDo7FAMG8
XRP: rUW7Q64vR4PwDM3F27etd6ipxK8MtuxsFs

OXC: oxc1q3psft0hvlslddyp8ktr3s737req7q8hrl0rkly
OXG: oxg1q34apjkn2yc6rsvuua98432ctqdrjh9hdkhpx0t

All donations will be direclty used to develop and maintain the blockchain VPN package and system, as well as expand to support multiple chains and connection methods such as native Bitcoin and Socks5 proxy vpns.

## 1. Architecture Overview

The system utilizes the blockchain as an immutable, censorship-resistant bulletin board for service discovery and a settlement layer for payments.

### Components

1.  **OrdexCoin Layer (Consensus & Data):**
    *   Used to broadcast service advertisements via `OP_RETURN` outputs.
    *   Used for payment settlement (subscription or pay-as-you-go).

2.  **VPN Provider (Go Application):**
    *   Runs alongside an `ordexcoind` node (or connects to a remote one).
    *   Publishes "Service Announcement" transactions to the blockchain.
    *   Listens for incoming TLS connections over TCP and forwards traffic via a TUN interface.
    *   Monitors the blockchain for incoming payments.
    *   Manages a UDP echo server for latency testing.

3.  **VPN Client (Go Application):**
    *   Connects to an `ordexcoind` node to scan for Service Announcements.
    *   Maintains a local cache of available providers.
    *   Performs active probing (UDP Echo) to determine availability and speed.
    *   Performs GeoIP lookups to estimate country of origin.
    *   Facilitates payment and configures the local network interface to tunnel traffic.

## 2. Protocol Specification

### Service Announcement (Provider -> Network)

To announce a VPN service, the Provider broadcasts a transaction containing a specific `OP_RETURN` output. This output holds the metadata required for clients to connect.

**Payload Structure (Max 80 bytes):**

| Field | Size (Bytes) | Description |
| :--- | :--- | :--- |
| **Magic Bytes** | 4 | `0x56504E01` ("VPN" v1) |
| **IP Type** | 1 | `0x04` for IPv4, `0x06` for IPv6. |
| **IP Address** | 4 or 16 | The public IP of the VPN endpoint. |
| **Port** | 2 | The TCP listening port for TLS connections (uint16, Big Endian). |
| **Price** | 8 | Cost in Satoshis per session (uint64, Big Endian). |
| **Public Key** | 33 | Provider's compressed `secp256k1` Public Key. |

*Note: If the IP address changes, the provider simply broadcasts a new transaction. Clients always prioritize the most recent transaction from a specific provider identity.*

### Payment & Authorization (Client -> Provider)

Clients purchase access by sending a transaction to the provider's address (derived from the announcement transaction inputs). This transaction must include an `OP_RETURN` output with the client's public key. This key is then used to generate a client certificate to authorize the TLS connection.

**Payload Structure:**

| Field | Size (Bytes) | Description |
| :--- | :--- | :--- |
| **Magic Bytes** | 4 | `0x50415901` ("PAY" v1) |
| **Public Key** | 33 | Client's compressed `secp256k1` Public Key. |

## 3. Application Logic

### A. VPN Provider

1.  **Initialization:**
    *   Generates or loads a `secp256k1` private/public key (`btcec`).
    *   Detects public IP address (automatically or via config).
    *   Sets up a TUN interface and applies runtime networking configuration (platform-dependent backend).
2.  **Announcement:**
    *   Constructs a raw transaction using `ordexcoind` RPC.
    *   Adds an output with `OP_RETURN` containing the service payload.
    *   Signs and broadcasts the transaction.
    *   Re-announces periodically (e.g., every 24 hours).
3.  **Service Loop:**
    *   Listens on the specified TCP port for TLS connections.
    *   Runs a UDP echo server for latency checks.
    *   **Payment Monitor:** Scans the blockchain for transactions paying the service price to the provider's address containing a valid "PAY" `OP_RETURN` payload.
    *   **Access Control:** Verifies the client's TLS certificate against a list of authorized public keys derived from valid payments.

### B. VPN Client

1.  **Discovery:**
    *   Queries `ordexcoind` for recent blocks.
    *   Scans transactions for outputs starting with `OP_RETURN` + `Magic Bytes`.
    *   Decodes payload to extract IP, Port, Price, and Key.
2.  **Enrichment & Filtering:**
    *   **Availability/Speed:** Sends a UDP ping to the IP:Port to measure RTT.
    *   **GeoIP:** Uses a local GeoLite2 database to map the IP to a Country Code.
3.  **Selection:**
    *   User selects a provider from a sorted list (by Country, Speed, or Cost).
4.  **Connection:**
    *   Generates a temporary `secp256k1` key pair.
    *   Sends a payment transaction with the generated public key.
    *   Creates a local TUN interface and establishes a TLS connection to the provider, forwarding traffic between them.

## 4. Getting Started

### Prerequisites

*   **Go 1.22+** installed.
*   **OrdexCoin Core (`ordexcoind`)** running and fully synced.
    *   RPC must be enabled (`server=1`).
    *   Transaction indexing (`txindex=1`) is recommended for faster scanning but not strictly required for basic operation.
*   Elevated privileges are required on Linux/macOS/Windows to configure TUN, routes, DNS, and provider NAT features.
*   **GeoIP Database**: Download `GeoLite2-Country.mmdb` from MaxMind and place it in the project root for country detection.
*   See [docs/INSTALL.md](docs/INSTALL.md) for OS-specific installation and privilege setup.

### Installation

Build the CLI and GUI for your current OS:

```bash
go build -o bcvpn ./cmd/bcvpn
go build -o bcvpn-gui ./cmd/bcvpn-gui
```

Cross-compile the CLI for major platforms:

```bash
GOOS=linux GOARCH=amd64 go build -o bcvpn-linux-amd64 ./cmd/bcvpn
GOOS=darwin GOARCH=amd64 go build -o bcvpn-darwin-amd64 ./cmd/bcvpn
GOOS=windows GOARCH=amd64 go build -o bcvpn-windows-amd64.exe ./cmd/bcvpn
```

Or use `Makefile` targets:

```bash
make build            # native CLI
make build-gui        # native GUI
make build-cli-all    # cross-platform CLI artifacts (Linux/Darwin/Windows)
make test             # run unit tests
make tidy             # sync go.mod
```

Notes:

*   CLI cross-compilation is supported for Linux/macOS/Windows.
*   GUI builds use Fyne/OpenGL dependencies and are best built natively on the target OS.

### Configuration

Generate a default configuration file:

```bash
./bcvpn generate-config
```

Edit `config.json` to match your environment:

*   **rpc**: Connection details for your local `ordexcoind` node.
*   **logging**: Runtime log output format (`text` or `json`).
*   **security**: Key storage backend, revocation cache, TLS policy, and metrics auth (`key_storage_mode`, `revocation_cache_file`, `tls_min_version`, `tls_profile`, `metrics_auth_token`).
*   **provider**: Settings if you intend to sell VPN service (IP, Port, Price, `max_consumers`, `enable_egress_nat`, `nat_outbound_interface`, `isolation_mode`, `allowlist_file`, `denylist_file`, `cert_lifetime_hours`, `cert_rotate_before_hours`, `health_check_enabled`, `health_check_interval`).
*   **client**: Settings for connecting to others (Interface Name).
*   By default, the app stores `config.json`, `provider.key`, and `history.json` in your OS user config directory under `BlockchainVPN` (for example `~/.config/BlockchainVPN` on Linux).

Sample `config.json`:

```json
{
  "rpc": {
    "host": "localhost:18443",
    "user": "yourrpcuser",
    "pass": "yourrpcpassword"
  },
  "logging": {
    "format": "text",
    "level": "info"
  },
  "security": {
    "key_storage_mode": "file",
    "key_storage_service": "BlockchainVPN",
    "revocation_cache_file": "",
    "tls_min_version": "1.3",
    "tls_profile": "modern",
    "metrics_auth_token": ""
  },
  "provider": {
    "interface_name": "bcvpn0",
    "listen_port": 51820,
    "announce_ip": "",
    "country": "",
    "price_sats_per_session": 1000,
    "max_consumers": 0,
    "private_key_file": "<APP_CONFIG_DIR>/provider.key",
    "bandwidth_limit": "10mbit",
    "enable_nat": true,
    "enable_egress_nat": false,
    "nat_outbound_interface": "",
    "isolation_mode": "none",
    "allowlist_file": "",
    "denylist_file": "",
    "cert_lifetime_hours": 720,
    "cert_rotate_before_hours": 24,
    "health_check_enabled": true,
    "health_check_interval": "30s",
    "metrics_listen_addr": "127.0.0.1:9090",
    "bandwidth_monitor_interval": "30s",
    "tun_ip": "10.10.0.1",
    "tun_subnet": "24"
  },
  "client": {
    "interface_name": "bcvpn1",
    "tun_ip": "10.10.0.2",
    "tun_subnet": "24",
    "enable_kill_switch": false,
    "metrics_listen_addr": "127.0.0.1:9091",
    "strict_verification": false,
    "verify_throughput_after_connect": true
  }
}
```

## Usage

### For VPN Providers

To start selling bandwidth:

```bash
./bcvpn start-provider
```

Non-interactive key unlock for automation (file mode):

```bash
export BCVPN_KEY_PASSWORD='your-password'
./bcvpn start-provider --key-password-env BCVPN_KEY_PASSWORD
```

*   This command will:
    1.  Set up the provider TUN interface (requires elevated privileges on most systems).
    2.  Announce your service on the blockchain (requires wallet funds).
    3.  Start a payment monitor to listen for incoming customers.
    4.  Start a UDP echo server for latency testing.
    5.  Automatically verify connecting clients based on valid payments.

### For VPN Clients

1.  **Scan for Providers**:
    Find available VPNs, with filters/sorting for latency, price, country, bandwidth, and capacity.
    ```bash
    ./bcvpn scan --sort=score --country=US --max-price=2000 --min-bandwidth-kbps=25000 --max-latency-ms=80 --min-available-slots=2
    ```

2.  **Connect**:
    Follow the interactive prompts in the `scan` command to select a provider. The tool will:
    *   Generate a temporary key pair.
    *   Send the payment transaction.
    *   Configure a local TLS-over-TUN tunnel to the provider.

3.  **Payment History**:
    View a log of past payments.
    ```bash
    ./bcvpn history
    ```

4.  **Rotate Provider Key**:
    Rotate the encrypted provider key file and generate a new provider identity.
    ```bash
    ./bcvpn rotate-provider-key
    ./bcvpn rotate-provider-key --old-password-env BCVPN_OLD_PASS --new-password-env BCVPN_NEW_PASS
    ```

5.  **Status (Human or JSON)**:
    Inspect current config/runtime readiness for automation or diagnostics (including networking privilege readiness).
    ```bash
    ./bcvpn status
    ./bcvpn status --json
    ```

6.  **CLI Config Management**:
    Read, update, and validate settings from the CLI.
    ```bash
    ./bcvpn config get provider.listen_port
    ./bcvpn config set provider.listen_port 51820
    ./bcvpn config set client.enable_kill_switch true
    ./bcvpn config validate
    ```
    Security-related keys are also exposed:
    ```bash
    ./bcvpn config set security.key_storage_mode auto
    ./bcvpn config set security.revocation_cache_file /path/to/revoked_keys.txt
    ./bcvpn config set security.tls_profile compat
    ./bcvpn config set client.strict_verification true
    ```
    Import/export are also supported:
    ```bash
    ./bcvpn config export ./my-profile.json
    ./bcvpn config import ./my-profile.json --validate
    ```

7.  **Runtime Metrics Endpoint (Optional)**:
    Set `provider.metrics_listen_addr` or `client.metrics_listen_addr` to expose:
    ```bash
    curl http://127.0.0.1:9090/metrics.json
    ```
    To require auth, set `security.metrics_auth_token` and send:
    ```bash
    curl -H "X-BCVPN-Metrics-Token: <token>" http://127.0.0.1:9090/metrics.json
    ```

8.  **Structured Logs (Optional)**:
    Set `logging.format` (`text`/`json`) and `logging.level` (`debug`/`info`/`warn`/`error`) in `config.json`, or override at runtime:
    ```bash
    BCVPN_LOG_FORMAT=json ./bcvpn status --json
    BCVPN_LOG_LEVEL=warn ./bcvpn start-provider
    ```

9.  **Version Info**:
    Print semantic version and build metadata:
    ```bash
    ./bcvpn version
    ./bcvpn version --json
    ```

10. **Runtime Doctor Checks**:
    Run startup diagnostics for config, privileges, key storage backend, and platform tools:
    ```bash
    ./bcvpn doctor
    ./bcvpn doctor --json
    ```
11. **Event Timeline and Diagnostics**:
    ```bash
    ./bcvpn events --limit=100
    ./bcvpn events --json
    ./bcvpn diagnostics
    ./bcvpn diagnostics --out ./diag.json
    ```

## 5. Using Other Blockchains

While designed for OrdexCoin, this software is compatible with most Bitcoin-derived blockchains (Bitcoin, Litecoin, Dogecoin, etc.) that support `OP_RETURN` and the standard RPC interface.

To adapt this for another chain:

1.  **RPC Configuration**: Update `config.json` with the RPC credentials and port of the target blockchain's daemon (e.g., `8332` for Bitcoin).
2.  **Address Format**: The code dynamically detects the chain (Mainnet, Testnet, Regtest) from the RPC connection and adjusts address decoding accordingly.
3.  **Fees**: Fee selection uses node-reported dynamic estimates (`estimatesmartfee`, relay fee fallback).

## 6. Project Status

### Feature Checklist

- [x] On-chain service announcement and discovery protocol (`OP_RETURN` payloads).
- [x] v2 provider metadata payload (bandwidth, country, capacity, availability flags) with scanner compatibility for v1/v2.
- [x] Advanced scanner filtering (country, price, bandwidth, latency, slots) and multi-field sorting.
- [x] Provider score/ranking algorithm and output in CLI/GUI.
- [x] Provider service announcement rebroadcasting and price update announcements.
- [x] Provider heartbeat/availability broadcasts for freshness-aware discovery.
- [x] TLS-over-TUN tunnel transport with cert identity bound to provider public key.
- [x] Payment flow with deterministic UTXO selection and dynamic fee estimation (`estimatesmartfee` + relay fallback).
- [x] Payment monitor with reorg handling and tx->peer authorization tracking.
- [x] Dynamic provider-side client IP allocation.
- [x] Throughput accounting and optional bandwidth limiting per session.
- [x] Provider session-capacity control (`provider.max_consumers`, `0` = unlimited).
- [x] Active provider health checks for TUN interface and TLS listener.
- [x] Provider access policies via optional allowlist/denylist files.
- [x] Provider key encryption at rest and rotation workflow.
- [x] Optional provider sandbox hardening mode on Linux (`isolation_mode=sandbox`).
- [x] NAT traversal support for providers (UPnP + NAT-PMP).
- [x] Provider egress NAT backend on Linux, macOS, and Windows.
- [x] Client routing and DNS auto-configuration for Linux, macOS, and Windows.
- [x] Optional cross-platform client kill switch mode.
- [x] Post-connect client security checks (egress IP transition, DNS leak heuristic, provider-country verification).
- [x] OS-native DNS introspection checks and optional strict verification mode.
- [x] Throughput verification against advertised provider bandwidth after connect.
- [x] RPC retry + exponential backoff for transient failures.
- [x] Payment history storage and reporting.
- [x] Machine-readable status output for automation (`bcvpn status --json`).
- [x] Runtime metrics endpoint (`/metrics.json`) for provider/client health and throughput.
- [x] Structured JSON log mode for CLI/GUI backend actions.
- [x] Crash-safe route/DNS restore marker recovery on startup.
- [x] Optional hardware-backed provider key storage integration (macOS Keychain, Windows DPAPI, Linux libsecret).
- [x] Mutual TLS revocation cache enforcement for provider/client cert identity keys.
- [x] Configurable TLS minimum version/profile with cipher/profile reporting in `status --json`.
- [x] Scriptable CLI config subcommands (`config get/set/validate`).
- [x] CLI config profile export/import (`config export`, `config import`).
- [x] Runtime event timeline and diagnostics bundle export (`events`, `diagnostics`).
- [x] Cross-platform GUI application (`cmd/bcvpn-gui`) using Fyne.
- [x] GUI first-run setup wizard (config, RPC, key, privilege checks).
- [x] GUI auto-elevation relaunch flow (Linux/macOS/Windows backends).
- [x] GUI parity for provider controls and diagnostics (rebroadcast, price update, metrics snapshots, doctor checks, version visibility).
- [x] GUI event timeline and one-click diagnostics/profile import-export actions.
- [x] OS-agnostic application config directory for `config.json`, `provider.key`, and `history.json`.
- [x] Port conflict detection for provider/client listeners.
- [x] Built-in credit manager for tracking prepaid usage.
- [x] TLS certificate pinning for critical RPC/tunnel endpoints.
- [x] Payment amount verification for client and provider payment flows.
- [x] GUI real-time wallet balance display.
- [x] GUI metrics/health panels with reactive auto-refresh.
- [x] GUI confirmation dialogs for high-impact actions (e.g. "Disconnect All" and closing active clients).
- [x] GUI log panel enhancements (auto-scroll, search, export) for easier diagnostics.
- [x] GUI progress indicators for long-running operations (scan/connect flows).
- [x] GUI reactive status labels that accurately reflect backend connection/payment state.

### How It Works

1.  Provider starts (`start-provider`), optionally opens NAT mappings, announces endpoint to chain, and serves TLS-over-TUN sessions.
2.  Client scans chain (`scan`), enriches candidates with latency/GeoIP, and selects a provider.
3.  Client sends on-chain payment containing a temporary public key.
4.  Provider payment monitor authorizes that key until session expiry and enforces optional max active sessions.
5.  Client connects over TLS, receives a dynamic TUN IP, traffic is forwarded through provider, and post-connect security checks run.

### Platform Coverage

- Linux: full runtime support, including provider egress NAT and sandbox hardening mode.
- macOS: full client route/DNS/TUN automation support and provider egress NAT backend.
- Windows: full client route/DNS/TUN automation support and provider egress NAT backend.
- Other OSes: explicit runtime stubs and clear unsupported errors.
- Privilege preflight is enforced before provider start and before client payment/connection.

- [ ] Improve throughput verification with provider-assisted active probes (less dependence on external download endpoints).
- [ ] Add signed provider reputation/quality metadata and configurable weighted-selection profiles.

- Detailed install and privilege setup: [docs/INSTALL.md](docs/INSTALL.md)
- UI design and behavior: [docs/GUI.md](docs/GUI.md)
- Networking notes: [docs/NETWORKING.md](docs/NETWORKING.md)
- End-to-end user guide: [docs/USER_GUIDE.md](docs/USER_GUIDE.md)
- Troubleshooting by OS: [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)
- Packaging and installer guidance: [docs/PACKAGING.md](docs/PACKAGING.md)
- Security model: [docs/SECURITY.md](docs/SECURITY.md)
- Provider operations runbook: [docs/OPERATIONS.md](docs/OPERATIONS.md)
- Automation JSON examples: [docs/AUTOMATION.md](docs/AUTOMATION.md)
- Versioning policy and release progression: [docs/VERSIONING.md](docs/VERSIONING.md)
- Version history and notable changes: [docs/CHANGELOG.md](docs/CHANGELOG.md)

## 7. Project File Layout

The project is organized into the following directory structure. Please ensure your files are moved to the correct locations. 

```
.
├── Makefile
├── README.md
├── docs/
├── cmd/
│   ├── bcvpn/
│   │   └── main.go // Main CLI application entrypoint
│   └── bcvpn-gui/
│       └── main.go // GUI application entrypoint
├── internal/
│   ├── auth/ // Authorization management
│   ├── blockchain/ // Blockchain interaction (payment, provider, scanner)
│   ├── config/ // Configuration loading and management
│   ├── crypto/ // Encryption/decryption logic
│   ├── geoip/ // GeoIP and latency enrichment
│   ├── history/ // Payment history management
│   ├── nat/ // UPnP and NAT-PMP logic
│   ├── protocol/ // On-chain data structures and encoding
│   ├── tunnel/ // Core VPN logic (TUN, TLS, Networking)
│   └── util/ // Miscellaneous utility functions
└── go.mod
```


## License

General open source, source available but the copyright holder keeps the right to commercial use. 
