package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/marcelsecco1/dnsquerypulse/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := cli.Run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if !errors.Is(err, cli.ErrQueryFailures) {
			fmt.Fprintln(os.Stderr, "dnsquerypulse:", err)
		}
		os.Exit(1)
	}
}
