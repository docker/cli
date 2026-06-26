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

func TestRestart(t *testing.T) {
	for _, tc := range []struct {
		name         string
		args         []string
		restarted    []string
		expectedOpts client.ContainerRestartOptions
		expectedErr  string
	}{
		{
			name:      "without options",
			args:      []string{"container-1", "container-2"},
			restarted: []string{"container-1", "container-2"},
		},
		{
			name:        "with unknown container",
			args:        []string{"container-1", "nosuchcontainer", "container-2"},
			expectedErr: "no such container",
			restarted:   []string{"container-1", "container-2"},
		},
		{
			name:         "with -t",
			args:         []string{"-t", "2", "container-1"},
			expectedOpts: client.ContainerRestartOptions{Timeout: func(to int) *int { return &to }(2)},
			restarted:    []string{"container-1"},
		},
		{
			name:         "with --timeout",
			args:         []string{"--timeout", "2", "container-1"},
			expectedOpts: client.ContainerRestartOptions{Timeout: func(to int) *int { return &to }(2)},
			restarted:    []string{"container-1"},
		},
		{
			name:         "with --time",
			args:         []string{"--time", "2", "container-1"},
			expectedOpts: client.ContainerRestartOptions{Timeout: func(to int) *int { return &to }(2)},
			restarted:    []string{"container-1"},
		},
		{
			name:        "conflicting options",
			args:        []string{"--timeout", "2", "--time", "2", "container-1"},
			expectedErr: "conflicting options: cannot specify both --timeout and --time",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var restarted []string
			mutex := new(sync.Mutex)

			cli := test.NewFakeCli(&fakeClient{
				containerRestartFunc: func(ctx context.Context, containerID string, options client.ContainerRestartOptions) (client.ContainerRestartResult, error) {
					assert.Check(t, is.DeepEqual(options, tc.expectedOpts))
					if containerID == "nosuchcontainer" {
						return client.ContainerRestartResult{}, notFound(errors.New("Error: no such container: " + containerID))
					}

					// TODO(thaJeztah): consider using parallelOperation for restart, similar to "stop" and "remove"
					mutex.Lock()
					restarted = append(restarted, containerID)
					mutex.Unlock()
					return client.ContainerRestartResult{}, nil
				},
				Version: "1.36",
			})
			cmd := newRestartCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if tc.expectedErr != "" {
				assert.Check(t, is.ErrorContains(err, tc.expectedErr))
			} else {
				assert.Check(t, is.Nil(err))
			}
			sort.Strings(restarted)
			assert.Check(t, is.DeepEqual(restarted, tc.restarted))
		})
	}
}
