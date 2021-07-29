package container

import (
	"context"
	"fmt"
	"os"
	gosignal "os/signal"

	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/pkg/signal"
	"github.com/sirupsen/logrus"
)

// ForwardAllSignals forwards signals to the container
//
// The channel you pass in must already be setup to receive any signals you want to forward.
func ForwardAllSignals(ctx context.Context, cli command.Cli, cid string, sigc <-chan os.Signal) {
	var s os.Signal
	for {
		select {
		case s = <-sigc:
		case <-ctx.Done():
			return
		}

		if s == signal.SIGCHLD || s == signal.SIGPIPE {
			continue
		}

		// In go1.14+, the go runtime issues SIGURG as an interrupt to support pre-emptable system calls on Linux.
		// Since we can't forward that along we'll check that here.
		if isRuntimeSig(s) {
			continue
		}
		var sig string
		for sigStr, sigN := range signal.SignalMap {
			if sigN == s {
				sig = sigStr
				break
			}
		}
		if sig == "" {
			fmt.Fprintf(cli.Err(), "Unsupported signal: %v. Discarding.\n", s)
			continue
		}

		if err := cli.Client().ContainerKill(ctx, cid, sig); err != nil {
			logrus.Debugf("Error sending signal: %s", err)
		}
	}
}

func notfiyAllSignals() chan os.Signal {
	sigc := make(chan os.Signal, 128)
	gosignal.Notify(sigc)
	return sigc
}
