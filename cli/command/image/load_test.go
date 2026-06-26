package image

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestNewLoadCommandErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		isTerminalIn  bool
		expectedError string
		imageLoadFunc func(input io.Reader, options ...client.ImageLoadOption) (client.ImageLoadResult, error)
	}{
		{
			name:          "wrong-args",
			args:          []string{"arg"},
			expectedError: "accepts no arguments",
		},
		{
			name:          "input-to-terminal",
			args:          []string{},
			isTerminalIn:  true,
			expectedError: "requested load from stdin, but stdin is empty",
		},
		{
			name:          "pull-error",
			args:          []string{},
			expectedError: "something went wrong",
			imageLoadFunc: func(input io.Reader, options ...client.ImageLoadOption) (client.ImageLoadResult, error) {
				return nil, errors.New("something went wrong")
			},
		},
		{
			name:          "invalid platform",
			args:          []string{"--platform", "<invalid>"},
			expectedError: `invalid platform`,
			imageLoadFunc: func(input io.Reader, options ...client.ImageLoadOption) (client.ImageLoadResult, error) {
				return io.NopCloser(strings.NewReader("")), nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{imageLoadFunc: tc.imageLoadFunc})
			cli.In().SetIsTerminal(tc.isTerminalIn)
			cmd := newLoadCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestNewLoadCommandInvalidInput(t *testing.T) {
	expectedError := "open *"
	cmd := newLoadCommand(test.NewFakeCli(&fakeClient{}))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--input", "*"})
	err := cmd.Execute()
	assert.ErrorContains(t, err, expectedError)
}

func mockImageLoadResult(content string) client.ImageLoadResult {
	return io.NopCloser(strings.NewReader(content))
}

func TestNewLoadCommandSuccess(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		imageLoadFunc func(input io.Reader, options ...client.ImageLoadOption) (client.ImageLoadResult, error)
	}{
		{
			name: "simple",
			args: []string{},
			imageLoadFunc: func(input io.Reader, options ...client.ImageLoadOption) (client.ImageLoadResult, error) {
				// FIXME(thaJeztah): how to mock this?
				// return client.ImageLoadResult{
				// 	Body: io.NopCloser(strings.NewReader(`{"ID":"simple","Status":"success"}`)),
				// }, nil
				return mockImageLoadResult(`{"ID":"simple","Status":"success"}`), nil
			},
		},
		{
			name: "input-file",
			args: []string{"--input", "testdata/load-command-success.input.txt"},
			imageLoadFunc: func(input io.Reader, options ...client.ImageLoadOption) (client.ImageLoadResult, error) {
				// FIXME(thaJeztah): how to mock this?
				// return client.ImageLoadResult{Body: io.NopCloser(strings.NewReader(`{"ID":"input-file","Status":"success"}`))}, nil
				return mockImageLoadResult(`{"ID":"input-file","Status":"success"}`), nil
			},
		},
		{
			name: "with-single-platform",
			args: []string{"--platform", "linux/amd64"},
			imageLoadFunc: func(input io.Reader, options ...client.ImageLoadOption) (client.ImageLoadResult, error) {
				// FIXME(thaJeztah): need to find appropriate way to test the result of "ImageHistoryWithPlatform" being applied
				assert.Check(t, len(options) > 0) // can be 1 or two depending on whether a terminal is attached :/
				// assert.Check(t, is.Contains(options, client.ImageHistoryWithPlatform(ocispec.Platform{OS: "linux", Architecture: "amd64"})))
				// FIXME(thaJeztah): how to mock this?
				// return client.ImageLoadResult{Body: io.NopCloser(strings.NewReader(`{"ID":"single-platform","Status":"success"}`))}, nil
				return mockImageLoadResult(`{"ID":"single-platform","Status":"success"}`), nil
			},
		},
		{
			name: "with-comma-separated-platforms",
			args: []string{"--platform", "linux/amd64,linux/arm64/v8,linux/riscv64"},
			imageLoadFunc: func(input io.Reader, options ...client.ImageLoadOption) (client.ImageLoadResult, error) {
				assert.Check(t, len(options) > 0) // can be 1 or two depending on whether a terminal is attached :/
				// FIXME(thaJeztah): how to mock this?
				// return client.ImageLoadResult{Body: io.NopCloser(strings.NewReader(`{"ID":"with-comma-separated-platforms","Status":"success"}`))}, nil
				return mockImageLoadResult(`{"ID":"with-comma-separated-platforms","Status":"success"}`), nil
			},
		},
		{
			name: "with-multiple-platform-options",
			args: []string{"--platform", "linux/amd64", "--platform", "linux/arm64/v8", "--platform", "linux/riscv64"},
			imageLoadFunc: func(input io.Reader, options ...client.ImageLoadOption) (client.ImageLoadResult, error) {
				assert.Check(t, len(options) > 0) // can be 1 or two depending on whether a terminal is attached :/
				// FIXME(thaJeztah): how to mock this?
				// return client.ImageLoadResult{Body: io.NopCloser(strings.NewReader(`{"ID":"with-multiple-platform-options","Status":"success"}`))}, nil
				return mockImageLoadResult(`{"ID":"with-multiple-platform-options","Status":"success"}`), nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{imageLoadFunc: tc.imageLoadFunc})
			cmd := newLoadCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			assert.NilError(t, err)
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("load-command-success.%s.golden", tc.name))
		})
	}
}
