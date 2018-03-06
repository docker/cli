package container

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

var logFn = func(expectedOut string) func(string, types.ContainerLogsOptions) (io.ReadCloser, error) {
	return func(container string, opts types.ContainerLogsOptions) (io.ReadCloser, error) {
		return ioutil.NopCloser(strings.NewReader(expectedOut)), nil
	}
}

func TestRunLogs(t *testing.T) {
	inspectFn := func(containerID string) (types.ContainerJSON, error) {
		return types.ContainerJSON{
			Config:            &container.Config{Tty: true},
			ContainerJSONBase: &types.ContainerJSONBase{State: &types.ContainerState{Running: false}},
		}, nil
	}

	var testcases = []struct {
		doc           string
		options       *logsOptions
		client        fakeClient
		expectedError string
		expectedOut   string
		expectedErr   string
	}{
		{
			doc:         "successful logs",
			expectedOut: "foo",
			options:     &logsOptions{},
			client:      fakeClient{logFunc: logFn("foo"), inspectFunc: inspectFn},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.doc, func(t *testing.T) {
			cli := test.NewFakeCli(&testcase.client)

			err := runLogs(cli, testcase.options)
			if testcase.expectedError != "" {
				assert.ErrorContains(t, err, testcase.expectedError)
			} else {
				if !assert.Check(t, err) {
					return
				}
			}
			assert.Check(t, is.Equal(testcase.expectedOut, cli.OutBuffer().String()))
			assert.Check(t, is.Equal(testcase.expectedErr, cli.ErrBuffer().String()))
		})
	}
}
