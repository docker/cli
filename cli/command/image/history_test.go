package image

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/image"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/gotestyourself/gotestyourself/skip"
	"github.com/pkg/errors"
)

func TestNewHistoryCommandErrors(t *testing.T) {
	testCases := []struct {
		name             string
		args             []string
		expectedError    string
		imageHistoryFunc func(img string) ([]image.HistoryResponseItem, error)
	}{
		{
			name:          "wrong-args",
			args:          []string{},
			expectedError: "requires exactly 1 argument.",
		},
		{
			name:          "client-error",
			args:          []string{"image:tag"},
			expectedError: "something went wrong",
			imageHistoryFunc: func(img string) ([]image.HistoryResponseItem, error) {
				return []image.HistoryResponseItem{{}}, errors.Errorf("something went wrong")
			},
		},
	}
	for _, tc := range testCases {
		cmd := NewHistoryCommand(test.NewFakeCli(&fakeClient{imageHistoryFunc: tc.imageHistoryFunc}))
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func notUTCTimezone() bool {
	now := time.Now()
	return now != now.UTC()
}

func TestNewHistoryCommandSuccess(t *testing.T) {
	skip.If(t, notUTCTimezone, "expected output requires UTC timezone")
	testCases := []struct {
		name             string
		args             []string
		imageHistoryFunc func(img string) ([]image.HistoryResponseItem, error)
	}{
		{
			name: "simple",
			args: []string{"image:tag"},
			imageHistoryFunc: func(img string) ([]image.HistoryResponseItem, error) {
				return []image.HistoryResponseItem{{
					ID:      "1234567890123456789",
					Created: time.Now().Unix(),
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
			imageHistoryFunc: func(img string) ([]image.HistoryResponseItem, error) {
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
			imageHistoryFunc: func(img string) ([]image.HistoryResponseItem, error) {
				return []image.HistoryResponseItem{{
					ID:      "1234567890123456789",
					Created: time.Now().Unix(),
				}}, nil
			},
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{imageHistoryFunc: tc.imageHistoryFunc})
		cmd := NewHistoryCommand(cli)
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		err := cmd.Execute()
		assert.NilError(t, err)
		actual := cli.OutBuffer().String()
		golden.Assert(t, actual, fmt.Sprintf("history-command-success.%s.golden", tc.name))
	}
}
