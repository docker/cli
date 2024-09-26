package container

import (
	"context"
	"io"
	"sort"
	"sync"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/errdefs"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestRestart(t *testing.T) {
	for _, tc := range []struct {
		name         string
		args         []string
		restarted    []string
		expectedOpts container.StopOptions
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
			expectedOpts: container.StopOptions{Timeout: func(to int) *int { return &to }(2)},
			restarted:    []string{"container-1"},
		},
		{
			name:         "with --time",
			args:         []string{"--time", "2", "container-1"},
			expectedOpts: container.StopOptions{Timeout: func(to int) *int { return &to }(2)},
			restarted:    []string{"container-1"},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var restarted []string
			mutex := new(sync.Mutex)

			cli := test.NewFakeCli(&fakeClient{
				containerRestartFunc: func(ctx context.Context, containerID string, options container.StopOptions) error {
					assert.Check(t, is.DeepEqual(options, tc.expectedOpts))
					if containerID == "nosuchcontainer" {
						return errdefs.NotFound(errors.New("Error: no such container: " + containerID))
					}

					// TODO(thaJeztah): consider using parallelOperation for restart, similar to "stop" and "remove"
					mutex.Lock()
					restarted = append(restarted, containerID)
					mutex.Unlock()
					return nil
				},
				Version: "1.36",
			})
			cmd := NewRestartCommand(cli)
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
