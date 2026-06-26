package container

import (
	"context"
	"errors"
	"io"
	"sort"
	"sync"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestStop(t *testing.T) {
	for _, tc := range []struct {
		name         string
		args         []string
		stopped      []string
		expectedOpts client.ContainerStopOptions
		expectedErr  string
	}{
		{
			name:    "without options",
			args:    []string{"container-1", "container-2"},
			stopped: []string{"container-1", "container-2"},
		},
		{
			name:        "with unknown container",
			args:        []string{"container-1", "nosuchcontainer", "container-2"},
			expectedErr: "no such container",
			stopped:     []string{"container-1", "container-2"},
		},
		{
			name:         "with -t",
			args:         []string{"-t", "2", "container-1"},
			expectedOpts: client.ContainerStopOptions{Timeout: func(to int) *int { return &to }(2)},
			stopped:      []string{"container-1"},
		},
		{
			name:         "with --timeout",
			args:         []string{"--timeout", "2", "container-1"},
			expectedOpts: client.ContainerStopOptions{Timeout: func(to int) *int { return &to }(2)},
			stopped:      []string{"container-1"},
		},
		{
			name:         "with --time",
			args:         []string{"--time", "2", "container-1"},
			expectedOpts: client.ContainerStopOptions{Timeout: func(to int) *int { return &to }(2)},
			stopped:      []string{"container-1"},
		},
		{
			name:        "conflicting options",
			args:        []string{"--timeout", "2", "--time", "2", "container-1"},
			expectedErr: "conflicting options: cannot specify both --timeout and --time",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stopped []string
			mutex := new(sync.Mutex)

			cli := test.NewFakeCli(&fakeClient{
				containerStopFunc: func(ctx context.Context, containerID string, options client.ContainerStopOptions) (client.ContainerStopResult, error) {
					assert.Check(t, is.DeepEqual(options, tc.expectedOpts))
					if containerID == "nosuchcontainer" {
						return client.ContainerStopResult{}, notFound(errors.New("Error: no such container: " + containerID))
					}

					// containerStopFunc is called in parallel for each container
					// so append must be synchronized.
					mutex.Lock()
					stopped = append(stopped, containerID)
					mutex.Unlock()
					return client.ContainerStopResult{}, nil
				},
				Version: "1.36",
			})
			cmd := newStopCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if tc.expectedErr != "" {
				assert.Check(t, is.ErrorContains(err, tc.expectedErr))
			} else {
				assert.Check(t, is.Nil(err))
			}
			sort.Strings(stopped)
			assert.Check(t, is.DeepEqual(stopped, tc.stopped))
		})
	}
}
