# Install dnsquerypulse

[Back to project overview](README.md) · [Command guide](README_USAGE.md)

## Requirements

- Windows 10/11 or Linux, x86-64 (other Go-supported architectures can be built from source).
- Go 1.22.2 or newer to build from source.
- UDP port 53 reachability to the DNS resolver you choose. TCP port 53 is also used if a UDP answer is truncated.

The released executable is standalone and has no runtime dependencies. No administrator/root privileges are needed.

Download the prebuilt binaries from the [v0.1.0 GitHub release](https://github.com/MarcelSecco1/dnsquerypulse/releases/tag/v0.1.0), or build from source below.

## Build from source

```sh
git clone https://github.com/marcelsecco1/dnsquerypulse.git
cd dnsquerypulse
go build -o dnsquerypulse ./cmd/dnsquerypulse
./dnsquerypulse --help
```

PowerShell:

```powershell
git clone https://github.com/marcelsecco1/dnsquerypulse.git
Set-Location dnsquerypulse
go build -o dnsquerypulse.exe ./cmd/dnsquerypulse
.\dnsquerypulse.exe --help
```

## Build release binaries

From the repository root:

```sh
mkdir -p dist
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/dnsquerypulse-linux-amd64 ./cmd/dnsquerypulse
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/dnsquerypulse-windows-amd64.exe ./cmd/dnsquerypulse
```

On Windows PowerShell, build the native binary with:

```powershell
New-Item -ItemType Directory -Force dist | Out-Null
go build -trimpath -ldflags="-s -w" -o dist/dnsquerypulse-windows-amd64.exe ./cmd/dnsquerypulse
```

Copy the executable to a directory in your `PATH`, or invoke it from its current directory. The program never changes OS resolver settings.

## Verify the installation

```sh
dnsquerypulse version
dnsquerypulse check example.com --resolver 1.1.1.1
```

A failed DNS request exits with code `1` and writes its concise error to stderr. A successful request exits with code `0`.
