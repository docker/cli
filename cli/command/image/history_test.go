package image

import (
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/image"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestNewHistoryCommandErrors(t *testing.T) {
	testCases := []struct {
		name             string
		args             []string
		expectedError    string
		imageHistoryFunc func(img string, options image.HistoryOptions) ([]image.HistoryResponseItem, error)
	}{
		{
			name:          "wrong-args",
			args:          []string{},
			expectedError: "requires 1 argument",
		},
		{
			name:          "client-error",
			args:          []string{"image:tag"},
			expectedError: "something went wrong",
			imageHistoryFunc: func(img string, options image.HistoryOptions) ([]image.HistoryResponseItem, error) {
				return []image.HistoryResponseItem{{}}, errors.Errorf("something went wrong")
			},
		},
		{
			name:          "invalid platform",
			args:          []string{"--platform", "<invalid>", "arg1"},
			expectedError: `invalid platform`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewHistoryCommand(test.NewFakeCli(&fakeClient{imageHistoryFunc: tc.imageHistoryFunc}))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestNewHistoryCommandSuccess(t *testing.T) {
	testCases := []struct {
		name             string
		args             []string
		imageHistoryFunc func(img string, options image.HistoryOptions) ([]image.HistoryResponseItem, error)
	}{
		{
			name: "simple",
			args: []string{"image:tag"},
			imageHistoryFunc: func(img string, options image.HistoryOptions) ([]image.HistoryResponseItem, error) {
				return []image.HistoryResponseItem{{
					ID:      "1234567890123456789",
					Created: time.Now().Unix(),
					Comment: "none",
				}}, nil
			},
		},
		{
			name: "quiet",
			args: []string{"--quiet", "image:tag"},
		},
		{
			name: "non-human",
			args: []string{"--human=false", "image:tag"},
			imageHistoryFunc: func(img string, options image.HistoryOptions) ([]image.HistoryResponseItem, error) {
				return []image.HistoryResponseItem{{
					ID:        "abcdef",
					Created:   time.Date(2017, 1, 1, 12, 0, 3, 0, time.UTC).Unix(),
					CreatedBy: "rose",
					Comment:   "new history item!",
				}}, nil
			},
		},
		{
			name: "quiet-no-trunc",
			args: []string{"--quiet", "--no-trunc", "image:tag"},
			imageHistoryFunc: func(img string, options image.HistoryOptions) ([]image.HistoryResponseItem, error) {
				return []image.HistoryResponseItem{{
					ID:      "1234567890123456789",
					Created: time.Now().Unix(),
				}}, nil
			},
		},
		{
			name: "platform",
			args: []string{"--platform", "linux/amd64", "image:tag"},
			imageHistoryFunc: func(img string, options image.HistoryOptions) ([]image.HistoryResponseItem, error) {
				assert.Check(t, is.DeepEqual(ocispec.Platform{OS: "linux", Architecture: "amd64"}, *options.Platform))
				return []image.HistoryResponseItem{{
					ID:      "1234567890123456789",
					Created: time.Now().Unix(),
				}}, nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set to UTC timezone as timestamps in output are
			// printed in the current timezone
			t.Setenv("TZ", "UTC")
			cli := test.NewFakeCli(&fakeClient{imageHistoryFunc: tc.imageHistoryFunc})
			cmd := NewHistoryCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			assert.NilError(t, err)
			actual := cli.OutBuffer().String()
			golden.Assert(t, actual, fmt.Sprintf("history-command-success.%s.golden", tc.name))
		})
	}
}
