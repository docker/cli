package volume

import (
	"fmt"
	"io/ioutil"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/pkg/errors"
	"gotest.tools/assert"
	"gotest.tools/golden"
	"gotest.tools/skip"
)

func TestVolumePruneErrors(t *testing.T) {
	testCases := []struct {
		args            []string
		flags           map[string]string
		volumePruneFunc func(args filters.Args) (types.VolumesPruneReport, error)
		expectedError   string
	}{
		{
			args:          []string{"foo"},
			expectedError: "accepts no argument",
		},
		{
			flags: map[string]string{
				"force": "true",
			},
			volumePruneFunc: func(args filters.Args) (types.VolumesPruneReport, error) {
				return types.VolumesPruneReport{}, errors.Errorf("error pruning volumes")
			},
			expectedError: "error pruning volumes",
		},
	}
	for _, tc := range testCases {
		cmd := NewPruneCommand(
			test.NewFakeCli(&fakeClient{
				volumePruneFunc: tc.volumePruneFunc,
			}),
		)
		cmd.SetArgs(tc.args)
		for key, value := range tc.flags {
			cmd.Flags().Set(key, value)
		}
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestVolumePruneForce(t *testing.T) {
	testCases := []struct {
		name            string
		volumePruneFunc func(args filters.Args) (types.VolumesPruneReport, error)
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

		cli.SetIn(streams.NewIn(ioutil.NopCloser(strings.NewReader(input))))
		cmd := NewPruneCommand(cli)
		assert.NilError(t, cmd.Execute())
		golden.Assert(t, cli.OutBuffer().String(), "volume-prune-yes.golden")
	}
}

func TestVolumePrunePromptNo(t *testing.T) {
	// FIXME(vdemeester) make it work..
	skip.If(t, runtime.GOOS == "windows", "TODO: fix test on windows")

	for _, input := range []string{"n", "N", "no", "anything", "really"} {
		cli := test.NewFakeCli(&fakeClient{
			volumePruneFunc: simplePruneFunc,
		})

		cli.SetIn(streams.NewIn(ioutil.NopCloser(strings.NewReader(input))))
		cmd := NewPruneCommand(cli)
		assert.NilError(t, cmd.Execute())
		golden.Assert(t, cli.OutBuffer().String(), "volume-prune-no.golden")
	}
}

func simplePruneFunc(args filters.Args) (types.VolumesPruneReport, error) {
	return types.VolumesPruneReport{
		VolumesDeleted: []string{
			"foo", "bar", "baz",
		},
		SpaceReclaimed: 2000,
	}, nil
}
