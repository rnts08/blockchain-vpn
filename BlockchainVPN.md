# BlockchainVPN: Decentralized VPN Network on OrdexCoin

This document outlines the architecture for a decentralized VPN network built on top of the OrdexCoin blockchain. The system utilizes the blockchain as an immutable, censorship-resistant bulletin board for service discovery and a settlement layer for payments.

## 1. Architecture Overview

The system consists of two primary applications: the **VPN Provider** and the **VPN Client**. They interact through the OrdexCoin blockchain and direct peer-to-peer connections.

### Components

1.  **OrdexCoin Layer (Consensus & Data):**
    *   Used to broadcast service advertisements via `OP_RETURN` outputs.
    *   Used for payment settlement (subscription or pay-as-you-go).

2.  **VPN Provider (Go Application):**
    *   Runs alongside an `ordexcoind` node (or connects to a remote one).
    *   Publishes "Service Announcement" transactions to the blockchain.
    *   Listens for incoming TLS connections over TCP and forwards traffic via a TUN interface.
    *   Monitors the blockchain for incoming payments.

3.  **VPN Client (Go Application):**
    *   Connects to an `ordexcoind` node to scan for Service Announcements.
    *   Maintains a local cache of available providers.
    *   Performs active probing (Ping/Handshake) to determine availability and speed.
    *   Performs GeoIP lookups locally to determine country of origin.
    *   Facilitates payment and configures the local network interface to tunnel traffic.

---

## 2. Data Protocol Specification

To announce a VPN service, the Provider broadcasts a transaction containing a specific `OP_RETURN` output. This output holds the metadata required for clients to connect.

### Service Announcement Payload (Max 80 bytes)

| Field | Size (Bytes) | Description |
| :--- | :--- | :--- |
| **Magic Bytes** | 4 | Protocol Identifier (`0x56504E01` for "VPN" v1) |
| **IP Type** | 1 | `0x04` for IPv4, `0x06` for IPv6. |
| **IP Address** | 4 or 16 | The public IP of the VPN endpoint. |
| **Port** | 2 | The listening port (uint16). |
| **Price** | 8 | Cost in Satoshis per Gigabyte or Hour (uint64). |
| **Encryption Key** | 33 | Provider's compressed `secp256k1` Public Key for establishing the secure tunnel. |
| **Metadata** | Variable | Reserved for future use. |

*Note: If the IP address changes, the provider simply broadcasts a new transaction. Clients always prioritize the most recent transaction from a specific provider identity.*

---

## 3. Application Logic

### A. VPN Provider

1.  **Initialization:**
    *   Generate a `secp256k1` private/public key.
    *   Detect public IP address.
2.  **Announcement:**
    *   Construct a raw transaction using `ordexcoind` RPC.
    *   Add an output with `OP_RETURN` containing the payload defined above.
    *   Sign and broadcast the transaction.
3.  **Service Loop:**
    *   Listen on the specified TCP port for TLS connections.
    *   When a client connects, verify its TLS certificate against a list of authorized keys derived from on-chain payments.
    *   Allow traffic flow.

### B. VPN Client

1.  **Discovery:**
    *   Query `ordexcoind` for recent blocks (e.g., last 1000 blocks).
    *   Scan transactions for outputs starting with `OP_RETURN` + `Magic Bytes`.
    *   Decode payload to extract IP, Port, Price, and Key.
2.  **Enrichment & Filtering:**
    *   **Availability:** Send a lightweight UDP ping to the IP:Port to check if it's online.
    *   **Speed:** Measure the round-trip time (RTT) of the ping.
    *   **GeoIP:** Use a local GeoLite2 database to map the IP to a Country Code.
3.  **Sorting & Selection:**
    *   Present list to user sorted by:
        *   Country (User preference)
        *   Speed (Lowest Latency)
        *   Cost (Lowest Price)
4.  **Connection:**
    *   User selects a provider.
    *   Client sends a payment transaction to the Provider's address (derived from the input of the Announcement transaction).
    *   Client creates a local TUN interface and establishes a TLS connection to the provider using their public key for verification.

---

## 5. Future Improvements

*   **Payment Channels:** Instead of on-chain transactions for every hour of service, implement a unidirectional payment channel (like Lightning) to stream satoshis as data flows.
*   **Reputation System:** Clients can sign a message referencing the provider's transaction ID to leave a rating, stored off-chain (IPFS) or on-chain (expensive).
*   **Privacy:** Use Tor onion addresses in the IP field (requires protocol update) to hide the physical location of the VPN provider.