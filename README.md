# BlockchainVPN: Decentralized VPN Marketplace

BlockchainVPN is a peer-to-peer VPN marketplace built on top of the OrdexCoin blockchain. It allows anyone to become a VPN provider by announcing their service on-chain, and allows clients to discover, pay for, and connect to these services in a decentralized, permissionless manner.

## 1. Architecture Overview

The system utilizes the blockchain as an immutable, censorship-resistant bulletin board for service discovery and a settlement layer for payments.

### Components

1.  **OrdexCoin Layer (Consensus & Data):**
    *   Used to broadcast service advertisements via `OP_RETURN` outputs.
    *   Used for payment settlement (subscription or pay-as-you-go).

2.  **VPN Provider (Go Application):**
    *   Runs alongside an `ordexcoind` node (or connects to a remote one).
    *   Publishes "Service Announcement" transactions to the blockchain.
    *   Listens for incoming VPN connections (WireGuard).
    *   Monitors the blockchain for incoming payments.
    *   Manages a UDP echo server for latency testing.

3.  **VPN Client (Go Application):**
    *   Connects to an `ordexcoind` node to scan for Service Announcements.
    *   Maintains a local cache of available providers.
    *   Performs active probing (UDP Echo) to determine availability and speed.
    *   Performs GeoIP lookups locally to determine country of origin.
    *   Facilitates payment and configures the local network interface to tunnel traffic.

## 2. Protocol Specification

### Service Announcement (Provider -> Network)

To announce a VPN service, the Provider broadcasts a transaction containing a specific `OP_RETURN` output. This output holds the metadata required for clients to connect.

**Payload Structure (Max 80 bytes):**

| Field | Size (Bytes) | Description |
| :--- | :--- | :--- |
| **Magic Bytes** | 4 | `0x56504E01` ("VPN" v1) |
| **IP Type** | 1 | `0x04` for IPv4, `0x06` for IPv6 |
| **IP Address** | 4 or 16 | The public IP of the VPN endpoint |
| **Port** | 2 | The UDP listening port (uint16, Big Endian) |
| **Price** | 8 | Cost in Satoshis per session (uint64, Big Endian) |
| **Public Key** | 32 | Provider's WireGuard Public Key |

*Note: If the IP address changes, the provider simply broadcasts a new transaction. Clients always prioritize the most recent transaction from a specific provider identity.*

### Payment & Authorization (Client -> Provider)

Clients purchase access by sending a transaction to the provider's address (derived from the announcement transaction inputs). This transaction must include an `OP_RETURN` output with the client's public key to authorize the connection.

**Payload Structure:**

| Field | Size (Bytes) | Description |
| :--- | :--- | :--- |
| **Magic Bytes** | 4 | `0x50415901` ("PAY" v1) |
| **Public Key** | 32 | Client's WireGuard Public Key |

## 3. Application Logic

### A. VPN Provider

1.  **Initialization:**
    *   Generates or loads a WireGuard private/public key.
    *   Detects public IP address (automatically or via config).
    *   Sets up the WireGuard interface and applies bandwidth limits (Linux only).
2.  **Announcement:**
    *   Constructs a raw transaction using `ordexcoind` RPC.
    *   Adds an output with `OP_RETURN` containing the service payload.
    *   Signs and broadcasts the transaction.
    *   Re-announces periodically (e.g., every 24 hours).
3.  **Service Loop:**
    *   Listens on the specified UDP port for WireGuard traffic.
    *   Runs a UDP echo server for latency checks.
    *   **Payment Monitor:** Scans the blockchain for transactions paying the service price to the provider's address containing a valid "PAY" `OP_RETURN` payload.
    *   **Access Control:** Updates the WireGuard interface peers list based on valid, non-expired payments.

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
    *   Generates a temporary WireGuard key pair.
    *   Sends a payment transaction with the generated public key.
    *   Configures local network interface to tunnel traffic.
    *   Monitors latency and disconnects if it exceeds a threshold.

## 4. Getting Started

### Prerequisites

*   **Go 1.16+** installed.
*   **OrdexCoin Core (`ordexcoind`)** running and fully synced.
    *   RPC must be enabled (`server=1`).
    *   Transaction indexing (`txindex=1`) is recommended for faster scanning but not strictly required for basic operation.
*   **WireGuard** tools installed (`wg`, `ip` command on Linux).
*   **GeoIP Database**: Download `GeoLite2-Country.mmdb` from MaxMind and place it in the project root for country detection.

### Installation

```bash
cd blockchain-vpn
go build -o bcvpn .
```

### Configuration

Generate a default configuration file:

```bash
./bcvpn generate-config
```

Edit `config.json` to match your environment:

*   **rpc**: Connection details for your local `ordexcoind` node.
*   **provider**: Settings if you intend to sell VPN service (IP, Port, Price).
*   **client**: Settings for connecting to others (Interface Name).

## Usage

### For VPN Providers

To start selling bandwidth:

```bash
sudo ./bcvpn start-provider
```

*   This command will:
    1.  Set up the WireGuard interface (requires `sudo`).
    2.  Apply bandwidth limits (if configured).
    3.  Announce your service on the blockchain (requires wallet funds).
    4.  Start a payment monitor to listen for incoming customers.
    5.  Start a UDP echo server for latency testing.
    6.  Automatically manage the WireGuard peer list based on valid payments.

### For VPN Clients

1.  **Scan for Providers**:
    Find available VPNs, sorted by latency, price, or country.
    ```bash
    ./bcvpn scan --sort=latency --country=US
    ```

2.  **Connect**:
    Follow the interactive prompts in the `scan` command to select a provider. The tool will:
    *   Generate a temporary WireGuard key pair.
    *   Send the payment transaction.
    *   Configure your network interface to connect.
    *   Monitor latency and auto-disconnect if specified.

3.  **Monitor Status**:
    View connection stats (data usage, handshake time).
    ```bash
    sudo ./bcvpn monitor
    ```

4.  **Payment History**:
    View a log of past payments.
    ```bash
    ./bcvpn history
    ```

4.  **Disconnect**:
    Tear down the connection.
    ```bash
    sudo ./bcvpn disconnect
    ```

## 5. Using Other Blockchains

While designed for OrdexCoin, this software is compatible with most Bitcoin-derived blockchains (Bitcoin, Litecoin, Dogecoin, etc.) that support `OP_RETURN` and the standard RPC interface.

To adapt this for another chain:

1.  **RPC Configuration**: Update `config.json` with the RPC credentials and port of the target blockchain's daemon (e.g., `8332` for Bitcoin).
2.  **Address Format**: The code dynamically detects the chain (Mainnet, Testnet, Regtest) from the RPC connection and adjusts address decoding accordingly.
3.  **Fees**: Ensure the hardcoded fees (e.g., `10000` sats in `provider.go`) are appropriate for the target network's fee market.

## 6. Project Status & Roadmap

### Completed Features

- [x] **Core Protocol**: Service announcement and payment payloads defined and implemented.
- [x] **Provider Logic**:
    - [x] Automatic IP detection.
    - [x] Service announcement broadcasting (with re-announce).
    - [x] WireGuard interface setup and management.
    - [x] Payment monitoring and automatic peer authorization.
    - [x] Bandwidth limiting (Linux).
    - [x] UDP Echo server for latency tests.
- [x] **Client Logic**:
    - [x] Blockchain scanning for providers.
    - [x] GeoIP enrichment and Latency testing.
    - [x] Sorting and filtering (Price, Country, Latency).
    - [x] Interactive selection and connection.
    - [x] Payment transaction construction (with OP_RETURN).
    - [x] Payment retry mechanism.
    - [x] Latency monitoring and auto-disconnect.
    - [x] Payment history logging.
- [x] **Cross-Platform**:
    - [x] Linux support (full feature set).
    - [x] Windows/macOS compilation support (stubs for interface management).

### Todo List

- [ ] **Robustness & Error Handling**
  - [ ] Handle blockchain reorgs in the Payment Monitor (remove authorization if payment tx is orphaned).
  - [ ] Validate `chaincfg` parameters dynamically for Altchains beyond standard testnets.

- [ ] **Cross-Platform Support**
  - [ ] Replace `exec.Command("ip", ...)` with a cross-platform library for interface management (Windows/macOS support).
  - [ ] Ensure file paths for config and keys are OS-agnostic.

- [ ] **Security**
  - [ ] Encrypt the `provider.key` file on disk.
  - [ ] Validate input data from `OP_RETURN` strictly to prevent injection attacks.
  - [ ] Run the WireGuard interface in a separate network namespace (optional, for better isolation).

- [ ] **Advanced Features**
  - [ ] **NAT Traversal**: Implement UPnP or NAT-PMP for providers behind home routers.
  - [ ] **Dynamic Pricing**: Allow providers to update price without re-announcing (or minimize re-announcement cost).
  - [ ] **Session Management**: Implement logic to handle session expiration gracefully (auto-disconnect or auto-renew).
  - [ ] **Multi-Chain Support**: Abstract the `chaincfg` dependency to easily switch between Bitcoin, Litecoin, OrdexCoin, etc. via config.

## License

General open source, source available but the copyright holder keeps the right to commercial use. 