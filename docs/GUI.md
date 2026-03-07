# BlockchainVPN GUI Design

This document outlines the user interface for the desktop application version of BlockchainVPN. The application is designed to allow users to act as both a **VPN Provider** (seller) and a **VPN Client** (buyer) simultaneously, with a focus on ease of use and network stability.

## Layout Overview

The application uses a **Tabbed Layout** to separate distinct functions.

**Tabs:**
1.  **Provider Mode** (Sell Bandwidth)
2.  **Client Mode** (Buy VPN)
3.  **Network Status** (Dashboard)
4.  **Wallet** (Settings & Keys)

---

## 1. Tab: Provider Mode (Sell Bandwidth)

This tab configures the local machine to act as a VPN server.

### Configuration Panel

*   **Service Location:**
    *   [Dropdown Menu] Country List.
    *   `[BUTTON] Auto-Locate` (Uses GeoIP to detect and set country automatically).

*   **Bandwidth Management:**
    *   **Max Upload Speed:** `[Input Field] Mbps` (0 for unlimited).
    *   **Max Download Speed:** `[Input Field] Mbps` (0 for unlimited).
    *   `[BUTTON] Run Speed Test`: Runs an internal speed test (e.g., via Ookla or fast.com API) to help the user determine their actual limits before setting caps.

*   **Capacity & Pricing:**
    *   **Max Concurrent Users:** `[Input Field]` (e.g., 5).
    *   **Price per Hour:** `[Input Field] ORDX` (or Satoshis).
    *   **Payout Address:** `[Input Field]` (Destination for earnings).
        *   *Hint: Defaults to the internal wallet address.*

*   **Service Control:**
    *   **Status Indicator:** 🔴 Stopped / 🟢 Running
    *   `[TOGGLE SWITCH] Enable Provider Mode`
    *   *Note: Enabling this broadcasts the Service Announcement transaction to the blockchain.*

### Active Sessions (Provider)
*   **Table View:**
    *   Peer Public Key (Short) | Connected Time | Data Used | Current Rate | Status

---

## 2. Tab: Client Mode (Buy VPN)

This tab allows the user to browse the blockchain for available providers and connect.

### Search & Filter

*   **Filters:**
    *   **Country:** `[Dropdown]` (e.g., "Any", "United States", "Germany").
    *   **Min Speed:** `[Slider]` (e.g., 10 Mbps+).
    *   **Max Price:** `[Input]` ORDX/hr.

*   **Sort By:**
    *   `[Radio Buttons]` Speed (Latency) | Cost (Lowest) | Bandwidth (Highest).

### Provider List (Data Grid)

| Country | Latency (Ping) | Bandwidth Claim | Price/Hr | Rating | Action |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 🇺🇸 US | 45ms | 100 Mbps | 50 sats | ⭐⭐⭐⭐⭐ | `[Connect]` |
| 🇩🇪 DE | 120ms | 500 Mbps | 10 sats | ⭐⭐⭐⭐ | `[Connect]` |

*   *Note: The "Latency" column is populated by active probing (UDP Echo) performed by the client in the background.*

### Connection Settings

*   **Payment Source:** `[Dropdown]` Select Wallet Address / UTXO to pay from.
*   **Auto-Disconnect:** `[Checkbox]` Disconnect if latency > `[Input]` ms.
*   **Kill Switch:** `[Checkbox]` Block all traffic if VPN drops.

---

## 3. Tab: Network Status

A unified dashboard showing the health of both Client and Provider interfaces.

### Visualizations

*   **Traffic Graph:** Real-time line chart showing Upload (Green) and Download (Blue) throughput.
*   **Data Counters:**
    *   Total Sent: `1.2 GB`
    *   Total Received: `4.5 GB`

### Financial Overview

*   **Wallet Balance:** `1,050.00 OCX`
*   **Session Cost (Current):** `50 sats / hr`
*   **Total Earned (Provider):** `5,000 sats`
*   **Total Spent (Client):** `200 sats`

### Interface Status

*   **Physical Interface (eth0):** `192.168.1.5` (Up)
*   **Provider Tunnel (bcvpn0):** `10.10.0.1` (Listening on TCP :51820)
*   **Client Tunnel (bcvpn1):** `10.10.0.2` (Connected to Peer X)

---

## 4. Networking Logic (Avoiding Dead Routing)

To ensure a user can be both a Provider and a Client simultaneously without creating routing loops (e.g., routing the Provider's incoming VPN traffic *through* the Client's outgoing VPN tunnel), the GUI application enforces specific networking rules.

### Split Routing & Policy Based Routing (PBR)

When both modes are active:

1.  **Provider Binding:**
    *   The Provider listener (UDP port) is explicitly bound to the **Physical Interface IP**, not `0.0.0.0`.
    *   *Reason:* Prevents the listener from accepting packets via the Client VPN tunnel interface.

2.  **Routing Tables:**
    *   **Client Traffic:** The application creates a specific routing table for the Client VPN tunnel (`bcvpn1`).
    *   **Provider Traffic:** Incoming traffic destined for the Provider port is marked (fwmark) and routed via the default gateway (Physical Interface), bypassing the Client VPN tunnel.

3.  **Firewall Rules (platform-specific backend):**
    *   **Input:** Allow UDP traffic on Provider Port on Physical Interface.
    *   **Forward:** Allow traffic from `bcvpn0` (Provider Clients) to Physical Interface (Internet).
    *   **Output:** Ensure replies to Provider Clients leave via Physical Interface.

### Visual Warning

If the application detects a potential conflict (e.g., the user tries to connect to a Provider that resolves to their own IP, or port conflicts), a modal warning appears:

> **⚠️ Network Conflict Detected**
> You are attempting to connect to a VPN while running a Provider node on the same port.
> The application will automatically adjust routing tables to prevent loops.

---

## 5. Mockup (ASCII)

```text
+-----------------------------------------------------------------------+
|  BLOCKCHAIN VPN                                      [ Wallet: 500 $ ]|
+-----------------------------------------------------------------------+
|  [ PROVIDER ]    [ CLIENT ]    [ STATUS ]    [ SETTINGS ]             |
+-----------------------------------------------------------------------+
|                                                                       |
|  PROVIDER CONFIGURATION                                               |
|                                                                       |
|  Location:   [ United States (US) [v] ]  [ Auto-Locate ]              |
|                                                                       |
|  Bandwidth:  Up: [ 50 ] Mbps   Down: [ 100 ] Mbps   [ Test Speed ]    |
|                                                                       |
|  Capacity:   Max Users: [ 5 ]                                         |
|                                                                       |
|  Pricing:    [ 100 ] Sats/Hour                                        |
|                                                                       |
|  Payout To:  [ ord1qxy2...89a ]                                       |
|                                                                       |
|  -------------------------------------------------------------------  |
|                                                                       |
|  STATUS:  🔴 STOPPED               [ START SERVICE ]                  |
|                                                                       |
+-----------------------------------------------------------------------+
```
