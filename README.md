# dnsquerypulse

**Cross-platform DNS diagnostics and monitoring CLI written in Go.**

`dnsquerypulse` helps developers and network engineers inspect DNS answers, compare resolvers, monitor availability, and run bounded latency benchmarks from Linux and Windows terminals. It is a local command-line tool: there is no account, server, telemetry, or automatic change to system DNS settings.

```text
+------------------- dnsquerypulse --------------------+
| check      compare answers and response time          |
| monitor    observe availability over time             |
| benchmark  measure bounded concurrent queries         |
| diagnose   inspect A and AAAA responses                |
+-------------------------------------------------------+
```

## Download

Download the prebuilt Linux and Windows binaries from the [dnsquerypulse v0.1.0 release](https://github.com/MarcelSecco1/dnsquerypulse/releases/tag/v0.1.0).

## Documentation

- [Install and build](README_INSTALL.md)
- [Command guide and examples](README_USAGE.md)
- [Architecture, DNS behavior, and limitations](README_ARCHITECTURE.md)
- [Development, tests, and releases](README_DEVELOPMENT.md)
- [Example YAML configuration](configs/config.example.yaml)
- [Architecture decision records](docs/decisions/README.md)

## Features

- Query A, AAAA, CNAME, NS, PTR, MX, TXT, SRV, CAA, and numeric `TYPE####` records.
- Compare multiple user-selected resolvers from one command.
- Report DNS RCODE separately from network status, timeout, and parse errors.
- Monitor with bounded, non-overlapping rounds and graceful Ctrl+C cancellation.
- Benchmark with a fixed worker pool and p50, p95, p99, mean, availability, and failure counts.
- Print an ASCII terminal interface or emit JSON and CSV for scripts.
- Validate YAML configuration and reject unknown keys.
- Build for Windows and Linux without a server or privileged access.

## Quick start

```sh
git clone https://github.com/marcelsecco1/dnsquerypulse.git
cd dnsquerypulse
go run ./cmd/dnsquerypulse --help
go run ./cmd/dnsquerypulse check example.com --resolver 1.1.1.1,8.8.8.8
```

See [installation](README_INSTALL.md) for prebuilt binaries and [usage](README_USAGE.md) for all commands.

## Project status

MVP implementation, version `0.1.0`. Public DNS availability is outside the tool's control. Latency is the observed client-to-resolver round trip and includes local network, path, and resolver cache effects; it does not measure a resolver's internal processing time. See the [limitations](README_ARCHITECTURE.md#limitations) before interpreting results.

## License

MIT. See [LICENSE](LICENSE).
