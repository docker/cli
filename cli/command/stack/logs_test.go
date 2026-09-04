package stack

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestStackLogsErrors(t *testing.T) {
	testCases := []struct {
		doc             string
		args            []string
		serviceListFunc func(options client.ServiceListOptions) (client.ServiceListResult, error)
		serviceLogsFunc func(serviceID string, options client.ServiceLogsOptions) (client.ServiceLogsResult, error)
		expectedError   string
	}{
		{
			doc:           "no args",
			args:          []string{},
			expectedError: "requires 1 argument",
		},
		{
			doc:           "too many args",
			args:          []string{"foo", "bar"},
			expectedError: "requires 1 argument",
		},
		{
			doc:           "invalid stack name",
			args:          []string{"  "},
			expectedError: "invalid stack name",
		},
		{
			doc:  "service list error",
			args: []string{"foo"},
			serviceListFunc: func(client.ServiceListOptions) (client.ServiceListResult, error) {
				return client.ServiceListResult{}, errors.New("error getting services")
			},
			expectedError: "error getting services",
		},
		{
			doc:           "empty stack",
			args:          []string{"emptystack"},
			expectedError: "nothing found in stack",
		},
		{
			doc:  "service logs error",
			args: []string{"foo"},
			serviceListFunc: func(client.ServiceListOptions) (client.ServiceListResult, error) {
				return client.ServiceListResult{
					Items: []swarm.Service{serviceFromName("foo_web")},
				}, nil
			},
			serviceLogsFunc: func(string, client.ServiceLogsOptions) (client.ServiceLogsResult, error) {
				return nil, errors.New("error getting logs")
			},
			expectedError: "error getting logs",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			cmd := newLogsCommand(test.NewFakeCli(&fakeClient{
				serviceListFunc: tc.serviceListFunc,
				serviceLogsFunc: tc.serviceLogsFunc,
			}))
			cmd.SetArgs(tc.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

// muxStdout wraps payload in the stdcopy stream format (stdout frame), as
// returned by the service logs endpoint for services without a TTY.
func muxStdout(payload string) io.ReadCloser {
	var buf bytes.Buffer
	hdr := make([]byte, 8)
	hdr[0] = 1 // stdout
	binary.BigEndian.PutUint32(hdr[4:], uint32(len(payload)))
	buf.Write(hdr)
	buf.WriteString(payload)
	return io.NopCloser(&buf)
}

func TestStackLogsPrefixesOutput(t *testing.T) {
	logs := map[string]string{
		"ID-mystack_web": "hello from web\n",
		"ID-mystack_db":  "hello from db\n",
	}

	cli := test.NewFakeCli(&fakeClient{
		services: []string{"mystack_web", "mystack_db"},
		serviceLogsFunc: func(serviceID string, _ client.ServiceLogsOptions) (client.ServiceLogsResult, error) {
			return muxStdout(logs[serviceID]), nil
		},
	})
	cmd := newLogsCommand(cli)
	cmd.SetArgs([]string{"--no-color", "mystack"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	assert.NilError(t, cmd.Execute())

	out := cli.OutBuffer().String()
	assert.Check(t, is.Contains(out, "mystack_web | hello from web\n"))
	assert.Check(t, is.Contains(out, "mystack_db  | hello from db\n"))
}

func TestStackLogsPassesOptions(t *testing.T) {
	var got client.ServiceLogsOptions
	cli := test.NewFakeCli(&fakeClient{
		services: []string{"mystack_web"},
		serviceLogsFunc: func(_ string, options client.ServiceLogsOptions) (client.ServiceLogsResult, error) {
			got = options
			return io.NopCloser(bytes.NewReader(nil)), nil
		},
	})
	cmd := newLogsCommand(cli)
	cmd.SetArgs([]string{"--since", "42m", "--tail", "10", "--timestamps", "mystack"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	assert.NilError(t, cmd.Execute())

	assert.Check(t, got.ShowStdout)
	assert.Check(t, got.ShowStderr)
	assert.Check(t, is.Equal(got.Since, "42m"))
	assert.Check(t, is.Equal(got.Tail, "10"))
	assert.Check(t, got.Timestamps)
	assert.Check(t, !got.Follow)
}

func TestPrefixWriter(t *testing.T) {
	var out bytes.Buffer
	var mu sync.Mutex
	w := newPrefixWriter(&out, []byte("P | "), &mu)

	// partial writes must be buffered until a newline arrives
	_, err := w.Write([]byte("a"))
	assert.NilError(t, err)
	assert.Check(t, is.Equal(out.String(), ""))

	_, err = w.Write([]byte("b\nc"))
	assert.NilError(t, err)
	assert.Check(t, is.Equal(out.String(), "P | ab\n"))

	_, err = w.Write([]byte("\n"))
	assert.NilError(t, err)
	assert.Check(t, is.Equal(out.String(), "P | ab\nP | c\n"))
}

func TestLogPrefix(t *testing.T) {
	assert.Check(t, is.Equal(string(logPrefix("web", 0, 5, true)), "web   | "))
	colored := string(logPrefix("web", 0, 3, false))
	assert.Check(t, is.Contains(colored, "\x1b["))
	assert.Check(t, is.Contains(colored, "web | "))
}
