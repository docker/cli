package container

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/moby/moby/client"
	"github.com/moby/sys/signal"
)

func TestForwardSignals(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	called := make(chan struct{})
	apiClient := &fakeClient{containerKillFunc: func(ctx context.Context, container string, options client.ContainerKillOptions) (client.ContainerKillResult, error) {
		close(called)
		return client.ContainerKillResult{}, nil
	}}

	sigc := make(chan os.Signal)
	defer close(sigc)

	go ForwardAllSignals(ctx, apiClient, t.Name(), sigc)

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
