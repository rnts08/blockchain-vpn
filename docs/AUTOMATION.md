# Automation JSON Examples

## 1. Version JSON

```bash
./bcvpn version --json
```

Example:

```json
{
  "version": "0.1.0",
  "commit": "dev",
  "built": "unknown"
}
```

## 2. Status JSON

```bash
./bcvpn status --json
```

Example fields used by automation:

- `networking.privileges_ok`
- `security.key_storage_supported`
- `security.tls_min_version`
- `provider.metrics_listen_addr`
- `warnings[]`

## 3. Metrics JSON

```bash
curl http://127.0.0.1:9090/metrics.json
```

With auth token:

```bash
curl -H "X-BCVPN-Metrics-Token: <token>" http://127.0.0.1:9090/metrics.json
```

Example fields:

- `provider_running`
- `client_connected`
- `active_sessions`
- `total_up_bytes`
- `total_down_bytes`
- `error_count`
- `health.tun_ok`
- `health.listener_ok`
