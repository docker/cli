package image

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestNewPruneCommandErrors(t *testing.T) {
	testCases := []struct {
		name           string
		args           []string
		expectedError  string
		imagePruneFunc func(client.ImagePruneOptions) (client.ImagePruneResult, error)
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
			imagePruneFunc: func(client.ImagePruneOptions) (client.ImagePruneResult, error) {
				return client.ImagePruneResult{}, errors.New("something went wrong")
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newPruneCommand(test.NewFakeCli(&fakeClient{
				imagePruneFunc: tc.imagePruneFunc,
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
		name           string
		args           []string
		imagePruneFunc func(client.ImagePruneOptions) (client.ImagePruneResult, error)
	}{
		{
			name: "all",
			args: []string{"--all"},
			imagePruneFunc: func(opts client.ImagePruneOptions) (client.ImagePruneResult, error) {
				assert.Check(t, opts.Filters["dangling"]["false"])
				return client.ImagePruneResult{}, nil
			},
		},
		{
			name: "force-deleted",
			args: []string{"--force"},
			imagePruneFunc: func(opts client.ImagePruneOptions) (client.ImagePruneResult, error) {
				assert.Check(t, opts.Filters["dangling"]["true"])
				return client.ImagePruneResult{
					Report: image.PruneReport{
						ImagesDeleted:  []image.DeleteResponse{{Deleted: "image1"}},
						SpaceReclaimed: 1,
					},
				}, nil
			},
		},
		{
			name: "label-filter",
			args: []string{"--force", "--filter", "label=foobar"},
			imagePruneFunc: func(opts client.ImagePruneOptions) (client.ImagePruneResult, error) {
				assert.Check(t, opts.Filters["label"]["foobar"])
				return client.ImagePruneResult{}, nil
			},
		},
		{
			name: "force-untagged",
			args: []string{"--force"},
			imagePruneFunc: func(opts client.ImagePruneOptions) (client.ImagePruneResult, error) {
				assert.Check(t, opts.Filters["dangling"]["true"])
				return client.ImagePruneResult{
					Report: image.PruneReport{
						ImagesDeleted:  []image.DeleteResponse{{Untagged: "image1"}},
						SpaceReclaimed: 2,
					},
				}, nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{imagePruneFunc: tc.imagePruneFunc})
			// when prompted, answer "Y" to confirm the prune.
			// will not be prompted if --force is used.
			cli.SetIn(streams.NewIn(io.NopCloser(strings.NewReader("Y\n"))))
			cmd := newPruneCommand(cli)
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
		imagePruneFunc: func(client.ImagePruneOptions) (client.ImagePruneResult, error) {
			return client.ImagePruneResult{}, errors.New("fakeClient imagePruneFunc should not be called")
		},
	})
	cmd := newPruneCommand(cli)
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	test.TerminatePrompt(ctx, t, cmd, cli)
}
