# Monitoring System Health

This document describes the monitoring and observability features in BlockchainVPN.

## Overview

BlockchainVPN provides logging and observability through the `internal/obs` package.

## Logging System (`internal/obs/logging.go`)

### Log Levels

| Level | Description |
|-------|-------------|
| DEBUG | Detailed information for debugging |
| INFO | General informational messages |
| WARN | Warning messages |
| ERROR | Error messages |

### Log Formats

**Text Format (default):**
```
2026-03-14 12:00:00 INFO Starting tunnel on bcvpn0
```

**JSON Format:**
```json
{"level":"info","time":"2026-03-14T12:00:00Z","msg":"Starting tunnel","component":"tunnel","interface":"bcvpn0"}
```

### Configuration

Configure logging via command-line flags or environment:

```bash
# Text format with info level
bcvpn --log-format text --log-level info

# JSON format with debug level  
bcvpn --log-format json --log-level debug
```

## Metrics

### Tunnel Metrics (`internal/tunnel/metrics.go`)

The tunnel package tracks:

- Connection count
- Bytes sent/received
- Session duration
- Error rates

### Provider Health (`internal/tunnel/provider_health.go`)

Providers report health status:

- Current connections
- Available bandwidth
- Uptime percentage
- Last heartbeat

## Logging Components

### JSON Logger

Structured JSON logging for machine parsing:

```go
type jsonLogWriter struct {
    component string
    out       io.Writer
    minLevel  int
    mu        sync.Mutex
}
```

### Text Logger

Human-readable text logging:

```go
type textLogWriter struct {
    out      io.Writer
    minLevel int
    mu       sync.Mutex
}
```

## Best Practices

1. **Use appropriate log levels**: ERROR for failures, INFO for important events, DEBUG for troubleshooting
2. **Include context**: Add relevant fields (tunnel ID, provider, etc.)
3. **JSON for production**: Easier to parse and analyze
4. **Text for development**: Easier to read during debugging

## Integration

### Log Aggregation

For production deployment, consider:
- Filebeat + Elasticsearch
- Loki
- CloudWatch Logs
- Google Cloud Logging

### Metrics Collection

For metrics:
- Prometheus exporter
- InfluxDB
- CloudWatch Metrics

## Example Usage

```go
import "blockchain-vpn/internal/obs"

// Configure JSON logging for production
obs.ConfigureLogging("json", "info", "bcvpn")

// Or use defaults
obs.ConfigureLogging("text", "debug", "")
```

## Health Checks

Providers can expose health check endpoints:

- `/health` - Basic liveness check
- `/ready` - Readiness for connections
- `/metrics` - Prometheus-style metrics
