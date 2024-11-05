package volume

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
	"gotest.tools/v3/skip"
)

func TestVolumePruneErrors(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		flags           map[string]string
		volumePruneFunc func(args filters.Args) (volume.PruneReport, error)
		expectedError   string
	}{
		{
			name:          "accepts no arguments",
			args:          []string{"foo"},
			expectedError: "accepts no argument",
		},
		{
			name: "forced but other error",
			flags: map[string]string{
				"force": "true",
			},
			volumePruneFunc: func(args filters.Args) (volume.PruneReport, error) {
				return volume.PruneReport{}, errors.Errorf("error pruning volumes")
			},
			expectedError: "error pruning volumes",
		},
		{
			name: "conflicting options",
			flags: map[string]string{
				"all":    "true",
				"filter": "all=1",
			},
			expectedError: "conflicting options: cannot specify both --all and --filter all=1",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewPruneCommand(
				test.NewFakeCli(&fakeClient{
					volumePruneFunc: tc.volumePruneFunc,
				}),
			)
			cmd.SetArgs(tc.args)
			for key, value := range tc.flags {
				cmd.Flags().Set(key, value)
			}
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestVolumePruneSuccess(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		input           string
		volumePruneFunc func(args filters.Args) (volume.PruneReport, error)
	}{
		{
			name:  "all",
			args:  []string{"--all"},
			input: "y",
			volumePruneFunc: func(pruneFilter filters.Args) (volume.PruneReport, error) {
				assert.Check(t, is.DeepEqual([]string{"true"}, pruneFilter.Get("all")))
				return volume.PruneReport{}, nil
			},
		},
		{
			name: "all-forced",
			args: []string{"--all", "--force"},
			volumePruneFunc: func(pruneFilter filters.Args) (volume.PruneReport, error) {
				return volume.PruneReport{}, nil
			},
		},
		{
			name:  "label-filter",
			args:  []string{"--filter", "label=foobar"},
			input: "y",
			volumePruneFunc: func(pruneFilter filters.Args) (volume.PruneReport, error) {
				assert.Check(t, is.DeepEqual([]string{"foobar"}, pruneFilter.Get("label")))
				return volume.PruneReport{}, nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{volumePruneFunc: tc.volumePruneFunc})
			cmd := NewPruneCommand(cli)
			if tc.input != "" {
				cli.SetIn(streams.NewIn(io.NopCloser(strings.NewReader(tc.input))))
			}
			cmd.SetOut(io.Discard)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			assert.NilError(t, err)
			golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("volume-prune-success.%s.golden", tc.name))
		})
	}
}

func TestVolumePruneForce(t *testing.T) {
	testCases := []struct {
		name            string
		volumePruneFunc func(args filters.Args) (volume.PruneReport, error)
	}{
		{
			name: "empty",
		},
		{
			name:            "deletedVolumes",
			volumePruneFunc: simplePruneFunc,
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			volumePruneFunc: tc.volumePruneFunc,
		})
		cmd := NewPruneCommand(cli)
		cmd.Flags().Set("force", "true")
		assert.NilError(t, cmd.Execute())
		golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("volume-prune.%s.golden", tc.name))
	}
}

func TestVolumePrunePromptYes(t *testing.T) {
	// FIXME(vdemeester) make it work..
	skip.If(t, runtime.GOOS == "windows", "TODO: fix test on windows")

	for _, input := range []string{"y", "Y"} {
		cli := test.NewFakeCli(&fakeClient{
			volumePruneFunc: simplePruneFunc,
		})

		cli.SetIn(streams.NewIn(io.NopCloser(strings.NewReader(input))))
		cmd := NewPruneCommand(cli)
		cmd.SetArgs([]string{})
		assert.NilError(t, cmd.Execute())
		golden.Assert(t, cli.OutBuffer().String(), "volume-prune-yes.golden")
	}
}

func TestVolumePrunePromptNo(t *testing.T) {
	// FIXME(vdemeester) make it work..
	skip.If(t, runtime.GOOS == "windows", "TODO: fix test on windows")

	for _, input := range []string{"n", "N", "no", "anything", "really"} {
		t.Run(input, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				volumePruneFunc: simplePruneFunc,
			})

			cli.SetIn(streams.NewIn(io.NopCloser(strings.NewReader(input))))
			cmd := NewPruneCommand(cli)
			cmd.SetArgs([]string{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			assert.ErrorContains(t, cmd.Execute(), "volume prune has been cancelled")
			golden.Assert(t, cli.OutBuffer().String(), "volume-prune-no.golden")
		})
	}
}

func simplePruneFunc(filters.Args) (volume.PruneReport, error) {
	return volume.PruneReport{
		VolumesDeleted: []string{
			"foo", "bar", "baz",
		},
		SpaceReclaimed: 2000,
	}, nil
}

func TestVolumePrunePromptTerminate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cli := test.NewFakeCli(&fakeClient{
		volumePruneFunc: func(filter filters.Args) (volume.PruneReport, error) {
			return volume.PruneReport{}, errors.New("fakeClient volumePruneFunc should not be called")
		},
	})

	cmd := NewPruneCommand(cli)
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	test.TerminatePrompt(ctx, t, cmd, cli)
	golden.Assert(t, cli.OutBuffer().String(), "volume-prune-terminate.golden")
}
