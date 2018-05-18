package volume

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/pkg/errors"
	// Import builders to get the builder function as package function
	. "github.com/docker/cli/internal/test/builders"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/golden"
)

func TestVolumeListErrors(t *testing.T) {
	testCases := []struct {
		args           []string
		flags          map[string]string
		volumeListFunc func(filter filters.Args) (volumetypes.VolumeListOKBody, error)
		expectedError  string
	}{
		{
			args:          []string{"foo"},
			expectedError: "accepts no argument",
		},
		{
			volumeListFunc: func(filter filters.Args) (volumetypes.VolumeListOKBody, error) {
				return volumetypes.VolumeListOKBody{}, errors.Errorf("error listing volumes")
			},
			expectedError: "error listing volumes",
		},
	}
	for _, tc := range testCases {
		cmd := newListCommand(
			test.NewFakeCli(&fakeClient{
				volumeListFunc: tc.volumeListFunc,
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

func TestVolumeListWithoutFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		volumeListFunc: func(filter filters.Args) (volumetypes.VolumeListOKBody, error) {
			return volumetypes.VolumeListOKBody{
				Volumes: []*types.Volume{
					Volume(),
					Volume(VolumeName("foo"), VolumeDriver("bar")),
					Volume(VolumeName("baz"), VolumeLabels(map[string]string{
						"foo": "bar",
					})),
				},
			}, nil
		},
	})
	cmd := newListCommand(cli)
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "volume-list-without-format.golden")
}

func TestVolumeListWithConfigFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		volumeListFunc: func(filter filters.Args) (volumetypes.VolumeListOKBody, error) {
			return volumetypes.VolumeListOKBody{
				Volumes: []*types.Volume{
					Volume(),
					Volume(VolumeName("foo"), VolumeDriver("bar")),
					Volume(VolumeName("baz"), VolumeLabels(map[string]string{
						"foo": "bar",
					})),
				},
			}, nil
		},
	})
	cli.SetConfigFile(&configfile.ConfigFile{
		VolumesFormat: "{{ .Name }} {{ .Driver }} {{ .Labels }}",
	})
	cmd := newListCommand(cli)
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "volume-list-with-config-format.golden")
}

func TestVolumeListWithFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		volumeListFunc: func(filter filters.Args) (volumetypes.VolumeListOKBody, error) {
			return volumetypes.VolumeListOKBody{
				Volumes: []*types.Volume{
					Volume(),
					Volume(VolumeName("foo"), VolumeDriver("bar")),
					Volume(VolumeName("baz"), VolumeLabels(map[string]string{
						"foo": "bar",
					})),
				},
			}, nil
		},
	})
	cmd := newListCommand(cli)
	cmd.Flags().Set("format", "{{ .Name }} {{ .Driver }} {{ .Labels }}")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "volume-list-with-format.golden")
}
