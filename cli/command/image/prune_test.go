package image

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestNewPruneCommandErrors(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		expectedError   string
		imagesPruneFunc func(pruneFilter filters.Args) (image.PruneReport, error)
	}{
		{
			name:          "wrong-args",
			args:          []string{"something"},
			expectedError: "accepts no arguments",
		},
		{
			name:          "prune-error",
			args:          []string{"--force"},
			expectedError: "something went wrong",
			imagesPruneFunc: func(pruneFilter filters.Args) (image.PruneReport, error) {
				return image.PruneReport{}, errors.Errorf("something went wrong")
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewPruneCommand(test.NewFakeCli(&fakeClient{
				imagesPruneFunc: tc.imagesPruneFunc,
			}))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestNewPruneCommandSuccess(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		imagesPruneFunc func(pruneFilter filters.Args) (image.PruneReport, error)
	}{
		{
			name: "all",
			args: []string{"--all"},
			imagesPruneFunc: func(pruneFilter filters.Args) (image.PruneReport, error) {
				assert.Check(t, is.Equal("false", pruneFilter.Get("dangling")[0]))
				return image.PruneReport{}, nil
			},
		},
		{
			name: "force-deleted",
			args: []string{"--force"},
			imagesPruneFunc: func(pruneFilter filters.Args) (image.PruneReport, error) {
				assert.Check(t, is.Equal("true", pruneFilter.Get("dangling")[0]))
				return image.PruneReport{
					ImagesDeleted:  []image.DeleteResponse{{Deleted: "image1"}},
					SpaceReclaimed: 1,
				}, nil
			},
		},
		{
			name: "label-filter",
			args: []string{"--force", "--filter", "label=foobar"},
			imagesPruneFunc: func(pruneFilter filters.Args) (image.PruneReport, error) {
				assert.Check(t, is.Equal("foobar", pruneFilter.Get("label")[0]))
				return image.PruneReport{}, nil
			},
		},
		{
			name: "force-untagged",
			args: []string{"--force"},
			imagesPruneFunc: func(pruneFilter filters.Args) (image.PruneReport, error) {
				assert.Check(t, is.Equal("true", pruneFilter.Get("dangling")[0]))
				return image.PruneReport{
					ImagesDeleted:  []image.DeleteResponse{{Untagged: "image1"}},
					SpaceReclaimed: 2,
				}, nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{imagesPruneFunc: tc.imagesPruneFunc})
			// when prompted, answer "Y" to confirm the prune.
			// will not be prompted if --force is used.
			cli.SetIn(streams.NewIn(io.NopCloser(strings.NewReader("Y\n"))))
			cmd := NewPruneCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			assert.NilError(t, err)
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("prune-command-success.%s.golden", tc.name))
		})
	}
}

func TestPrunePromptTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cli := test.NewFakeCli(&fakeClient{
		imagesPruneFunc: func(pruneFilter filters.Args) (image.PruneReport, error) {
			return image.PruneReport{}, errors.New("fakeClient imagesPruneFunc should not be called")
		},
	})
	cmd := NewPruneCommand(cli)
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	test.TerminatePrompt(ctx, t, cmd, cli)
}
