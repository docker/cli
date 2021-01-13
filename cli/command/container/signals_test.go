package container

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/pkg/signal"
)

func TestForwardSignals(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	called := make(chan struct{})
	client := &fakeClient{containerKillFunc: func(ctx context.Context, container, signal string) error {
		close(called)
		return nil
	}}

	cli := test.NewFakeCli(client)
	sigc := make(chan os.Signal)
	defer close(sigc)

	go ForwardAllSignals(ctx, cli, t.Name(), sigc)

	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
		t.Fatal("timeout waiting to send signal")
	case sigc <- signal.SignalMap["TERM"]:
	}
	if !timer.Stop() {
		<-timer.C
	}
	timer.Reset(30 * time.Second)

	select {
	case <-called:
	case <-timer.C:
		t.Fatal("timeout waiting for signal to be processed")
	}

}
