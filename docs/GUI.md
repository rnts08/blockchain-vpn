# BlockchainVPN GUI Design

This document outlines the user interface for the desktop application version of BlockchainVPN. The application is designed to allow users to act as both a **VPN Provider** (seller) and a **VPN Client** (buyer) simultaneously, with a focus on ease of use and network stability.

## Current Implementation Notes

The current GUI implementation in `cmd/bcvpn-gui` provides:

- Provider control panel with editable provider networking/security settings (interface, listen port, NAT options, access policy files, cert/health settings, pricing).
- Provider panel includes explicit `Rebroadcast Service` and `Broadcast Price Update` actions matching CLI provider announcement commands.
- Client panel with provider discovery, connect flow, and editable client tunnel settings (interface, TUN IP/subnet).
- Client panel includes an `Enable Kill Switch` toggle for session-level traffic blocking outside tunnel.
- Dedicated Settings tab with RPC config, logging format/level, key-storage backend mode/service, revocation cache file, metrics auth token, and TLS policy fields.
- Preflight privilege checks before provider start and before non-dry-run client payment/connection.
- Status tab with config path, interface settings, privilege readiness summary, and provider/client metrics snapshot refresh.
- Status tab includes local doctor checks (config/privilege/tool/key-storage readiness) and version display.
- Status tab includes runtime event timeline and diagnostics bundle export.
- Wallet tab showing payment history from `history.json`.
- First-run setup wizard with steps for config readiness, RPC connectivity, provider key creation, and privilege checks.
- Auto-elevation relaunch action from the wizard (Linux/macOS/Windows backends).
- Optional runtime metrics endpoints configurable from Provider/Client panels.
- Settings tab supports profile import/export path actions.

Items in this document not yet implemented are tracked in `docs/TODO.md`.

## Actual Implementation Notes

The GUI is built with **Fyne** (fyne.io/fyne/v2) and has the following actual structure:

### Tab Structure (5 tabs)

1. **Provider Mode** - Provider configuration and control
2. **Client Mode** - Provider discovery and connection
3. **Network Status** - Runtime metrics, doctor checks, event timeline
4. **Settings** - RPC, security, logging configuration
5. **Wallet** - Payment history display

### First-Run Setup Wizard

When first launched (no setup-complete marker), the GUI shows a wizard with:
- Title: "First-Run Setup Wizard"
- Description: "Complete these checks to enable click-and-run provider/client operation."
- Elevation hint showing platform-specific privilege requirements
- 4 step buttons: "Ensure Config", "Check RPC Connectivity", "Create/Unlock Provider Key", "Check Networking Privileges"
- Password input field for provider key (step 3)
- "Relaunch Elevated" button (disabled on platforms without elevation support)
- "Finish Setup" button
- Status card showing setup progress

### Provider Mode Tab (Actual)

**Header:** "Provider Control" (bold, left-aligned)

**Form Fields (widget.NewForm):**
- Interface Name (text entry, default: "bcvpn0")
- Listen Port (entry, default: 51820)
- Announce IP (optional, entry)
- Country (entry, e.g., "US")
- Price (sats/session, entry)
- Max Consumers (entry, 0=unlimited)
- Bandwidth Limit (entry, e.g., "10mbit")
- Provider TUN IP (entry, e.g., "10.10.0.1")
- Provider TUN Subnet (entry, e.g., "24")
- NAT Traversal checkbox ("Enable NAT Traversal (UPnP/NAT-PMP)")
- Provider Egress NAT checkbox
- NAT Outbound Interface (entry)
- Isolation Mode dropdown ("none", "sandbox")
- Allowlist File (entry)
- Denylist File (entry)
- Cert Lifetime Hours (entry, default: 720)
- Rotate Before Hours (entry, default: 24)
- Health Checks checkbox
- Health Check Interval (entry, e.g., "30s")
- Metrics Listen Addr (entry, e.g., "127.0.0.1:9090")
- Key Password (password entry, "file mode only")

**Action Buttons (grid, 6 columns):**
- Save Provider Config
- Auto-Locate Country
- Start Provider
- Stop Provider
- Rebroadcast Service
- Broadcast Price Update

**Secondary Row (2 columns):**
- Rotate Provider Key
- Status Label ("Status: stopped" / "Status: running")

**Log Panel:**
- Card titled "Activity Log" with "Runtime events, errors, and actions"
- Multi-line entry showing timestamped log messages
- 8 visible rows, scrollable

### Client Mode Tab (Actual)

**Header:** "Client Discovery & Connect" (bold)

**Sort/Filter Row (12 columns):**
- Label "Sort:" + dropdown ("latency", "price", "country", "bandwidth", "capacity", "score")
- Label "Country:" + entry (placeholder: "Country filter e.g. US")
- Label "Max Price:" + entry (placeholder: "Max price sats (optional)")
- Label "Min BW:" + entry (placeholder: "Min bandwidth Kbps")
- Label "Max Latency:" + entry (placeholder: "Max latency ms")
- Label "Min Slots:" + entry (placeholder: "Min available slots")

**Settings Row (8 columns):**
- Label "Interface" + entry (default: "bcvpn1")
- Label "TUN IP" + entry (e.g., "10.10.0.2")
- Label "Subnet" + entry (e.g., "24")
- Label "Metrics" + entry

**Action Row (6 columns):**
- Scan Providers button
- Connect Selected button
- Disconnect All button
- Save Client Settings button
- Enable Kill Switch checkbox
- Dry run checkbox ("Dry run (no payment, no interface changes)")

**Security Options Row (2 columns):**
- Strict Verification checkbox
- Verify Throughput After Connect checkbox

**Provider List Card:**
- Title: "Provider List"
- Subtitle: "Latency, price, and country-enriched endpoint table"
- widget.NewList showing results in format: `[index] IP:port | price sats | latency | bandwidth Kbps | cap=X | score=X.X`

**Log Panel:**
- Same as Provider tab

### Network Status Tab (Actual)

**Header:** "Network Status" (bold)

**Version/Config Info:**
- Version label (e.g., "Version: X.Y.Z")
- Config Path label
- Provider Interface label (name + TUN IP/subnet)
- Client Interface label
- Client Kill Switch status label
- Privileges status label ("Privileges: OK" or error message)

**Runtime Metrics Card:**
- Title: "Runtime Metrics"
- Subtitle: "Provider/client runtime metrics endpoint snapshots"
- Refresh Metrics button
- Multi-line entry (8 rows) showing:
  - Provider Metrics Addr
  - Client Metrics Addr  
  - Metrics Auth configured status
  - Metrics endpoint JSON data

**Doctor Card:**
- Title: "Doctor"
- Subtitle: "Config/privilege/tool readiness checks"
- Run Doctor Checks button
- Multi-line entry (8 rows) showing check results:
  - config.validate status
  - security.keystore status
  - networking.privileges status
  - tool.* status (ip, iptables, ifconfig, etc.)
  - security.metrics_auth status

**Event Timeline Card:**
- Title: "Event Timeline"
- Subtitle: "Recent runtime session and auth events"
- Refresh Events button
- Multi-line entry (8 rows) showing events: timestamp [role] type: detail

**Export Button:**
- "Export Diagnostics Bundle" - exports JSON to app config dir

**Log Panel:**
- Same as Provider tab

### Settings Tab (Actual)

**Header:** "Global Settings" (bold)

**Hint:** "Validation hints: host required, ports 1-65535, valid IP/prefix, valid health_check_interval duration (e.g. 30s)."

**RPC Card:**
- Title: "RPC"
- Subtitle: "Global daemon connection settings"
- Form fields:
  - RPC Host (entry, default: "localhost:18443")
  - RPC User (entry)
  - RPC Pass (password entry)
  - Key Storage Mode dropdown ("file", "auto", "keychain", "libsecret", "dpapi")
  - Key Storage Service (entry)
  - Revocation Cache File (entry)
  - TLS Min Version dropdown ("1.3", "1.2")
  - TLS Profile dropdown ("modern", "compat")
  - Metrics Auth Token (password entry)
  - Log Format dropdown ("text", "json")
  - Log Level dropdown ("debug", "info", "warn", "error")

**Profile Path Row (3 columns):**
- Label "Profile Path" + entry (placeholder) + Export Profile button + Import Profile button

**Action Buttons (3 columns):**
- Save + Validate
- Validate Current Config
- Apply Defaults For Empty Fields

**Validation Output Card:**
- Multi-line entry (6 rows) showing validation results

**Log Panel:**
- Same as Provider tab

### Wallet Tab (Actual)

**Header:** "Wallet & History" (bold)

**Reload Button:** "Reload History"

**Payment History Card:**
- Title: "Payment History"
- Subtitle: "Most recent transactions"
- Multi-line entry (word wrapping, disabled) showing:
  - Format: `RFC3339 timestamp | amount sats | provider address | txid`

### Visual Theme

- Primary color: Green (#0C5C40)
- Background: Off-white (#F6F5EF)
- Button color: Teal (#1C8062)
- Font: Default Fyne theme

### Missing from Design Doc

The GUI does NOT implement:
- Speed test functionality (mentioned in design)
- Real-time traffic graphs (Status tab only shows static metrics on refresh)
- Bandwidth speed/duration inputs for client connection
- Auto-disconnect latency threshold input
- Payout address field in Provider (uses internal wallet)
- Visual provider session table with peer details
- Traffic counters (Total Sent/Received)
- Financial overview (Balance, Session Cost, Total Earned/Spent)
- Network conflict detection modal

## Layout Overview

The application uses a **Tabbed Layout** to separate distinct functions.

**Tabs:**
1.  **Provider Mode** (Sell Bandwidth)
2.  **Client Mode** (Buy VPN)
3.  **Status** (Runtime and readiness summary)
4.  **Settings** (RPC, logging, security policy)
5.  **Wallet** (History)

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
|  OCX BLOCKCHAIN VPN                             [ Wallet: 500.42 OCX ]|
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
|  Payout To:  [ ocx1qxy2...89a ]                                       |
|                                                                       |
|  -------------------------------------------------------------------  |
|                                                                       |
|  STATUS:  🔴 STOPPED               [ START SERVICE ]                  |
|                                                                       |
+-----------------------------------------------------------------------+
```
