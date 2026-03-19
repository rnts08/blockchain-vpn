# Operations Manual and Commands

This document expands and explains all commands available for the BlockchainVPN CLI.

## 1. Basics

   * Version:
      ```bash
         bcvpn version
      ```
   * Help:
      ```bash
         bcvpn help
      ```
   * About:
      ```bash
         bcvpn about
         bcvpn -a
      ```
   * Detailed Help about command if available:
      ```bash
         bcvpn help config
         bcvpn help config rpc
         bcvpn help config country
      ```
## 2. Basic commands

    * Interactive Setup (recommended for new users)
       ```bash
       bcvpn setup
       ```
    * Status
       ```bash
       bcvpn status [--json]
      ```
   * Events
      ```bash
         bcvpn events [--limit 100] [--json] 
      ```   
   * Diagnostics & Doctoring 
      ```bash
         bcvpn diagnostics
         bcvpn doctor
      ```
   * Connection history with metrics and incoming/outgoing payments
      ```bash
         bcvpn history [--json] [--table]
         bcvpn history --from <datetime> --to <datatime>
      ```

## 3. Configuration

   * Generate Basic Configuration with defaults:
      ```bash
         bcvpn generate-config
      ```
   * View/vaildate configuration
      ```bash
         bcvpn config 
         bcvpn config validate
         bcvpn config get <field>
         bcvpn config set <field> <value>
      ```
   * RPC connection details:
      ```bash
         bcvpn config set rpc host <host:port>
         bcvpn config set rpc user <username>
         bcvpn config set rpc pass <password>
         bcvpn config set rpc network <mainnet|testnet>
         bcvpn config set rpc enable_tls true
         bcvpn config set rpc token_symbol <token>
      ```
   * Logging settings:
      ```bash
         bcvpn config set logging format [text|json] 
         bcvpn config set level [info|debug|warning|error]
         bvcpn config set logfile <path>
      ```
   * Security settings:
      ```
         bcvpn config set security key_storage_mode [file|keychain|libsecret]
         bcvpn config set security key_storage_service <name>
         bcvpn config set security tls_min_version <1.0|1.1|1.3>
         bcvpn config set security tls_profile <default|modern>
         bcvpn config set security tls_custom_cipher_suites <null>
         bcvpn config set security revokation_cache_file <path>
     ```
   * Metrics settings:
      ```bash
         bcvpn config set metrics_listen_addr <localhost:port>
         bcvpn config set metrics token <secrettoken>
      ```         
   * Other settings:
      ```bash
         bcvpn config set dns_servers [1.1.1.1, 4.4.4.4]
      ```

### VPN Client Configuration
      ```bash
         bcvpn config set client sending-address <address>
         bcvpn config set client interface_name <name>
         bcvpn config set client tun_ip <ip>
         bcvpn config set client tun_subnet <subnet>
         bcvpn config set client tunnel_dns_traffic [true|false]
         bcvpn config set client enable_kill_switch [true|false]
         bcvpn config set client strict_verification [true|false]
         bcvpn config set client verify_throughput_after_connect [true|false]
         bcvpn config set client max_parallel_tunnels 1
         bcvpn config set client enable_websocket_fallback [true|false]
         bcvpn config set client spending_limit_enabled [true|false]
         bcvpn config set client spending_limit_sats <sats>
         bcvpn config set client spending_warning_percent <percent>
         bcvpn config set client auto_disconnect_on_limit [true|false]
         bcvpn config set client max_session_spending_sats <sats>
         bcvpn config set client auto_recharge_enabled [true|false]
         bcvpn config set client auto_recharge_threshold <sats>
         bcvpn config set client auto_recharge_amount <sats>
         bcvpn config set client auto_recharge_min_balance <sats>
      ```
### VPN Provider Configuration

   ```bash
         bcvpn config set provider receive-address <address>
         bcvpn config set provider interface_name <name>
         bcvpn config set provider tun_ip <ip>
         bcvpn config set provider tun_subnet <subnet>
         bcvpn config set provider enable_nat [true|false]
         bcvpn config set provider nat_outbound_interface <name>
         bcvpn config set provider enable_egress_nat [true|false]
         bcvpn config set provider nat_traversal_method [auto|upnp|natpmp|none]
         bcvpn config set provider isolation_mode [none|sandbox]
         
         bcvpn config set provider private_key_file <path>
         bcvpn config set provider country <country|auto>
         bcvpn config set provider max_consumers <count>
         bcvpn config set provider announce_ip <ip|auto>
         bcvpn config set provider listen_port <port>
         bcvpn config set provider auto_rotate_port [true|false]

         bcvpn config set provider bandwidth limit <kpbps>
         bcvpn config set provider bandwidth auto_test [true|false]
         bcvpn config set provider pricing_method [session|time|data]
         bcvpn config set provider price_sats_per_session <sats>
         bcvpn config set provider billing_time_unit [minute|hour]
         bcvpn config set provider billing_data_unit [MB|GB]

         bcvpn config set provider health_check_enabled [true|false]
         bcvpn config set provider health_check_interval <duration>
         bcvpn config set provider heartbeat_interval <duration>
         bcvpn config set provider cert_lifetime_hours <hours>
         bcvpn config set provider cert_rotate_before_hours <hours>
         bcvpn config set provider allowlist_file <path>
         bcvpn config set provider denylist_file <path>
         bcvpn config set provider bandwidth_monitor_interval <duration>
         bcvpn config set provider throughput_probe_port <port>
         bcvpn config set provider announcement_fee_target_blocks <blocks>
         bcvpn config set provider announcement_fee_mode <>
         bcvpn config set provider announcement_interval <duration>
         bcvpn config set provider max_session_duration_secs <seconds>
         bcvpn config set provider bandwidth_throughput_test_port <port>
         bcvpn config set provider websocket_fallback_port <port>
         bcvpn config set provider payment_monitor_interval <duration>
         bcvpn config set provider shutdown_timeout <duration>
   ```

## VPN Client commands
   * Scan for providers:
      ```
         bcvpn scan
         bcvpn scan --country <> \
                    --max-price <> \
                    --min-bandwidth-kbps <> \
                    --max-latency-ms <> \
                    --min-available-slots <> \
                    --pricing-method [session|time|data] \
                    --min-score <> \
                    --sort [country|price|bw|latency|capacity|score] 
                    --limit N (max 100)
                    --rescan
      ```

      This will scan the blockchain for available providers, with the filters you've applied up to a maximum of 100. If no filter is given you will receive the top 100 providers by score. The scan will be cached so that you don't have to re-run it every time, if you want to re-scan and get 
      new results before the cache times out use the rescan option.

   * Generate send address (via RPC)
      ```
         bcvpn generate-send-address
      ```

   * Generate TLS compatible keypair for the tunnel
      ```
         bcvpn generate-tls-keypair
      ```

   * Connect to provider
      ```
         bvcpn connect <provider-signature>
      ```

   * Disconnect from provider
      ```
         bvcpn disconnect
      ```

   * Favorite provider management
      ```
         bcvpn favorite [add|remove] <provider-signature> [comment]
      ```

   * Rate provider
      ```
         bcvpn rate <provider-signature> <rating> [comment]
     ```

## VPN Provider commands
   * Generate recieve address (via RPC)
      ```
         bcvpn generate-receive-address
      ```

   * Generate TLS compatible keypair for the tunnel
      ```
         bcvpn generate-tls-keypair
      ```

   * Manage provider-key
      ```
         bcvpn generate-provider-key
         bcvpn rotate-provider-key
      ```

   * Manage provider service operations
      ```
         bcvpn start-provider
         bcvpn restart-provider
         bcvpn stop-provider
      ```

   * Broadcast announcement
      ```
         bcvpn [re]broadcast
      ```

## Incident Response

When compromise is suspected:

1. Stop provider immediately.
2. Rotate provider key.
3. Revoke suspicious client keys.
4. Review payment history and auth logs.
5. Re-announce service with fresh identity.

## Upgrade Strategy

1. Capture pre-upgrade diagnostics:
   - `./bcvpn status --json`
2. Backup config directory (`config.json`, key material, history).
3. Upgrade binaries.
4. Run validation + status checks.
5. Re-enable provider and verify latency/traffic path.
