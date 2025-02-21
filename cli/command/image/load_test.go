package image // import "docker.com/cli/v28/cli/command/image"

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/v28/internal/test"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestNewLoadCommandErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		isTerminalIn  bool
		expectedError string
		imageLoadFunc func(input io.Reader, options ...client.ImageLoadOption) (image.LoadResponse, error)
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
			imageLoadFunc: func(io.Reader, ...client.ImageLoadOption) (image.LoadResponse, error) {
				return image.LoadResponse{}, errors.New("something went wrong")
			},
		},
		{
			name:          "invalid platform",
			args:          []string{"--platform", "<invalid>"},
			expectedError: `invalid platform`,
			imageLoadFunc: func(io.Reader, ...client.ImageLoadOption) (image.LoadResponse, error) {
				return image.LoadResponse{}, nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{imageLoadFunc: tc.imageLoadFunc})
			cli.In().SetIsTerminal(tc.isTerminalIn)
			cmd := NewLoadCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestNewLoadCommandInvalidInput(t *testing.T) {
	expectedError := "open *"
	cmd := NewLoadCommand(test.NewFakeCli(&fakeClient{}))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--input", "*"})
	err := cmd.Execute()
	assert.ErrorContains(t, err, expectedError)
}

func TestNewLoadCommandSuccess(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		imageLoadFunc func(input io.Reader, options ...client.ImageLoadOption) (image.LoadResponse, error)
	}{
		{
			name: "simple",
			args: []string{},
			imageLoadFunc: func(io.Reader, ...client.ImageLoadOption) (image.LoadResponse, error) {
				return image.LoadResponse{Body: io.NopCloser(strings.NewReader("Success"))}, nil
			},
		},
		{
			name: "json",
			args: []string{},
			imageLoadFunc: func(io.Reader, ...client.ImageLoadOption) (image.LoadResponse, error) {
				return image.LoadResponse{
					Body: io.NopCloser(strings.NewReader(`{"ID": "1"}`)),
					JSON: true,
				}, nil
			},
		},
		{
			name: "input-file",
			args: []string{"--input", "testdata/load-command-success.input.txt"},
			imageLoadFunc: func(input io.Reader, options ...client.ImageLoadOption) (image.LoadResponse, error) {
				return image.LoadResponse{Body: io.NopCloser(strings.NewReader("Success"))}, nil
			},
		},
		{
			name: "with platform",
			args: []string{"--platform", "linux/amd64"},
			imageLoadFunc: func(input io.Reader, options ...client.ImageLoadOption) (image.LoadResponse, error) {
				// FIXME(thaJeztah): need to find appropriate way to test the result of "ImageHistoryWithPlatform" being applied
				assert.Check(t, len(options) > 0) // can be 1 or two depending on whether a terminal is attached :/
				// assert.Check(t, is.Contains(options, client.ImageHistoryWithPlatform(ocispec.Platform{OS: "linux", Architecture: "amd64"})))
				return image.LoadResponse{Body: io.NopCloser(strings.NewReader("Success"))}, nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{imageLoadFunc: tc.imageLoadFunc})
			cmd := NewLoadCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			assert.NilError(t, err)
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("load-command-success.%s.golden", tc.name))
		})
	}
}
