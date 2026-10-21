# Architecture and DNS behavior

[Back to project overview](README.md) · [Command guide](README_USAGE.md)

## Design

`dnsquerypulse` is a single-process, CLI-first Go application. Each command reuses the same DNS query result model; output format is chosen only after query execution.

```text
CLI -> check / monitor / benchmark / diagnose
                    |
                    v
         DNS wire client (UDP, TCP retry)
                    |
                    v
        typed result -> table / JSON / CSV
                    |
                    v
       latency and availability summary
```

The DNS client uses Go's standard `net` package and a small wire-format encoder/decoder. YAML configuration uses `gopkg.in/yaml.v3`. The rest of the application uses the Go standard library. This keeps the built binary self-contained and avoids adding a dependency just to wrap the DNS wire format.

Key packages:

- `cmd/dnsquerypulse`: executable entry point and Ctrl+C signal context.
- `internal/cli`: command parsing, defaults, validation, and use-case orchestration.
- `internal/dns`: DNS message encoding, UDP/TCP transport, RCODE classification, and answer formatting.
- `internal/stats`: success rate and latency aggregates.
- `internal/config`: strict YAML loading and validation.
- `internal/output`: terminal tables, JSON, CSV, and ASCII header.

## Result model

Each result contains domain, query type, resolver, transport, wall-clock start timestamp, elapsed duration, operation status, DNS RCODE, answer strings, and optional error kind/message. Elapsed time uses Go's monotonic clock component where available; the timestamp is for correlation.

`status` is independent from `rcode`:

- `success`: DNS response with `NOERROR`, including an empty answer set.
- `rcode`: response received with a non-`NOERROR` DNS response code, including `NXDOMAIN` and `SERVFAIL`.
- `timeout`: no response before the deadline.
- `error`: network, invalid query, or parsing failure.
- `cancelled`: parent context ended during a query.

UDP responses marked truncated are retried over TCP. The supported query types are A, AAAA, CNAME, NS, PTR, MX, TXT, SRV, CAA, ANY, and numeric `TYPE####`. The answer renderer formats common records and prints unknown record data as RFC-style hexadecimal text.

## Concurrency and cancellation

- `check` issues sequential queries for each resolver.
- `monitor` uses a ticker and runs rounds sequentially; a slow round cannot accumulate overlapping work.
- `benchmark` uses a fixed worker pool. User input cannot create more than 512 workers or one million requests.
- Query contexts carry per-request deadlines. Ctrl+C cancels the command context and stops new work.

## Limitations

- Latency measures the client-to-resolver round trip. It includes host networking, the path, cache effects, and the resolver response; it does not isolate resolver CPU or internal processing time.
- Multiple resolvers may legitimately return different records due to CDN routing, geography, split-horizon DNS, or load balancing. Different answers are displayed, not automatically treated as a failure.
- Benchmarks use repeated queries for one supplied domain. Cache effects can dominate results; the program does not claim benchmark numbers are universally representative.
- DNS-over-HTTPS, DNS-over-TLS, persistent history, external alerts, daemon mode, and remote telemetry are out of scope.
- Results are not sent anywhere except to the resolver endpoints selected by the user. Names queried may be visible to those resolvers.
