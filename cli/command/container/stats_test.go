package container

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
)

// TestRunStatsRestart cancels each old request even when the daemon keeps its
// stats stream open across container exits, as it does for stopped containers.
func TestRunStatsRestart(t *testing.T) {
	const id = "test-container"
	actions := make(chan events.Action)
	opened := make(chan struct{}, 10)
	closed := make(chan struct{}, 10)
	ctx, cancel := context.WithCancel(context.Background())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		switch strings.TrimPrefix(r.URL.Path, "/v1.54") {
		case "/containers/json":
			_ = enc.Encode([]container.Summary{{ID: id}})
		case "/events":
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			for {
				select {
				case action := <-actions:
					if err := enc.Encode(events.Message{Type: events.ContainerEventType, Action: action, Actor: events.Actor{ID: id}}); err != nil {
						return
					}
					w.(http.Flusher).Flush()
				case <-r.Context().Done():
					return
				}
			}
		case "/containers/" + id + "/stats":
			_ = enc.Encode(container.StatsResponse{ID: id, OSType: "linux"})
			w.(http.Flusher).Flush()
			opened <- struct{}{}
			<-r.Context().Done()
			closed <- struct{}{}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	// Cancel before closing the server, including when an assertion fails.
	t.Cleanup(cancel)
	apiClient, err := client.New(client.WithHost(server.URL), client.WithAPIVersion("1.54"))
	assert.NilError(t, err)
	t.Cleanup(func() { assert.NilError(t, apiClient.Close()) })
	done := make(chan error, 1)
	go func() { done <- RunStats(ctx, test.NewFakeCli(apiClient), &StatsOptions{}) }()

	wait := func(ch <-chan struct{}, description string) {
		t.Helper()
		select {
		case <-ch:
		case <-time.After(5 * time.Second):
			t.Fatal(description)
		}
	}
	send := func(action events.Action) {
		t.Helper()
		select {
		case actions <- action:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out sending container event")
		}
	}
	wait(opened, "initial stats request was not opened")
	for range 3 {
		send(events.ActionDie)
		wait(closed, "stats request survived the die event")
		send(events.ActionStart)
		wait(opened, "stats request was not reopened after restart")
	}
	send(events.ActionDestroy)
	wait(closed, "stats request survived the destroy event")
	cancel()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("stats command did not stop after cancellation")
	}
}
