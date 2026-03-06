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
    *   Listens for incoming VPN connections (e.g., WireGuard or OpenVPN).
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
| **Encryption Key** | 33 | Compressed Public Key for establishing the secure tunnel. |
| **Metadata** | Variable | Flags (e.g., protocol type: WireGuard/OpenVPN). |

*Note: If the IP address changes, the provider simply broadcasts a new transaction. Clients always prioritize the most recent transaction from a specific provider identity.*

---

## 3. Application Logic

### A. VPN Provider

1.  **Initialization:**
    *   Generate a VPN configuration (e.g., WireGuard private/public key).
    *   Detect public IP address.
2.  **Announcement:**
    *   Construct a raw transaction using `ordexcoind` RPC.
    *   Add an output with `OP_RETURN` containing the payload defined above.
    *   Sign and broadcast the transaction.
3.  **Service Loop:**
    *   Listen on the specified UDP/TCP port.
    *   When a client initiates a handshake, verify payment on the blockchain (or via a payment channel).
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
    *   Client configures local network interface using the Provider's Public Key and Endpoint.

---

## 4. Example Implementation (Go)

Below is a simplified Go implementation demonstrating how to encode an announcement and how to scan/parse the blockchain for providers.

### Prerequisites
*   `go get github.com/btcsuite/btcd/rpcclient`
*   `go get github.com/btcsuite/btcd/wire`

### `vpn_protocol.go`

```go
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
)

const MagicBytes = 0x56504E01 // "VPN" + Version 1

type VPNEndpoint struct {
	IP        net.IP
	Port      uint16
	Price     uint64 // Satoshis per hour
	PublicKey []byte // 33 bytes compressed
}

// EncodePayload creates the OP_RETURN data
func (v *VPNEndpoint) EncodePayload() ([]byte, error) {
	buf := new(bytes.Buffer)

	// 1. Magic Bytes
	if err := binary.Write(buf, binary.BigEndian, uint32(MagicBytes)); err != nil {
		return nil, err
	}

	// 2. IP Address
	ip4 := v.IP.To4()
	if ip4 != nil {
		// It's an IPv4 address
		buf.WriteByte(0x04)
		buf.Write(ip4)
	} else {
		// It's an IPv6 address
		ip16 := v.IP.To16()
		if ip16 == nil {
			return nil, fmt.Errorf("invalid IP address format: not IPv4 or IPv6")
		}
		buf.WriteByte(0x06)
		buf.Write(ip16)
	}

	// 3. Port
	if err := binary.Write(buf, binary.BigEndian, v.Port); err != nil {
		return nil, err
	}

	// 4. Price
	if err := binary.Write(buf, binary.BigEndian, v.Price); err != nil {
		return nil, err
	}

	// 5. Public Key
	if len(v.PublicKey) != 33 {
		return nil, fmt.Errorf("invalid pubkey length")
	}
	buf.Write(v.PublicKey)

	return buf.Bytes(), nil
}

// DecodePayload parses the OP_RETURN data
func DecodePayload(data []byte) (*VPNEndpoint, error) {
	buf := bytes.NewReader(data)

	// 1. Check Magic
	var magic uint32
	if err := binary.Read(buf, binary.BigEndian, &magic); err != nil {
		return nil, fmt.Errorf("could not read magic bytes: %w", err)
	}
	if magic != MagicBytes {
		return nil, fmt.Errorf("invalid magic bytes")
	}

	// 2. IP Type and Address
	ipType, err := buf.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("could not read ip type: %w", err)
	}

	var ip net.IP
	switch ipType {
	case 0x04:
		ipBytes := make([]byte, 4)
		if _, err := buf.Read(ipBytes); err != nil {
			return nil, fmt.Errorf("could not read ipv4 address: %w", err)
		}
		ip = net.IP(ipBytes)
	case 0x06:
		ipBytes := make([]byte, 16)
		if _, err := buf.Read(ipBytes); err != nil {
			return nil, fmt.Errorf("could not read ipv6 address: %w", err)
		}
		ip = net.IP(ipBytes)
	default:
		return nil, fmt.Errorf("unknown ip type: %d", ipType)
	}

	// 3. Port
	var port uint16
	if err := binary.Read(buf, binary.BigEndian, &port); err != nil {
		return nil, fmt.Errorf("could not read port: %w", err)
	}

	// 4. Price
	var price uint64
	if err := binary.Read(buf, binary.BigEndian, &price); err != nil {
		return nil, fmt.Errorf("could not read price: %w", err)
	}

	// 5. PubKey
	// The remainder of the payload should be the public key.
	expectedPubKeyLen := 33
	if buf.Len() != expectedPubKeyLen {
		return nil, fmt.Errorf("incorrect remaining payload length for public key, expected %d, got %d", expectedPubKeyLen, buf.Len())
	}
	pubKey := make([]byte, expectedPubKeyLen)
	if _, err := buf.Read(pubKey); err != nil {
		return nil, fmt.Errorf("could not read public key: %w", err)
	}

	return &VPNEndpoint{
		IP:        ip,
		Port:      port,
		Price:     price,
		PublicKey: pubKey,
	}, nil
}
```

### `scanner.go` (Client Logic)

```go
package main

import (
	"encoding/hex"
	"log"

	"github.com/btcsuite/btcd/rpcclient"
)

// ScanForVPNs scans the blockchain for VPN service announcements starting from
// the current tip and going backwards until startBlock.
func ScanForVPNs(client *rpcclient.Client, startBlock int64) ([]*VPNEndpoint, error) {
	var endpoints []*VPNEndpoint

	// Get current block count
	count, err := client.GetBlockCount()
	if err != nil {
		return nil, err
	}

	// Iterate backwards from tip to startBlock
	for i := count; i > startBlock && i > 0; i-- {
		hash, err := client.GetBlockHash(i)
		if err != nil {
			log.Printf("Could not get block hash for height %d: %v", i, err)
			continue
		}
		block, err := client.GetBlockVerbose(hash)
		if err != nil {
			log.Printf("Could not get block for hash %s: %v", hash, err)
			continue
		}

		for _, tx := range block.Tx {
			for _, vout := range tx.Vout {
				pkScript, err := hex.DecodeString(vout.ScriptPubKey.Hex)
				if err != nil {
					continue
				}

				if payload, err := ExtractScriptPayload(pkScript); err == nil {
					if endpoint, err := DecodePayload(payload); err == nil {
						endpoints = append(endpoints, endpoint)
					}
				}
			}
		}
	}
	return endpoints, nil
}
```

## 5. Future Improvements

*   **Payment Channels:** Instead of on-chain transactions for every hour of service, implement a unidirectional payment channel (like Lightning) to stream satoshis as data flows.
*   **Reputation System:** Clients can sign a message referencing the provider's transaction ID to leave a rating, stored off-chain (IPFS) or on-chain (expensive).
*   **Privacy:** Use Tor onion addresses in the IP field (requires protocol update) to hide the physical location of the VPN provider.