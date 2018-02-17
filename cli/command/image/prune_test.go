package image

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewPruneCommandErrors(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		expectedError   string
		imagesPruneFunc func(pruneFilter filters.Args) (types.ImagesPruneReport, error)
	}{
		{
			name:          "wrong-args",
			args:          []string{"something"},
			expectedError: "accepts no arguments.",
		},
		{
			name:          "prune-error",
			args:          []string{"--force"},
			expectedError: "something went wrong",
			imagesPruneFunc: func(pruneFilter filters.Args) (types.ImagesPruneReport, error) {
				return types.ImagesPruneReport{}, errors.Errorf("something went wrong")
			},
		},
	}
	for _, tc := range testCases {
		cmd := NewPruneCommand(test.NewFakeCli(&fakeClient{
			imagesPruneFunc: tc.imagesPruneFunc,
		}))
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNewPruneCommandSuccess(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		imagesPruneFunc func(pruneFilter filters.Args) (types.ImagesPruneReport, error)
	}{
		{
			name: "all",
			args: []string{"--all"},
			imagesPruneFunc: func(pruneFilter filters.Args) (types.ImagesPruneReport, error) {
				assert.Equal(t, "false", pruneFilter.Get("dangling")[0])
				return types.ImagesPruneReport{}, nil
			},
		},
		{
			name: "force-deleted",
			args: []string{"--force"},
			imagesPruneFunc: func(pruneFilter filters.Args) (types.ImagesPruneReport, error) {
				assert.Equal(t, "true", pruneFilter.Get("dangling")[0])
				return types.ImagesPruneReport{
					ImagesDeleted:  []types.ImageDeleteResponseItem{{Deleted: "image1"}},
					SpaceReclaimed: 1,
				}, nil
			},
		},
		{
			name: "force-untagged",
			args: []string{"--force"},
			imagesPruneFunc: func(pruneFilter filters.Args) (types.ImagesPruneReport, error) {
				assert.Equal(t, "true", pruneFilter.Get("dangling")[0])
				return types.ImagesPruneReport{
					ImagesDeleted:  []types.ImageDeleteResponseItem{{Untagged: "image1"}},
					SpaceReclaimed: 2,
				}, nil
			},
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{imagesPruneFunc: tc.imagesPruneFunc})
		cmd := NewPruneCommand(cli)
		cmd.SetOutput(ioutil.Discard)
		cmd.SetArgs(tc.args)
		err := cmd.Execute()
		assert.NoError(t, err)
		golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("prune-command-success.%s.golden", tc.name))
	}
}
