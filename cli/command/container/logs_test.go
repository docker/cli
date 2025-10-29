package container

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

var logFn = func(expectedOut string) func(string, client.ContainerLogsOptions) (io.ReadCloser, error) {
	return func(container string, opts client.ContainerLogsOptions) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(expectedOut)), nil
	}
}

func TestRunLogs(t *testing.T) {
	inspectFn := func(containerID string) (client.ContainerInspectResult, error) {
		return client.ContainerInspectResult{
			Container: container.InspectResponse{
				Config: &container.Config{Tty: true},
				State:  &container.State{Running: false},
			},
		}, nil
	}

	testcases := []struct {
		doc           string
		options       *logsOptions
		client        *fakeClient
		expectedError string
		expectedOut   string
		expectedErr   string
	}{
		{
			doc:         "successful logs",
			expectedOut: "foo",
			options:     &logsOptions{},
			client:      &fakeClient{logFunc: logFn("foo"), inspectFunc: inspectFn},
		},
		{
			doc:         "clear logs for running container",
			options:     &logsOptions{clear: true},
			client: &fakeClient{
				inspectFunc: func(containerID string) (client.ContainerInspectResult, error) {
					return client.ContainerInspectResult{
						Container: container.InspectResponse{
							Config: &container.Config{Tty: true},
							State:  &container.State{Running: true}, // Running
						},
					}, nil
				},
			},
			expectedError: "cannot clear logs for running containers",
		},
		{
			doc:         "clear logs for stopped container as non-root",
			options:     &logsOptions{clear: true},
			client: &fakeClient{
				inspectFunc: func(containerID string) (client.ContainerInspectResult, error) {
					return client.ContainerInspectResult{
						Container: container.InspectResponse{
							Config: &container.Config{Tty: true},
							State:  &container.State{Running: false}, // Stopped
							LogPath: "/fake/path",                   // Provide a log path
						},
					}, nil
				},
			},
			expectedError: "clearing logs requires root permissions", // Assumes test runs as non-root
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.doc, func(t *testing.T) {
			cli := test.NewFakeCli(testcase.client)

			err := runLogs(context.TODO(), cli, testcase.options)
			if testcase.expectedError != "" {
				assert.ErrorContains(t, err, testcase.expectedError)
			} else if !assert.Check(t, err) {
				return
			}
			assert.Check(t, is.Equal(testcase.expectedOut, cli.OutBuffer().String()))
			assert.Check(t, is.Equal(testcase.expectedErr, cli.ErrBuffer().String()))
		})
	}
}