package jsonstream

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/client/pkg/progress"
	"github.com/moby/moby/client/pkg/streamformatter"
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
		id := test.RandomID()[:12] // short-ID
		progressOutput := streamformatter.NewJSONProgressOutput(server, true)
		for i := 0; i < 100; i++ {
			select {
			case <-ctx.Done():
				assert.NilError(t, server.Close(), "failed to close jsonmessage server")
				return
			default:
				err := progressOutput.WriteProgress(progress.Progress{
					ID:      id,
					Message: "Downloading",
					Current: int64(i),
					Total:   100,
				})
				if err != nil {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	streamCtx, cancelStream := context.WithCancel(context.Background())
	t.Cleanup(cancelStream)

	done := make(chan error)
	go func() {
		out := streams.NewOut(io.Discard)
		done <- Display(streamCtx, client, out)
	}()

	cancelStream()

	select {
	case <-time.After(time.Second * 3):
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
	}
}
