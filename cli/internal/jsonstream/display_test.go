package jsonstream

import (
	"context"
	"encoding/json"
	"fmt"
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
		enc := json.NewEncoder(server)
		for i := 0; i < 100; i++ {
			select {
			case <-ctx.Done():
				assert.NilError(t, server.Close(), "failed to close jsonmessage server")
				return
			default:
				err := enc.Encode(JSONMessage{
					Status:   "Downloading",
					ID:       fmt.Sprintf("id-%d", i),
					TimeNano: time.Now().UnixNano(),
					Time:     time.Now().Unix(),
					Progress: &JSONProgress{
						Current: int64(i),
						Total:   100,
						Start:   0,
					},
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
