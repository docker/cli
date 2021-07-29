// +build !windows

package container

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/docker/cli/internal/test"
	"golang.org/x/sys/unix"
	"gotest.tools/v3/assert"
)

func TestIgnoredSignals(t *testing.T) {
	ignoredSignals := []syscall.Signal{unix.SIGPIPE, unix.SIGCHLD, unix.SIGURG}

	for _, s := range ignoredSignals {
		t.Run(unix.SignalName(s), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var called bool
			client := &fakeClient{containerKillFunc: func(ctx context.Context, container, signal string) error {
				called = true
				return nil
			}}

			cli := test.NewFakeCli(client)
			sigc := make(chan os.Signal)
			defer close(sigc)

			done := make(chan struct{})
			go func() {
				ForwardAllSignals(ctx, cli, t.Name(), sigc)
				close(done)
			}()

			timer := time.NewTimer(30 * time.Second)
			defer timer.Stop()

			select {
			case <-timer.C:
				t.Fatal("timeout waiting to send signal")
			case sigc <- s:
			case <-done:
			}

			// cancel the context so ForwardAllSignals will exit after it has processed the signal we sent.
			// This is how we know the signal was actually processed and are not introducing a flakey test.
			cancel()
			<-done

			assert.Assert(t, !called, "kill was called")
		})
	}
}
