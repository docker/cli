package container

import (
	"context"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

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
			client: &fakeClient{
				logFunc: func(container string, opts client.ContainerLogsOptions) (client.ContainerLogsResult, error) {
					// FIXME(thaJeztah): how to mock this?
					return mockContainerLogsResult("foo"), nil
				},
				inspectFunc: inspectFn,
			},
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
