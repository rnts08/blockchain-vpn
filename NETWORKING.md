# Networking Configuration for Split Routing

This document details the Linux commands required to implement the "Split Routing" logic described in `GUI.md`. This configuration allows a user to run a VPN Provider (Server) and a VPN Client simultaneously without routing loops.

## Variables

Adjust these variables to match your system configuration:

```bash
PHY_IFACE="eth0"          # Your physical network interface (e.g., eth0, wlan0)
PHY_GW="192.168.1.1"      # Your physical default gateway
PROVIDER_PORT="51820"     # The TCP port your Provider node listens on
PROVIDER_IFACE="bcvpn0"     # The TUN interface for your Provider clients
CLIENT_IFACE="bcvpn1"       # The TUN interface for your outgoing Client connection
MARK_ID="0x100"           # Arbitrary firewall mark ID
TABLE_ID="200"            # Arbitrary routing table ID
```

## 1. Policy Based Routing (PBR)

When the Client VPN (`wg1`) is active, it typically replaces the default route. We must ensure that traffic related to the Provider service (incoming handshakes and their replies) bypasses `wg1` and exits via the physical interface.

### Step A: Create a separate routing table

Add a default route to the custom table that points to the physical gateway.

```bash
ip route add default via $PHY_GW dev $PHY_IFACE table $TABLE_ID
```

### Step B: Mark Provider Traffic

Use `iptables` to mark outgoing packets that originate from the Provider's listening port. This identifies replies to incoming VPN clients.

```bash
iptables -t mangle -A OUTPUT -p udp --sport $PROVIDER_PORT -j MARK --set-mark $MARK_ID
```

### Step C: Create a Routing Rule

Tell the kernel to use the custom table (which routes via `eth0`) for any packet bearing the mark.

```bash
ip rule add fwmark $MARK_ID lookup $TABLE_ID
```

*Note: You may also need to flush the route cache for changes to take effect immediately:*
```bash
ip route flush cache
```

## 2. Firewall Rules

Ensure traffic is allowed in and out of the Provider interface.

### Input (Allow external clients to connect)
```bash
iptables -A INPUT -p udp --dport $PROVIDER_PORT -j ACCEPT
```

### Forwarding (Allow Provider clients to access the Internet)
These rules allow traffic from your Provider clients (`wg0`) to exit via your physical interface (`eth0`), performing NAT (Masquerade) so they share your IP.

```bash
iptables -A FORWARD -i $PROVIDER_IFACE -o $PHY_IFACE -j ACCEPT
iptables -A FORWARD -i $PHY_IFACE -o $PROVIDER_IFACE -m state --state RELATED,ESTABLISHED -j ACCEPT
iptables -t nat -A POSTROUTING -o $PHY_IFACE -j MASQUERADE
```