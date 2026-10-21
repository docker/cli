package jsonstream

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/docker/cli/cli/streams"
	"gotest.tools/v3/assert"
)

func TestDisplay(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	client, server := io.Pipe()
	t.Cleanup(func() {
		assert.NilError(t, server.Close())
	})

	go func() {
		for range 100 {
			select {
			case <-ctx.Done():
				assert.NilError(t, server.Close(), "failed to close jsonmessage server")
				return
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	streamCtx, cancelStream := context.WithCancel(context.Background())
	t.Cleanup(cancelStream)

	done := make(chan error)
	go func() {
		done <- DisplayStream(streamCtx, client, streams.NewOut(io.Discard))
	}()

	cancelStream()

	select {
	case <-time.After(time.Second * 3):
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
	}
}
