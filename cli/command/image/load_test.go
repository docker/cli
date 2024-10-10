package image

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/image"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestNewLoadCommandErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		isTerminalIn  bool
		expectedError string
		imageLoadFunc func(input io.Reader, options image.LoadOptions) (image.LoadResponse, error)
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
			imageLoadFunc: func(input io.Reader, options image.LoadOptions) (image.LoadResponse, error) {
				return image.LoadResponse{}, fmt.Errorf("something went wrong")
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
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
		imageLoadFunc func(input io.Reader, options image.LoadOptions) (image.LoadResponse, error)
	}{
		{
			name: "simple",
			args: []string{},
			imageLoadFunc: func(input io.Reader, options image.LoadOptions) (image.LoadResponse, error) {
				return image.LoadResponse{Body: io.NopCloser(strings.NewReader("Success"))}, nil
			},
		},
		{
			name: "json",
			args: []string{},
			imageLoadFunc: func(input io.Reader, options image.LoadOptions) (image.LoadResponse, error) {
				json := "{\"ID\": \"1\"}"
				return image.LoadResponse{
					Body: io.NopCloser(strings.NewReader(json)),
					JSON: true,
				}, nil
			},
		},
		{
			name: "input-file",
			args: []string{"--input", "testdata/load-command-success.input.txt"},
			imageLoadFunc: func(input io.Reader, options image.LoadOptions) (image.LoadResponse, error) {
				return image.LoadResponse{Body: io.NopCloser(strings.NewReader("Success"))}, nil
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
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
