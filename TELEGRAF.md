# IRTT Telegraf Integration

IRTT now has native support for outputting metrics in InfluxDB line protocol format, making it directly compatible with Telegraf without requiring wrapper scripts.

## Usage

### Basic Usage

Output Telegraf metrics to stdout:

```bash
irtt client -Q --format=telegraf -o - hostname
```

Output to a file:

```bash
irtt client --format=telegraf -o metrics.txt hostname
```

### With Tags

Add custom tags to all metrics:

```bash
irtt client -Q --format=telegraf --telegraf-tags=site=dc1,env=prod -o - hostname
```

### Telegraf Configuration

Use with Telegraf's `exec` input plugin:

```toml
[[inputs.exec]]
  ## Command to run
  commands = [
    "irtt client -Q --format=telegraf --telegraf-tags=target=server1 -d 5s -o - server1.example.com"
  ]

  ## Timeout for command execution
  timeout = "10s"

  ## Data format (InfluxDB line protocol)
  data_format = "influx"

  ## Run tests every minute
  interval = "1m"
```

## Command-Line Flags

- `--format=telegraf` - Output in InfluxDB line protocol format (default: `json`)
- `--telegraf-tags=key=value,key=value` - Comma-separated tags to add to all metrics
- `-o file` - Output file (use `-` for stdout)
- `-Q` - Really quiet mode (suppresses normal output, shows only metrics)

## Output Format

IRTT outputs metrics in InfluxDB line protocol format:

```
measurement,tag=value field=value timestamp
```

### Example Output

```
irtt,target=127.0.0.1:2112,status=success,site=dc1 success=1i 1761231869148123000
irtt,target=127.0.0.1:2112,status=success,site=dc1 rtt_n=30i,rtt_min_ns=195667i,rtt_max_ns=5853791i,rtt_mean_ns=846841i,rtt_median_ns=588000i,rtt_stddev_ns=1022580i 1761231869148123000
irtt,target=127.0.0.1:2112,status=success,site=dc1 packets_sent=30i,packets_received=30i,packet_loss_percent=0.0000 1761231869148123000
```

## Metrics Reference

### Tags

All metrics include these tags:
- `target` - The IRTT server address (automatically added)
- `status` - Always `success` for successful tests
- Custom tags from `--telegraf-tags`

### Fields

#### Status
- `success` - 1i (always present for successful tests)

#### RTT Statistics
- `rtt_n` - Number of RTT samples
- `rtt_min_ns` - Minimum RTT in nanoseconds
- `rtt_max_ns` - Maximum RTT in nanoseconds
- `rtt_mean_ns` - Mean RTT in nanoseconds
- `rtt_median_ns` - Median RTT in nanoseconds
- `rtt_stddev_ns` - RTT standard deviation in nanoseconds

#### Send/Receive Delay
- `send_delay_*` - One-way send delay metrics (same fields as RTT)
- `receive_delay_*` - One-way receive delay metrics (same fields as RTT)

#### IPDV (Jitter)
- `ipdv_rtt_*` - Round-trip jitter metrics (same fields as RTT)
- `ipdv_send_*` - Send-side jitter metrics (same fields as RTT)
- `ipdv_receive_*` - Receive-side jitter metrics (same fields as RTT)

#### Server Processing
- `server_processing_*` - Server processing time metrics (same fields as RTT, no median)

#### Packet Loss
- `packets_sent` - Total packets sent
- `packets_received` - Total packets received
- `packet_loss_percent` - Overall packet loss percentage
- `upstream_loss_percent` - Packet loss to server
- `downstream_loss_percent` - Packet loss from server
- `duplicates` - Number of duplicate packets
- `duplicate_percent` - Duplicate packet percentage
- `late_packets` - Number of out-of-order packets
- `late_packets_percent` - Out-of-order packet percentage

#### Bitrate
- `send_rate_bps` - Send bitrate in bits per second
- `receive_rate_bps` - Receive bitrate in bits per second
- `bytes_sent` - Total bytes sent
- `bytes_received` - Total bytes received

#### Timer Error (optional)
- `timer_error_*` - Timer error metrics (same fields as RTT, no median)
- `timer_err_percent` - Timer error percentage
- `timer_misses` - Number of timer misses
- `timer_miss_percent` - Timer miss percentage

## Customizing Metrics

To control which metrics are included, modify `DefaultTelegrafOptions()` in `telegraf.go` and rebuild:

```go
func DefaultTelegrafOptions() *TelegrafOptions {
    return &TelegrafOptions{
        Tags:                    make(map[string]string),
        IncludeRTT:              true,   // Enable/disable RTT stats
        IncludeSendDelay:        true,   // Enable/disable send delay
        IncludeReceiveDelay:     true,   // Enable/disable receive delay
        IncludeIPDV:             true,   // Enable/disable jitter
        IncludePacketLoss:       true,   // Enable/disable packet loss
        IncludeBitrate:          true,   // Enable/disable bitrate
        IncludeServerProcessing: true,   // Enable/disable server processing
        IncludeTimerError:       false,  // Timer error disabled by default
    }
}
```

Then rebuild: `go build -o irtt ./cmd/irtt`

## Examples

### Monitor Multiple Targets

```toml
[[inputs.exec]]
  commands = [
    "irtt client -Q --format=telegraf --telegraf-tags=target=web1 -d 5s -o - web1.example.com",
    "irtt client -Q --format=telegraf --telegraf-tags=target=web2 -d 5s -o - web2.example.com",
    "irtt client -Q --format=telegraf --telegraf-tags=target=db1 -d 5s -o - db1.example.com"
  ]
  timeout = "10s"
  data_format = "influx"
  interval = "30s"
```

### Test with Specific Parameters

```bash
# IPv6 test
irtt client -Q -6 --format=telegraf --telegraf-tags=ipv=6 -o - server.example.com

# High-frequency test (10ms interval)
irtt client -Q --format=telegraf -d 5s -i 10ms -o - server.example.com

# Large packet test
irtt client -Q --format=telegraf -l 1472 -o - server.example.com

# With DSCP marking
irtt client -Q --format=telegraf --dscp=0xb8 -o - server.example.com
```

### Grafana Queries

```sql
-- RTT Mean over time
SELECT mean("rtt_mean_ns")/1000000 AS "RTT (ms)"
FROM "irtt"
WHERE $timeFilter
GROUP BY time($__interval), "target"

-- Packet Loss by target
SELECT mean("packet_loss_percent")
FROM "irtt"
WHERE $timeFilter
GROUP BY time($__interval), "target"

-- Jitter (IPDV)
SELECT mean("ipdv_rtt_mean_ns")/1000000 AS "Jitter (ms)"
FROM "irtt"
WHERE $timeFilter
GROUP BY time($__interval), "target"
```

## Advantages Over Wrapper Scripts

✅ **No wrapper needed** - IRTT outputs Telegraf format natively
✅ **One less process** - More efficient, less overhead
✅ **Simpler deployment** - Just the IRTT binary
✅ **Faster** - No subprocess spawning or JSON parsing
✅ **More reliable** - No wrapper script to maintain or debug
✅ **Easier to debug** - Direct output from IRTT

## Comparison with JSON Output

| Feature | JSON | Telegraf |
|---------|------|----------|
| **Format** | JSON | InfluxDB line protocol |
| **Telegraf Integration** | Requires parsing | Native support |
| **Per-packet data** | Yes | No (summary only) |
| **File size** | Large (can be gzipped) | Small |
| **Human readable** | Moderate | Limited |
| **Use case** | Analysis, archival | Real-time monitoring |

Use JSON format (`--format=json`) for detailed analysis and archival.
Use Telegraf format (`--format=telegraf`) for real-time monitoring with Telegraf.

## Error Handling

When IRTT cannot connect to the server or encounters an error, it outputs a failure metric instead of exiting with an error:

```bash
# Connection refused
irtt,target=127.0.0.1 success=0i,error="read udp4 127.0.0.1:54696->127.0.0.1:2112: read: connection refused" 1761234651571679000

# Timeout (no reply from server)
irtt,target=192.0.2.1 success=0i,error="[OpenTimeout] no reply from server" 1761234599245432000
```

This allows you to:
- **Monitor connectivity**: Alert when `success=0i`
- **Track uptime**: Calculate `mean(success) * 100` for success rate percentage
- **Debug issues**: Error message is included in the `error` field

The command exits with code 0 even on connection failures, so Telegraf continues collecting metrics from other targets.

## Notes

- All duration values are in nanoseconds for consistency
- Timestamps use nanosecond precision
- The `success` field is `1i` for successful tests, `0i` for failures
- When `success=0i`, an `error` field contains the error message
- The `status` tag indicates `success` or `failed` (for filtering in queries)
- IPDV metrics only include packets with valid previous RTT measurements
- Server processing time only available when server timestamps are enabled
- Use `success=1` in WHERE clauses to filter successful tests
- Use `mean(success) * 100` to calculate success rate percentage
