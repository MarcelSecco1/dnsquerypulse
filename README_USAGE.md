# dnsquerypulse command guide

[Back to project overview](README.md) · [Installation](README_INSTALL.md)

Commands print a blue-gradient DNSQUERYPULSE banner and aligned ASCII tables in terminal mode. Set `NO_COLOR=1` to disable color. Detailed errors remain in JSON/CSV; the terminal table keeps attention on the answer and status columns. Use `--format json` for machine-readable output without decorative text on stdout. Errors go to stderr.

## `check`

Query a domain on one or more resolvers. The default record type is A, timeout is 3 seconds per resolver, and default resolvers are `1.1.1.1` and `8.8.8.8`.

```sh
dnsquerypulse check example.com
dnsquerypulse check example.com --resolver 1.1.1.1,8.8.8.8
dnsquerypulse check example.com --type AAAA --format json
dnsquerypulse check example.com --type MX --timeout 5s --format csv
```

`status` describes the query operation; `rcode` describes the DNS server's response. For example, `NXDOMAIN` is a valid DNS response for a name that does not exist. An empty answer with `NOERROR` can also be valid.

## `monitor`

Run an immediate round, then start each following round after the configured interval. A slow round never overlaps with another round. Ctrl+C stops active DNS queries and returns the collected results.

```sh
dnsquerypulse monitor api.example.com --interval 5s --duration 10m
dnsquerypulse monitor example.com --resolver 1.1.1.1 --interval 1s --duration 30s --format json
```

The report includes total queries, successful queries, timeouts, other failures, availability, latency percentiles, and consecutive failures at the end of the run. Percentiles include successful responses only; failures remain visible in counts and result rows. In JSON mode the output is a single object with `results` and `summary` fields.

## `benchmark`

Run a fixed number of queries using a bounded worker pool. Concurrency is limited to 512 workers; query count is limited to one million to prevent accidental unbounded work.

```sh
dnsquerypulse benchmark example.com --count 1000 --concurrency 10
dnsquerypulse benchmark example.com --count 500 --concurrency 20 --resolver 1.1.1.1,8.8.8.8 --format json
```

When multiple resolvers are configured, requests are distributed round-robin. Keep resolver list and workload parameters identical when comparing runs. Repeated queries may be served from cache.

## `diagnose`

Run A and AAAA checks for each selected resolver:

```sh
dnsquerypulse diagnose api.example.com --resolver 1.1.1.1,8.8.8.8
```

This is a compact report, not an exhaustive network health assessment.

## `config`

Create a starter file in the current directory or at a chosen path:

```sh
dnsquerypulse config init
dnsquerypulse config init --config ./dnsquerypulse.yaml
```

Validate it before sharing or using it:

```sh
dnsquerypulse config validate --config ./dnsquerypulse.yaml
```

Configuration example:

```yaml
resolvers:
  - 1.1.1.1
  - 8.8.8.8
timeout: 3s
format: table
```

The accepted keys are `resolvers`, `timeout`, and `format`. Up to 64 resolvers may be configured. Unknown YAML keys are errors. Command-line values for `--resolver`, `--timeout`, and `--format` override config values.

## Common flags

| Flag | Meaning |
| --- | --- |
| `--resolver` | Comma-separated IPs or `host:port` endpoints |
| `--timeout` | Per-query timeout, such as `1500ms` or `5s` |
| `--type` | DNS record type (A, AAAA, CNAME, NS, PTR, MX, TXT, SRV, CAA, or `TYPE####`) |
| `--format` | `table`, `json`, or `csv` |
| `--config` | YAML configuration path |
| `--help` | Show help |

## Exit status

- `0`: all queries returned `NOERROR`.
- `1`: command usage/configuration error, network error, timeout, cancellation, or non-`NOERROR` RCODE.

JSON and CSV output are written to stdout. Command errors are written to stderr so pipelines can separate data from diagnostics.
