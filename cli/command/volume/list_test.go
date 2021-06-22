package volume

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	. "github.com/docker/cli/internal/test/builders" // Import builders to get the builder function as package function
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
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
		cmd.SetOut(ioutil.Discard)
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

func TestVolumeListSortOrder(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		volumeListFunc: func(filter filters.Args) (volumetypes.VolumeListOKBody, error) {
			return volumetypes.VolumeListOKBody{
				Volumes: []*types.Volume{
					Volume(VolumeName("volume-2-foo")),
					Volume(VolumeName("volume-10-foo")),
					Volume(VolumeName("volume-1-foo")),
				},
			}, nil
		},
	})
	cmd := newListCommand(cli)
	cmd.Flags().Set("format", "{{ .Name }}")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "volume-list-sort.golden")
}

func TestClusterVolumeList(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		volumeListFunc: func(filter filters.Args) (volumetypes.VolumeListOKBody, error) {
			return volumetypes.VolumeListOKBody{
				Volumes: []*types.Volume{
					{
						Name:   "volume1",
						Scope:  "global",
						Driver: "driver1",
						ClusterOpts: &types.ClusterVolume{
							Spec: types.ClusterVolumeSpec{
								Group: "group1",
								AccessMode: &types.VolumeAccessMode{
									Scope:       types.VolumeScopeSingleNode,
									Sharing:     types.VolumeSharingOneWriter,
									MountVolume: &types.VolumeTypeMount{},
								},
								Availability: types.VolumeAvailabilityActive,
							},
						},
					}, {
						Name:   "volume2",
						Scope:  "global",
						Driver: "driver1",
						ClusterOpts: &types.ClusterVolume{
							Spec: types.ClusterVolumeSpec{
								Group: "group1",
								AccessMode: &types.VolumeAccessMode{
									Scope:       types.VolumeScopeSingleNode,
									Sharing:     types.VolumeSharingOneWriter,
									MountVolume: &types.VolumeTypeMount{},
								},
								Availability: types.VolumeAvailabilityPause,
							},
							Info: &types.VolumeInfo{
								CapacityBytes: 100000000,
								VolumeID:      "driver1vol2",
							},
						},
					}, {
						Name:   "volume3",
						Scope:  "global",
						Driver: "driver2",
						ClusterOpts: &types.ClusterVolume{
							Spec: types.ClusterVolumeSpec{
								Group: "group2",
								AccessMode: &types.VolumeAccessMode{
									Scope:       types.VolumeScopeMultiNode,
									Sharing:     types.VolumeSharingAll,
									MountVolume: &types.VolumeTypeMount{},
								},
								Availability: types.VolumeAvailabilityActive,
							},
							PublishStatus: []*types.VolumePublishStatus{
								{
									NodeID: "nodeid1",
									State:  types.VolumePublished,
								},
							},
							Info: &types.VolumeInfo{
								CapacityBytes: 100000000,
								VolumeID:      "driver1vol3",
							},
						},
					}, {
						Name:   "volume4",
						Scope:  "global",
						Driver: "driver2",
						ClusterOpts: &types.ClusterVolume{
							Spec: types.ClusterVolumeSpec{
								Group: "group2",
								AccessMode: &types.VolumeAccessMode{
									Scope:       types.VolumeScopeMultiNode,
									Sharing:     types.VolumeSharingAll,
									MountVolume: &types.VolumeTypeMount{},
								},
								Availability: types.VolumeAvailabilityActive,
							},
							PublishStatus: []*types.VolumePublishStatus{
								{
									NodeID: "nodeid1",
									State:  types.VolumePublished,
								}, {
									NodeID: "nodeid2",
									State:  types.VolumePublished,
								},
							},
							Info: &types.VolumeInfo{
								CapacityBytes: 100000000,
								VolumeID:      "driver1vol4",
							},
						},
					},
					Volume(VolumeName("volume-local-1")),
				},
			}, nil
		},
	})

	cmd := newListCommand(cli)
	cmd.Flags().Set("cluster", "true")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "volume-cluster-volume-list.golden")
}
