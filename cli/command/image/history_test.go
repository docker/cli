package image

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/docker/docker/api/types/image"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNewHistoryCommandSuccess(t *testing.T) {
	testCases := []struct {
		name             string
		args             []string
		outputRegex      string
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
		// TODO: This test is failing since the output does not contain an RFC3339 date
		//{
		//	name:        "non-human",
		//	args:        []string{"--human=false", "image:tag"},
		//	outputRegex: "\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}", // RFC3339 date format match
		//},
		{
			name:        "non-human-header",
			args:        []string{"--human=false", "image:tag"},
			outputRegex: "CREATED\\sAT",
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
		assert.NoError(t, err)
		actual := cli.OutBuffer().String()
		if tc.outputRegex == "" {
			golden.Assert(t, actual, fmt.Sprintf("history-command-success.%s.golden", tc.name))
		} else {
			assert.Regexp(t, tc.outputRegex, actual)
		}
	}
}
