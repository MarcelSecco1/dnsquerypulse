package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/marcelsecco1/dnsquerypulse/internal/config"
	"github.com/marcelsecco1/dnsquerypulse/internal/dns"
	"github.com/marcelsecco1/dnsquerypulse/internal/output"
	"github.com/marcelsecco1/dnsquerypulse/internal/stats"
)

const version = "0.1.0"

var ErrQueryFailures = errors.New("DNS query failures")

func Run(ctx context.Context, args []string, out, errOut io.Writer) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		Help(out)
		return nil
	}
	if args[0] == "--version" || args[0] == "version" {
		fmt.Fprintf(out, "dnsquerypulse %s\n", version)
		return nil
	}
	cmd := args[0]
	args = args[1:]
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			commandHelp(out, cmd)
			return nil
		}
	}
	if cmd == "config" {
		return configCommand(args, out)
	}
	if cmd != "check" && cmd != "monitor" && cmd != "benchmark" && cmd != "diagnose" {
		return fmt.Errorf("unknown command %q; run dnsquerypulse --help", cmd)
	}
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(errOut)
	cfgPath := fs.String("config", "", "configuration YAML file")
	resolver := fs.String("resolver", "", "comma-separated resolver IPs or host:port")
	timeout := fs.Duration("timeout", 3*time.Second, "per-query timeout")
	format := fs.String("format", "", "table, json, or csv")
	typ := fs.String("type", "A", "DNS record type")
	interval := fs.Duration("interval", 5*time.Second, "time between monitor rounds")
	duration := fs.Duration("duration", time.Minute, "total monitoring duration")
	count := fs.Int("count", 100, "number of benchmark queries")
	concurrency := fs.Int("concurrency", 10, "maximum simultaneous DNS queries")
	if err := fs.Parse(intermix(args)); err != nil {
		return err
	}
	pos := fs.Args()
	if cmd == "check" || cmd == "monitor" || cmd == "benchmark" || cmd == "diagnose" {
		if len(pos) != 1 {
			return fmt.Errorf("usage: dnsquerypulse %s DOMAIN [flags]", cmd)
		}
	}
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return err
	}
	servers := chooseResolvers(*resolver, cfg.Resolvers)
	if *timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if fmtxOverride := *format; fmtxOverride != "" && fmtxOverride != "table" && fmtxOverride != "json" && fmtxOverride != "csv" {
		return fmt.Errorf("format must be table, json, or csv")
	}
	if len(servers) == 0 {
		return fmt.Errorf("at least one resolver is required")
	}
	to := *timeout
	timeoutChanged := false
	fs.Visit(func(v *flag.Flag) {
		if v.Name == "timeout" {
			timeoutChanged = true
		}
	})
	if !timeoutChanged {
		to = cfg.TimeoutDuration()
	}
	fmtx := *format
	if fmtx == "" {
		fmtx = cfg.Format
	}
	if fmtx == "table" {
		output.Banner(out)
	}
	switch cmd {
	case "check":
		results := make([]dns.Result, 0, len(servers))
		for _, r := range servers {
			results = append(results, dns.Query(ctx, pos[0], *typ, r, to))
		}
		if err = output.Results(out, results, fmtx); err != nil {
			return err
		}
		return anyFailed(results)
	case "monitor":
		if *interval <= 0 || *duration <= 0 {
			return fmt.Errorf("interval and duration must be positive")
		}
		mctx, cancel := context.WithTimeout(ctx, *duration)
		defer cancel()
		ticker := time.NewTicker(*interval)
		defer ticker.Stop()
		var results []dns.Result
		for {
			for _, r := range servers {
				if mctx.Err() != nil {
					break
				}
				results = append(results, dns.Query(mctx, pos[0], *typ, r, to))
			}
			if mctx.Err() != nil {
				break
			}
			select {
			case <-mctx.Done():
			case <-ticker.C:
			}
			if mctx.Err() != nil {
				break
			}
		}
		if err = output.Report(out, results, stats.Summarize(results), fmtx); err != nil {
			return err
		}
		return anyFailed(results)
	case "benchmark":
		if *count < 1 || *count > 1000000 {
			return fmt.Errorf("count must be between 1 and 1000000")
		}
		if *concurrency < 1 || *concurrency > 512 {
			return fmt.Errorf("concurrency must be between 1 and 512")
		}
		results := make([]dns.Result, *count)
		jobs := make(chan int)
		workers := *concurrency
		if workers > *count {
			workers = *count
		}
		var wg sync.WaitGroup
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range jobs {
					results[i] = dns.Query(ctx, pos[0], *typ, servers[i%len(servers)], to)
				}
			}()
		}
		for i := 0; i < *count; i++ {
			select {
			case jobs <- i:
			case <-ctx.Done():
				close(jobs)
				wg.Wait()
				return ctx.Err()
			}
		}
		close(jobs)
		wg.Wait()
		if err = output.Report(out, results, stats.Summarize(results), fmtx); err != nil {
			return err
		}
		return anyFailed(results)
	case "diagnose":
		var results []dns.Result
		for _, r := range servers {
			for _, q := range []string{"A", "AAAA"} {
				results = append(results, dns.Query(ctx, pos[0], q, r, to))
			}
		}
		if err = output.Report(out, results, stats.Summarize(results), fmtx); err != nil {
			return err
		}
		return anyFailed(results)
	}
	return nil
}
func commandHelp(w io.Writer, command string) {
	switch command {
	case "check":
		fmt.Fprintln(w, "Usage: dnsquerypulse check DOMAIN [--resolver IP[,IP]] [--type TYPE] [--timeout 3s] [--format table|json|csv] [--config PATH]")
	case "monitor":
		fmt.Fprintln(w, "Usage: dnsquerypulse monitor DOMAIN [--interval 5s] [--duration 1m] [--resolver IP[,IP]] [--format table|json|csv]")
	case "benchmark":
		fmt.Fprintln(w, "Usage: dnsquerypulse benchmark DOMAIN [--count 100] [--concurrency 10] [--resolver IP[,IP]] [--format table|json|csv]")
	case "diagnose":
		fmt.Fprintln(w, "Usage: dnsquerypulse diagnose DOMAIN [--resolver IP[,IP]] [--timeout 3s] [--format table|json|csv]")
	case "config":
		fmt.Fprintln(w, "Usage: dnsquerypulse config <init|validate> [--config PATH]")
	default:
		Help(w)
	}
}

func configCommand(args []string, out io.Writer) error {
	if len(args) == 0 || args[0] == "--help" {
		fmt.Fprintln(out, "Usage: dnsquerypulse config <init|validate> [--config PATH]")
		return nil
	}
	switch args[0] {
	case "init":
		path := "dnsquerypulse.yaml"
		if len(args) > 1 {
			if len(args) == 3 && args[1] == "--config" {
				path = args[2]
			} else {
				return fmt.Errorf("usage: dnsquerypulse config init [--config PATH]")
			}
		}
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("refusing to overwrite %s", path)
		}
		if err := os.WriteFile(path, []byte(config.Example), 0600); err != nil {
			return err
		}
		fmt.Fprintf(out, "Created %s\n", path)
		return nil
	case "validate":
		path := ""
		if len(args) == 3 && args[1] == "--config" {
			path = args[2]
		}
		if path == "" {
			return fmt.Errorf("usage: dnsquerypulse config validate --config PATH")
		}
		if _, err := config.Load(path); err != nil {
			return err
		}
		fmt.Fprintln(out, "Configuration is valid.")
		return nil
	default:
		return fmt.Errorf("unknown config command %q", args[0])
	}
}
func intermix(args []string) []string {
	valueFlags := map[string]bool{"--config": true, "--resolver": true, "--timeout": true, "--format": true, "--type": true, "--interval": true, "--duration": true, "--count": true, "--concurrency": true}
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			name := a
			if k := strings.IndexByte(a, '='); k >= 0 {
				name = a[:k]
			}
			if valueFlags[name] && !strings.Contains(a, "=") && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		} else {
			positional = append(positional, a)
		}
	}
	return append(flags, positional...)
}

func chooseResolvers(flag string, defaults []string) []string {
	if strings.TrimSpace(flag) == "" {
		return defaults
	}
	var out []string
	for _, r := range strings.Split(flag, ",") {
		if x := strings.TrimSpace(r); x != "" {
			out = append(out, x)
		}
	}
	return out
}
func anyFailed(rs []dns.Result) error {
	for _, r := range rs {
		if r.Status != "success" {
			return ErrQueryFailures
		}
	}
	return nil
}
func Help(w io.Writer) {
	fmt.Fprintln(w, `dnsquerypulse - DNS diagnostics, monitoring and benchmarking

Usage:
  dnsquerypulse <command> [DOMAIN] [flags]

Commands:
  check       Query a domain on one or more DNS resolvers
  monitor     Monitor DNS availability for a duration
  benchmark   Run a bounded concurrent DNS benchmark
  diagnose    Run basic A and AAAA checks
  config      Create or validate YAML configuration
  version     Print version

Common flags:
  --resolver  Comma-separated resolver IPs or host:port
  --timeout   Per-query timeout (default 3s)
  --type      DNS record type (default A)
  --format    table, json, or csv (default table)
  --config    YAML configuration file
  --help      Show help

Examples:
  dnsquerypulse check example.com --resolver 1.1.1.1,8.8.8.8
  dnsquerypulse check example.com --type AAAA --format json
  dnsquerypulse monitor example.com --interval 5s --duration 10m
  dnsquerypulse benchmark example.com --count 1000 --concurrency 10
  dnsquerypulse config init
  dnsquerypulse config validate --config dnsquerypulse.yaml`)
}
