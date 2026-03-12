# BlockchainVPN Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        BLOCKCHAINVPN ARCHITECTURE                          │
└─────────────────────────────────────────────────────────────────────────────┘

                              ┌──────────────────┐
                              │   ORDEXCOIND     │
                              │   (Blockchain)   │
                              │                  │
                              │  ┌────────────┐  │
                              │  │ OP_RETURN  │  │
                              │  │ Announce   │  │
                              │  └────────────┘  │
                              │  ┌────────────┐  │
                              │  │ Payments   │  │
                              │  │ (UTXO)     │  │
                              │  └────────────┘  │
                              └────────┬─────────┘
                                       │
         ┌─────────────────────────────┼─────────────────────────────┐
         │                             │                             │
         ▼                             │                             ▼
┌─────────────────────┐               │               ┌─────────────────────┐
│   VPN PROVIDER      │               │               │   VPN CLIENT        │
│                     │               │               │                     │
│ ┌─────────────────┐ │               │               │ ┌─────────────────┐ │
│ │ secp256k1 Keys  │ │               │               │ │ secp256k1 Keys  │ │
│ │ (btcec)         │ │               │               │ │ (temporary)     │ │
│ └────────┬────────┘ │               │               │ └────────┬────────┘ │
│          │          │               │               │          │          │
│          ▼          │               │               │          ▼          │
│ ┌─────────────────┐ │               │               │ ┌─────────────────┐ │
│ │ TUN Interface   │ │◄──────────────┤               │ │ TUN Interface   │ │
│ │ (bcvpn0)        │ │   TLS Tunnel  │               │ │ (bcvpn1)        │ │
│ └────────┬────────┘ │◄──────────────┤               │ └────────┬────────┘ │
│          │          │               │               │          │          │
│    ┌─────┴─────┐    │               │               │    ┌─────┴─────┐    │
│    │            │    │               │               │    │            │    │
│    ▼            ▼    │               │               │    ▼            ▼    │
│ ┌──────┐  ┌───────┐  │               │               │ ┌──────┐  ┌───────┐ │
│ │ NAT  │  │ TLS   │  │               │               │ │Route │  │ DNS   │ │
│ │ Egress│  │Server │  │               │               │ │Config│  │Config │ │
│ └──────┘  └───┬───┘  │               │               │ └──────┘  └───────┘ │
│               │      │               │               │          │        │
│               │      │               │               │          ▼        │
│               │      │               │               │ ┌─────────────────┐│
│               │      │               │               │ │ GeoIP + Latency││
│               │      │               │               │ │ Enrichment      ││
│               │      │               │               │ └────────┬────────┘│
│               │      │               │               │          │         │
│               ▼      │               │               │          ▼         │
│ ┌─────────────────┐   │               │               │ ┌─────────────────┐│
│ │ Payment Monitor│◄──┴───────────────┘               │ │ Provider Scanner│
│ │ (scans chain)  │                                   │ │ (OP_RETURN)     ││
│ └────────┬────────┘                                   │ └────────┬────────┘│
│          │                                            │          │         │
│          ▼                                            │          ▼         │
│ ┌─────────────────┐                                   │ ┌─────────────────┐│
│ │ UDP Echo Server│                                   │ │ Payment Sender │ │
│ │ (latency test) │                                   │ │ (OP_RETURN)    │ │
│ └─────────────────┘                                   │ └─────────────────┘│
└─────────────────────┘                                   └────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                           DATA FLOW DIAGRAMS                                │
└─────────────────────────────────────────────────────────────────────────────┘

PROVIDER SIDE (start-provider):
────────────────────────────────

  1. INIT                    2. ANNOUNCE                  3. SERVE
  ┌──────────────┐          ┌──────────────┐            ┌──────────────┐
  │ Generate/    │          │ Create TX    │            │ Listen on    │
  │ Load Keys   │─────────▶│ with         │───────────▶│ TCP:51820    │
  │ (secp256k1) │          │ OP_RETURN    │            │ (TLS)        │
  └──────────────┘          │ (VPN magic)  │            └──────┬───────┘
                             └──────┬───────┘                   │
                                    │                           │
                                    ▼                           ▼
                             ┌──────────────┐            ┌──────────────┐
                             │ Broadcast    │            │ UDP Echo     │
                             │ to Blockchain│            │ Server       │
                             └──────────────┘            └──────────────┘
                                    │
                                    ▼
                             ┌──────────────┐
                             │ Payment      │
                             │ Monitor      │◄────────── (scans blockchain)
                             │ (scans chain)│            for incoming payments
                             └──────────────┘


CLIENT SIDE (scan → connect):
──────────────────────────────

  1. DISCOVER              2. ENRICH                3. SELECT
  ┌──────────────┐         ┌──────────────┐         ┌──────────────┐
  │ Scan Blocks  │────────▶│ UDP Ping     │────────▶│ User Picks   │
  │ for OP_RETURN│         │ (latency)    │         │ Provider     │
  │ (VPN magic)  │         └──────────────┘         └──────┬───────┘
  └──────┬───────┘                                         │
         │                                                 ▼
         ▼                                        ┌──────────────┐
  ┌──────────────┐                                │ Generate     │
  │ Decode       │                                │ Temp Keys    │
  │ IP/Port/Price│                                └──────┬───────┘
  └──────────────┘                                       │
         │                                               ▼
         ▼                                       ┌──────────────┐
  ┌──────────────┐                                │ Create Payment│
  │ GeoIP Lookup │                                │ TX + OP_RETURN│
  │ (country)    │                                │ (PAY magic)  │
  └──────────────┘                                └──────┬───────┘
                                                         │
                                                         ▼
                                                  ┌──────────────┐
  4. CONNECT                                        │ Wait for     │
  ┌──────────────┐                                 │ Auth         │
  │ TLS Connect  │◄────────────────────────────────┘              │
  │ to Provider   │                                               │
  └──────┬───────┘                                               │
         │                                                        │
         ▼                                                        │
  ┌──────────────┐                                                │
  │ mTLS Handshake│◄────── (cert bound to pubkey)                │
  │ (cert verify) │                                                │
  └──────┬───────┘                                                │
         │                                                        │
         ▼                                                        │
  ┌──────────────┐                                                │
  │ Setup TUN    │◄────────────── (receive dynamic IP)           │
  │ Interface    │                                                │
  └──────┬───────┘                                                │
         │                                                        │
         ▼                                                        │
  ┌──────────────┐                                                │
  │ Forward      │◄──────── (all traffic through TLS tunnel)     │
  │ Traffic      │                                                │
  └──────────────┘                                                │


┌─────────────────────────────────────────────────────────────────────────────┐
│                        INTERNAL PACKAGES                                   │
└─────────────────────────────────────────────────────────────────────────────┘

cmd/bcvpn/           → CLI entrypoint
cmd/bcvpn-gui/       → GUI entrypoint (Fyne)

internal/
 ├── auth/          → Certificate generation, authorization
 ├── blockchain/    → RPC calls, provider/scanner, payment
 ├── config/        → Config loading & validation
 ├── crypto/        → Key storage (file/Keychain/DPAPI)
 ├── geoip/         → GeoLite2 lookups
 ├── history/       → Payment history
 ├── nat/           → UPnP / NAT-PMP
 ├── protocol/      → OP_RETURN encoding/decoding
 ├── tunnel/        → TUN device, TLS, packet forwarding
 └── obs/           → Logging, metrics
```
