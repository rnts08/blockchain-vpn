# BlockchainVPN TUI Design Specification

This document outlines the Terminal User Interface (TUI) design for BlockchainVPN using the Charmbracelet/Tea framework.

**Color Scheme:**
- Primary/Highlight: `#dca747` (gold/amber)
- Background: `#212121` (dark gray)
- Text: `#e0e0e0` (light gray)
- Success: `#4caf50` (green)
- Error: `#f44336` (red)

---

## 1. Main Application Layout

```
┌──────────────────────────────────────────────────────────────────────────────┐
│  [ STATUS ]  [ PROVIDER ]  [ CONNECT ]  [ STATS ]              [?]Help    │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│                              MAIN CONTENT AREA                               │
│                                                                              │
│                                                                              │
│                                                                              │
│                                                                              │
│                                                                              │
│                                                                              │
├──────────────────────────────────────────────────────────────────────────────┤
│  STATUS: Connected  │  MODE: Provider  │  ↓ 0.0 KB/s  ↑ 0.0 KB/s  │ RPC: ✓   │
│  BAL: 50,000 sats   │  TUN: bcvpn0     │  CONN: 2/10                         │
└──────────────────────────────────────────────────────────────────────────────┘
```

**Navigation:**
- `Tab` / `Shift+Tab`: Navigate between tabs
- `←` / `→`: Navigate between tabs
- `C`: Open configuration editor
- `Q`: Quit
- `?`: Show help

---

## 2. Tab 1: Status Overview (Provider Mode)

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ ●STATUS   │  PROVIDER  │  CONNECT  │  STATS                               │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ╔═══════════════════════════════════════════════════════════════════════╗  │
│  ║                        SYSTEM STATUS                                  ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║  Mode:              PROVIDER                                          ║  │
│  ║  Status:            RUNNING                                           ║  │
│  ║  External IP:       45.33.22.11 (Rotated)                            ║  │
│  ║  Country:           US (United States)                               ║  │
│  ║  Public Key:        02a1...b4c3                                       ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║                        CONNECTIONS                                     ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║  Active:             2 / 10                                           ║  │
│  ║  Total Sessions:    147                                                ║  │
│  ║  Uptime:            3d 14h 22m                                        ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║                        BANDWIDTH (Session)                             ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║  Downloaded:       1.24 GB    │  Uploaded:      456.78 MB             ║  │
│  ║  Current:           ↓ 2.4 MB/s │  ↑ 512 KB/s                         ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║                        PAYMENT                                         ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║  Model:             TIME (per minute)                                 ║  │
│  ║  Price:             10 sats/minute                                    ║  │
│  ║  Earned:            12,450 sats                                        ║  │
│  ║  Last Payment:      2 minutes ago                                      ║  │
│  ╚═══════════════════════════════════════════════════════════════════════╝  │
│                                                                              │
├──────────────────────────────────────────────────────────────────────────────┤
│ ●STATUS  │PROVIDER│CONNECT│STATS│  [R]efresh  [C]onfig  [Q]uit           │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Status Overview (Client Mode)

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ ●STATUS   │  PROVIDER  │  CONNECT  │  STATS                               │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ╔═══════════════════════════════════════════════════════════════════════╗  │
│  ║                        SYSTEM STATUS                                  ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║  Mode:              CONSUMER                                          ║  │
│  ║  Status:            CONNECTED                                         ║  │
│  ║  External IP:       45.33.22.11 (via VPN)                            ║  │
│  ║  Country:           DE (Germany)                                      ║  │
│  ║  Provider:          vpn.provider.com:51820                            ║  │
│  ║  Session Time:      00:45:23                                         ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║                        BANDWIDTH (This Session)                        ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║  Downloaded:       456.78 MB    │  Uploaded:      123.45 MB           ║  │
│  ║  Current:           ↓ 5.2 MB/s  │  ↑ 1.1 MB/s                         ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║                        PAYMENT                                         ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║  Model:             DATA (per GB)                                     ║  │
│  ║  Price:             100 sats/GB                                        ║  │
│  ║  Spent:            234 sats                                           ║  │
│  ║  Balance:           45,000 sats                                        ║  │
│  ║  Auto-Recharge:    Enabled (>500 sats)                               ║  │
│  ╚═══════════════════════════════════════════════════════════════════════╝  │
│                                                                              │
├──────────────────────────────────────────────────────────────────────────────┤
│ ●STATUS  │PROVIDER│CONNECT│STATS│  [R]efresh  [C]onfig  [Q]uit           │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Tab 2: Provider Setup

### Provider Configuration (Stopped)

```
┌──────────────────────────────────────────────────────────────────────────────┐
│   STATUS  │ ●PROVIDER │  CONNECT  │  STATS                                │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ╔═══════════════════════════════════════════════════════════════════════╗  │
│  ║                    PROVIDER CONFIGURATION                            ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║                                                                        ║  │
│  ║  Country:             [US        ▼]  Auto-detect: [●]                ║  │
│  ║                                                                        ║  │
│  ║  Connection Limits                                                      ║  │
│  ║  ├─ Max Consumers:    [10    ] (0 = unlimited)                       ║  │
│  ║  └─ Max Duration:    [0    ] hours (0 = unlimited)                    ║  │
│  ║                                                                        ║  │
│  ║  Bandwidth                                                           ║  │
│  ║  ├─ Limit:           [10mbit  ]                                       ║  │
│  ║  └─ Advertised:      [10     ] Mbps                                   ║  │
│  ║                                                                        ║  │
│  ║  Payment Model        [SESSION ●] [TIME] [DATA]                      ║  │
│  ║  ├─ Price:           [100   ] satoshis                                ║  │
│  ║  ├─ Time Unit:       [MINUTE ▼] (minute/hour)                        ║  │
│  ║  └─ Data Unit:       [MB    ▼] (MB/GB)                               ║  │
│  ║                                                                        ║  │
│  ║  Payment Address:     [bc1qxy...z5m ]  [Generate New]                 ║  │
│  ║                                                                        ║  │
│  ║  Advanced Options                                                     ║  │
│  ║  ├─ Enable NAT:       [●]  UPnP [●] NAT-PMP                         ║  │
│  ║  ├─ Isolation:        [NONE   ▼] (none/sandbox)                      ║  │
│  ║  └─ Health Checks:   [●]  Interval: [30s]                           ║  │
│  ║                                                                        ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║  Blockchain Status                                                    ║  │
│  ║  ├─ Announcement:     ANNOUNCED (height: 842,153)                    ║  │
│  ║  ├─ Last Update:      2 hours ago                                    ║  │
│  ║  └─ Re-announce:      [Force Re-announce]                           ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║                                                                        ║  │
│  ║              [ Validate ]  [ Save Config ]  [ START ]  [ STOP ]     ║  │
│  ║                                                                        ║  │
│  ╚═══════════════════════════════════════════════════════════════════════╝  │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Provider Running State

```
┌──────────────────────────────────────────────────────────────────────────────┐
│   STATUS  │ ●PROVIDER │  CONNECT  │  STATS                                │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ╔═══════════════════════════════════════════════════════════════════════╗  │
│  ║                    PROVIDER - RUNNING                                 ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║                                                                        ║  │
│  ║  ● LISTENING ON PORT 51820 (TLS)                                     ║  │
│  ║  ● NAT PORT MAPPED (TCP:51820, UDP:51820)                            ║  │
│  ║  ● ANNOUNCED TO BLOCKCHAIN                                            ║  │
│  ║  ● HEALTH CHECKS ACTIVE                                               ║  │
│  ║                                                                        ║  │
│  ║  ═════════════════════════════════════════════════════════════════    ║  │
│  ║                                                                        ║  │
│  ║  Active Connections:                                                   ║  │
│  ║  ┌────────────────────────────────────────────────────────────────┐    ║  │
│  ║  │ ID   │ IP Address     │ Country │ Download  │ Upload  │ Time  │    ║  │
│  ║  ├──────┼─────────────────┼─────────┼───────────┼─────────┼───────┤    ║  │
│  ║  │ 001  │ 10.0.0.2       │ DE      │ 45.2 MB  │ 12.1 MB │ 5:23  │    ║  │
│  ║  │ 002  │ 10.0.0.3       │ FR      │ 12.8 MB  │ 3.2 MB  │ 2:10  │    ║  │
│  ║  └────────────────────────────────────────────────────────────────┘    ║  │
│  ║                                                                        ║  │
│  ║  ═════════════════════════════════════════════════════════════════    ║  │
│  ║                                                                        ║  │
│  ║  Revenue This Session: 450 sats                                       ║  │
│  ║  Total Revenue: 12,450 sats                                           ║  │
│  ║                                                                        ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║                                                                        ║  │
│  ║                       [ UPDATE PRICE ]  [ STOP PROVIDER ]            ║  │
│  ║                                                                        ║  │
│  ╚═══════════════════════════════════════════════════════════════════════╝  │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Tab 3: Connect to VPN (Provider Discovery)

### Provider List

```
┌──────────────────────────────────────────────────────────────────────────────┐
│   STATUS  │  PROVIDER  │ ●CONNECT │  STATS                                 │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Search: [________________________]  [🔍 Search]  [↻ Rescan]               │
│                                                                              │
│  Filters:  Country: [ALL    ▼]  Min BW: [0  ] Mbps  Max Price: [0   ] sats │
│           Payment: [ALL    ▼]  Min Slots: [0  ]                          │
│                                                                              │
│  ═══════════════════════════════════════════════════════════════════════   │
│                                                                              │
│  ╔═══════════════════════════════════════════════════════════════════════╗  │
│  ║  PROVIDERS FOUND: 47                                    Sort: [PRICE] ║  │
│  ╠════════════════════════════╦════════════════════════════════════════╣  │
│  ║  #   Country  Host         │ Latency  Slots  Price   Payment  Actions║  │
│  ╠════════════════════════════╬════════════════════════════════════════╣  │
│  ║> 1   DE  1.2.3.4:51820    │ 45ms     8/10   100/s   session  [CONN] ║  │
│  ║  2   FR  2.3.4.5:51820    │ 62ms     5/10   150/s   session  [CONN] ║  │
│  ║  3   US  3.4.5.6:51820    │ 89ms     0/5    200/s   session  [CONN] ║  │
│  ║  4   GB  4.5.6.7:51820     │ 78ms     10/10  50/s    time     [CONN] ║  │
│  ║  5   JP  5.6.7.8:51820    │ 156ms    3/5    100/GB  data     [CONN] ║  │
│  ║  6   CA  6.7.8.9:51820    │ 95ms     2/10   75/s    session  [CONN] ║  │
│  ║  7   AU  7.8.9.0:51820    │ 245ms   10/10  200/s   session  [CONN] ║  │
│  ║  8   DE  8.9.0.1:51820    │ 48ms     6/10   80/s    time     [CONN] ║  │
│  ╚════════════════════════════╩════════════════════════════════════════╝  │
│                                                                              │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Provider Details Panel

```
┌──────────────────────────────────────────────────────────────────────────────┐
│   STATUS  │  PROVIDER  │ ●CONNECT │  STATS                                 │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Selected: #1 - 1.2.3.4:51820 (DE)                                         │
│  ╔═══════════════════════════════════════════════════════════════════════╗  │
│  ║  DETAILS                                         [ Speed Test ]      ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║  Country:           Germany (DE)                                     ║  │
│  ║  Bandwidth:         100 Mbps (Advertised: 10 Mbps)                     ║  │
│  ║  Max Consumers:     10                                                 ║  │
│  ║  Available Slots:    8                                                   ║  │
│  ║  Payment Model:     Per Session                                        ║  │
│  ║  Price:            100 sats                                           ║  │
│  ║  Reputation:       ★★★★☆ (42 reviews)                                 ║  │
│  ║  Protocol:        TLS 1.3 + WireGuard                                 ║  │
│  ║  NAT Traversal:    UPnP + NAT-PMP                                      ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║  PRE-FLIGHT CHECKS                                                    ║  │
│  ║  [✓] Payment Possible      [✓] Balance Sufficient (50,000 sats)      ║  │
│  ║  [✓] RPC Connected        [✓] Tunnel Available                        ║  │
│  ║                                                                        ║  │
│  ╠═══════════════════════════════════════════════════════════════════════╣  │
│  ║                                                                        ║  │
│  ║        [ ← Back ]        [ CONNECT ]        [ Test Speed → ]          ║  │
│  ║                                                                        ║  │
│  ╚═══════════════════════════════════════════════════════════════════════╝  │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Connection Progress

```
┌──────────────────────────────────────────────────────────────────────────────┐
│   STATUS  │  PROVIDER  │ ●CONNECT │  STATS                                 │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│                                                                              │
│                     CONNECTING TO PROVIDER...                               │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  [████████████████████████░░░░░░░░] 60%                            │   │
│  │                                                                      │   │
│  │  Step 1: Creating Payment Transaction...  ✓                        │   │
│  │  Step 2: Waiting for Confirmation........  ✓                        │   │
│  │  Step 3: Establishing TLS Tunnel........  ████████░░░              │   │
│  │  Step 4: Setting up TUN Interface...... ⏳                        │   │
│  │  Step 5: Running Security Tests........ ⏳                        │   │
│  │                                                                      │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Payment: 100 sats to bc1q...xyz (tx: 842a1...b3c2)                        │
│                                                                              │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Tab 4: Real-time Statistics

```
┌──────────────────────────────────────────────────────────────────────────────┐
│   STATUS  │  PROVIDER  │  CONNECT  │  [S]TATS                                │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ╔══════════════════════╗  ╔══════════════════════════════════════════════╗  │
│  ║   NETWORK STATS      ║  ║      REAL-TIME BANDWIDTH (60s window)        ║  │
│  ╠══════════════════════╣  ╠══════════════════════════════════════════════╣  │
│  ║                      ║  ║                                              ║  │
│  ║  Local TUN IP:       ║  ║  ↑ 5.2 MB/s ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░  512 KB/s ║  │
│  ║  10.0.0.2/24         ║  ║                                              ║  │
│  ║                      ║  ║  ↓ 12.4 MB/s ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓  1.2 MB/s  ║  │
│  ║  Remote Endpoint:    ║  ║                                              ║  │
│  ║  1.2.3.4:51820       ║  ╠══════════════════════════════════════════════╣  │
│  ║                      ║  ║      SESSION PROGRESS                        ║  │
│  ║  Tunnel Status:      ║  ║                                              ║  │
│  ║    ENCRYPTED         ║  ║  ┌─────────────────────────────────────────┐ ║  │
│  ║                      ║  ║  │ ████████████████████░░░░░░░░░░░░░░░░░░  │ ║  │
│  ║  Cert Valid Until:   ║  ║  │ Time: 00:45:23  |  Data: 1.24 GB        │ ║  │
│  ║  2026-03-15 14:22    ║  ║  │ Cost: 234 sats                          │ ║  │
│  ║                      ║  ║  └─────────────────────────────────────────┘ ║  │
│  ║  Connection Time:    ║  ╠══════════════════════════════════════════════╣  │
│  ║  00:45:23            ║  ║             COST / PROFIT HISTORY            ║  │
│  ║                      ║  ║                                              ║  │
│  ║  DNS Leak Test:      ║  ║ 25000 ┤                       ╱              ║  │
│  ║    PROTECTED         ║  ║       │                      ╱               ║  │
│  ║                      ║  ║       │                     ╱                ║  │
│  ║  Kill Switch:        ║  ║       │        ╱───────────╱                 ║  │
│  ║    ENABLED           ║  ║       │       ╱╱                             ║  │
│  ╚══════════════════════╝  ║  5000 ┼──────╱─────────────────────────────  ║  │
│                            ║       └────┬────┬────┬────┬────┬────┬────    ║  │
│                            ║            10m  20m  30m  40m  50m  60m      ║  │
│                            ╚══════════════════════════════════════════════╝  │
│                                                                              │
│  Session Total: ↓ 1.24 GB  ↑ 456.78 MB  │  Est. Cost: 234 sats               │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Configuration Editor (C key)

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                         CONFIGURATION EDITOR                                │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─[RPC SETTINGS]───────────────────────────────────────────────────────┐   │
│  │  Host:           [localhost:25173                                  ]  │   │
│  │  User:           [rpcuser                                         ]  │   │
│  │  Password:       [••••••••••••                                    ]  │   │
│  │  Network:       [MAINNET ▼]  Token Symbol: [ORDEX]                │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─[SECURITY]───────────────────────────────────────────────────────────┐   │
│  │  Key Storage:   [FILE    ▼]                                       │   │
│  │  TLS Min Ver:   [1.3    ▼]  Profile: [MODERN ▼]                   │   │
│  │  Metrics Token: [••••••••••••                                    ]  │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─[PROVIDER]───────────────────────────────────────────────────────────┐   │
│  │  Interface:    [bcvpn0  ]  Listen Port: [51820]                    │   │
│  │  TUN IP:       [10.0.0.1]  Subnet: [/24 ▼]                        │   │
│  │  DNS Servers:  [1.1.1.1, 8.8.8.8                                ]  │   │
│  │  Announce IP:  [                    ] (empty = auto-detect)        │   │
│  │  NAT:          [●]  Kill Switch: [ ]                              │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─[CLIENT]─────────────────────────────────────────────────────────────┐   │
│  │  Interface:    [bcvpn1  ]                                         │   │
│  │  Max Tunnels:  [1   ]  Kill Switch: [●]                           │   │
│  │  Strict Verify:[ ]  WebSocket Fallback: [ ]                          │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─[SPENDING LIMITS]────────────────────────────────────────────────────┐   │
│  │  Enabled:     [ ]  Limit: [     ] sats  Warning: [80]%           │   │
│  │  Auto-Disconnect: [ ]  Max Per Session: [     ] sats              │   │
│  │  Auto-Recharge: [ ]  Threshold: [     ]  Amount: [     ] sats     │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─[LOGGING]─────────────────────────────────────────────────────────────┐   │
│  │  Format:       [TEXT ▼]  Level: [INFO ▼]                          │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│                    [ Cancel ]                    [ Save & Apply ]           │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Bottom Status Bar

```
┌──────────────────────────────────────────────────────────────────────────────┐
│  CONNECTED │ ↓ 5.2 MB/s │ ↑ 1.1 MB/s      │ RPC: ✓ │ BAL: 50K sats │ TUN: ✓  │
└──────────────────────────────────────────────────────────────────────────────┘

Status Bar States:
  ● = Running/Active (green)
  ○ = Stopped (dim)
  ◐ = Connecting/Transitioning (yellow)

Status Bar Elements:
  - Connection status (CONNECTED/DISCONNECTED/PROVIDING/SCANNING)
  - Current download speed
  - Current upload speed  
  - RPC connection status (✓ / ✗)
  - Balance in sats
  - TUN interface status (✓ / ✗ / -)
```

---

## 8. Help Overlay (? key)

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                           KEYBOARD SHORTCUTS                                │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Navigation                                                                 │
│  ─────────────────────────────────────────────────────────────────────────   │
│  Tab / Shift+Tab    Next / Previous tab                                    │
│  ← / →              Navigate tabs                                          │
│  j / k              Down / Up (in lists)                                   │
│  Enter              Select / Confirm                                        │
│  Esc                Back / Cancel                                          │
│                                                                              │
│  Actions                                                                    
│  ─────────────────────────────────────────────────────────────────────────   │
│  C                   Open configuration editor                             │
│  R                   Refresh / Rescan                                       │
│  S                   Save / Start                                           │
│  T                   Test speed                                             │
│  Q                   Quit                                                   │
│  ?                   Show this help                                         │
│                                                                              │
│  Provider Mode                                                                
│  ─────────────────────────────────────────────────────────────────────────   │
│  A                   Announce to blockchain                                │
│  U                   Update price                                           │
│  P                   Pause provider                                         │
│                                                                              │
│  Consumer Mode                                                               
│  ─────────────────────────────────────────────────────────────────────────   │
│  F                   Filter providers                                        │
│  /                   Search                                                 │
│  C                   Connect to selected                                    │
│  D                   Disconnect                                             │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## 9. Implementation Notes

### Technology Stack
- **Framework**: Charmbracelet/Tea (github.com/charmbracelet/tea)
- **UI Components**: 
  - `lipgloss` for styling (colors: #dca747, #212121)
  - `bubbles` for inputs, spinners, progress
  - `bubbletea` for the main loop

### Key Components to Implement
1. **Main Model**: Manages tab state and global status
2. **Status Tab Model**: Displays system overview
3. **Provider Tab Model**: Configuration and control
4. **Connect Tab Model**: Provider search and connection
5. **Stats Tab Model**: Real-time graphs and metrics
6. **Config Model**: Full configuration editor

### Data Flow
- Status updates via polling `/metrics.json` endpoint
- Provider discovery via blockchain scan
- Real-time stats via metrics polling (1-second interval)
