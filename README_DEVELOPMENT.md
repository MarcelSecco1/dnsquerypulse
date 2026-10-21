# Development and release

[Back to project overview](README.md) · [Architecture](README_ARCHITECTURE.md)

## Local checks

```sh
gofmt -w ./cmd ./internal
go vet ./...
go test ./...
go test -race ./...
```

Build locally and for both MVP operating systems:

```sh
go build ./cmd/dnsquerypulse
GOOS=linux GOARCH=amd64 go build -o dist/dnsquerypulse-linux-amd64 ./cmd/dnsquerypulse
GOOS=windows GOARCH=amd64 go build -o dist/dnsquerypulse-windows-amd64.exe ./cmd/dnsquerypulse
```

Tests use deterministic query data and a local controlled DNS fixture. They do not query public resolvers. The local integration fixture may be skipped in restricted sandboxes that prohibit loopback sockets; CI runs it on GitHub-hosted runners.

## Benchmark methodology

A benchmark run reports raw query outcomes along with successful-response latency percentiles and failure counts. Record OS, CPU, RAM, Go version, program version, resolver endpoint, domain, query type, timeout, count, and concurrency alongside any published results. Repeat the same conditions and disclose that repeated single-domain queries may hit resolver caches. No benchmark measurements are claimed by this repository.

## CI and release

GitHub Actions checks formatting, `go vet`, tests, race tests on Linux, and builds on Linux and Windows. Release binaries should be built from a tagged commit. Publish checksums alongside binaries and verify them before distribution. The CLI does not update itself.

## Project structure

```text
cmd/dnsquerypulse/       executable
internal/cli/            commands and orchestration
internal/config/         YAML schema and validation
internal/dns/            DNS protocol client and results
internal/output/         terminal, JSON, CSV
internal/stats/          metrics and percentiles
configs/                 example configuration
docs/decisions/          short architecture decisions
tests/integration/       controlled DNS integration coverage
.github/workflows/       CI
```
