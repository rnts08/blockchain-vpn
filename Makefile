# Makefile for BlockchainVPN

BINARY=bcvpn
GO=go

# Networking Variables (Default values, can be overridden via environment variables)
PHY_IFACE?=eth0
PHY_GW?=192.168.1.1
PROVIDER_PORT?=51820     # The TCP port your Provider node listens on
PROVIDER_IFACE?=bcvpn0
CLIENT_IFACE?=bcvpn1
MARK_ID?=0x100
TABLE_ID?=200

.PHONY: all build clean apply-networking remove-networking check-networking

all: build

build:
	$(GO) build -o $(BINARY) .

clean:
	rm -f $(BINARY)

apply-networking:
	@echo "Applying networking rules for Split Routing..."
	@echo "Physical Interface: $(PHY_IFACE) via $(PHY_GW)"
	@echo "Provider Interface: $(PROVIDER_IFACE) :$(PROVIDER_PORT)"
	
	# 1. Policy Based Routing
	# Create a separate routing table with default route via physical gateway
	sudo ip route add default via $(PHY_GW) dev $(PHY_IFACE) table $(TABLE_ID) || echo "Route may already exist"
	
	# Mark outgoing packets from Provider Port
	sudo iptables -t mangle -A OUTPUT -p tcp --sport $(PROVIDER_PORT) -j MARK --set-mark $(MARK_ID)
	
	# Rule to use custom table for marked packets
	sudo ip rule add fwmark $(MARK_ID) lookup $(TABLE_ID) || echo "Rule may already exist"
	
	# Flush cache
	sudo ip route flush cache

	# 2. Firewall Rules
	sudo iptables -A INPUT -p tcp --dport $(PROVIDER_PORT) -j ACCEPT
	sudo iptables -A FORWARD -i $(PROVIDER_IFACE) -o $(PHY_IFACE) -j ACCEPT
	sudo iptables -A FORWARD -i $(PHY_IFACE) -o $(PROVIDER_IFACE) -m state --state RELATED,ESTABLISHED -j ACCEPT
	sudo iptables -t nat -A POSTROUTING -o $(PHY_IFACE) -j MASQUERADE
	@echo "Networking rules applied."

remove-networking:
	@echo "Removing networking rules..."
	
	# 2. Firewall Rules
	sudo iptables -t nat -D POSTROUTING -o $(PHY_IFACE) -j MASQUERADE || echo "Rule not found"
	sudo iptables -D FORWARD -i $(PHY_IFACE) -o $(PROVIDER_IFACE) -m state --state RELATED,ESTABLISHED -j ACCEPT || echo "Rule not found"
	sudo iptables -D FORWARD -i $(PROVIDER_IFACE) -o $(PHY_IFACE) -j ACCEPT || echo "Rule not found"
	sudo iptables -D INPUT -p tcp --dport $(PROVIDER_PORT) -j ACCEPT || echo "Rule not found"

	# 1. Policy Based Routing
	sudo ip rule del fwmark $(MARK_ID) lookup $(TABLE_ID) || echo "Rule not found"
	sudo iptables -t mangle -D OUTPUT -p tcp --sport $(PROVIDER_PORT) -j MARK --set-mark $(MARK_ID) || echo "Rule not found"
	sudo ip route del default via $(PHY_GW) dev $(PHY_IFACE) table $(TABLE_ID) || echo "Route not found"
	
	# Flush cache
	sudo ip route flush cache
	@echo "Networking rules removed."

check-networking:
	@echo "Checking networking configuration..."
	@echo "--- Routing Table $(TABLE_ID) ---"
	@ip route show table $(TABLE_ID) | grep -q "default via $(PHY_GW)" && echo "[OK] Default route exists" || echo "[MISSING] Default route in table $(TABLE_ID)"
	
	@echo "--- Routing Rules ---"
	@ip rule show | grep -q "fwmark $(MARK_ID) lookup $(TABLE_ID)" && echo "[OK] Fwmark rule exists" || echo "[MISSING] Fwmark rule"

	@echo "--- IPTables Mangle ---"
	@sudo iptables -t mangle -C OUTPUT -p tcp --sport $(PROVIDER_PORT) -j MARK --set-mark $(MARK_ID) 2>/dev/null && echo "[OK] Output mark rule exists" || echo "[MISSING] Output mark rule"

	@echo "--- IPTables Filter ---"
	@sudo iptables -C INPUT -p tcp --dport $(PROVIDER_PORT) -j ACCEPT 2>/dev/null && echo "[OK] Input accept rule exists" || echo "[MISSING] Input accept rule"
	@sudo iptables -C FORWARD -i $(PROVIDER_IFACE) -o $(PHY_IFACE) -j ACCEPT 2>/dev/null && echo "[OK] Forward Out rule exists" || echo "[MISSING] Forward Out rule"
	@sudo iptables -C FORWARD -i $(PHY_IFACE) -o $(PROVIDER_IFACE) -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null && echo "[OK] Forward In rule exists" || echo "[MISSING] Forward In rule"

	@echo "--- IPTables NAT ---"
	@sudo iptables -t nat -C POSTROUTING -o $(PHY_IFACE) -j MASQUERADE 2>/dev/null && echo "[OK] Masquerade rule exists" || echo "[MISSING] Masquerade rule"