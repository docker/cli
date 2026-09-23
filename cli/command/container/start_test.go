package container

import (
	"context"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestStartValidateFlags(t *testing.T) {
	for _, tc := range []struct {
		name        string
		args        []string
		expectedErr string
	}{
		{
			name:        "with invalid --detach-keys",
			args:        []string{"--detach-keys", "shift-a", "myimage"},
			expectedErr: "invalid detach keys (shift-a):",
		},
		{
			name:        "with --start-healthy-timeout and --attach",
			args:        []string{"--start-healthy-timeout", "30s", "--attach", "mycontainer"},
			expectedErr: "--start-healthy-timeout cannot be combined with --attach or --interactive",
		},
		{
			name:        "with --start-healthy-timeout and --interactive",
			args:        []string{"--start-healthy-timeout", "30s", "--interactive", "mycontainer"},
			expectedErr: "--start-healthy-timeout cannot be combined with --attach or --interactive",
		},
		{
			name:        "with negative --start-healthy-timeout",
			args:        []string{"--start-healthy-timeout", "-30s", "mycontainer"},
			expectedErr: "--start-healthy-timeout cannot be negative",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newStartCommand(test.NewFakeCli(&fakeClient{}))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if tc.expectedErr != "" {
				assert.Check(t, is.ErrorContains(err, tc.expectedErr))
			} else {
				assert.Check(t, is.Nil(err))
			}
		})
	}
}

func TestStartWithStartHealthyTimeout(t *testing.T) {
	for _, tc := range []struct {
		name        string
		state       container.State
		expectedErr string
	}{
		{
			name:  "healthy",
			state: container.State{Running: true, Health: &container.Health{Status: container.Healthy}},
		},
		{
			name:        "not healthy in time",
			state:       container.State{Running: true, Health: &container.Health{Status: container.Starting}},
			expectedErr: "container mycontainer did not become healthy within 100ms",
		},
		{
			name:        "no healthcheck configured",
			state:       container.State{Running: true},
			expectedErr: "container mycontainer does not have a healthcheck configured",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fakeCLI := test.NewFakeCli(&fakeClient{
				inspectFunc: func(containerID string) (client.ContainerInspectResult, error) {
					return client.ContainerInspectResult{
						Container: container.InspectResponse{ID: containerID, State: &tc.state},
					}, nil
				},
				containerStopFunc: func(context.Context, string, client.ContainerStopOptions) (client.ContainerStopResult, error) {
					t.Error("container should not be stopped")
					return client.ContainerStopResult{}, nil
				},
			})
			cmd := newStartCommand(fakeCLI)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{"--start-healthy-timeout", "100ms", "mycontainer"})

			err := cmd.Execute()
			if tc.expectedErr == "" {
				assert.NilError(t, err)
			} else {
				assert.Check(t, is.ErrorContains(err, tc.expectedErr))
			}
			// The container is reported as started regardless of whether it
			// becomes healthy.
			assert.Check(t, is.Equal(fakeCLI.OutBuffer().String(), "mycontainer\n"))
		})
	}
}
