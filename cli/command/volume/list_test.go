package volume

import (
	"io"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestVolumeListErrors(t *testing.T) {
	testCases := []struct {
		args           []string
		flags          map[string]string
		volumeListFunc func(filter filters.Args) (volume.ListResponse, error)
		expectedError  string
	}{
		{
			args:          []string{"foo"},
			expectedError: "accepts no argument",
		},
		{
			volumeListFunc: func(filter filters.Args) (volume.ListResponse, error) {
				return volume.ListResponse{}, errors.Errorf("error listing volumes")
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
			assert.Check(t, cmd.Flags().Set(key, value))
		}
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestVolumeListWithoutFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		volumeListFunc: func(filter filters.Args) (volume.ListResponse, error) {
			return volume.ListResponse{
				Volumes: []*volume.Volume{
					builders.Volume(),
					builders.Volume(builders.VolumeName("foo"), builders.VolumeDriver("bar")),
					builders.Volume(builders.VolumeName("baz"), builders.VolumeLabels(map[string]string{
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
		volumeListFunc: func(filter filters.Args) (volume.ListResponse, error) {
			return volume.ListResponse{
				Volumes: []*volume.Volume{
					builders.Volume(),
					builders.Volume(builders.VolumeName("foo"), builders.VolumeDriver("bar")),
					builders.Volume(builders.VolumeName("baz"), builders.VolumeLabels(map[string]string{
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
		volumeListFunc: func(filter filters.Args) (volume.ListResponse, error) {
			return volume.ListResponse{
				Volumes: []*volume.Volume{
					builders.Volume(),
					builders.Volume(builders.VolumeName("foo"), builders.VolumeDriver("bar")),
					builders.Volume(builders.VolumeName("baz"), builders.VolumeLabels(map[string]string{
						"foo": "bar",
					})),
				},
			}, nil
		},
	})
	cmd := newListCommand(cli)
	assert.Check(t, cmd.Flags().Set("format", "{{ .Name }} {{ .Driver }} {{ .Labels }}"))
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "volume-list-with-format.golden")
}

func TestVolumeListSortOrder(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		volumeListFunc: func(filter filters.Args) (volume.ListResponse, error) {
			return volume.ListResponse{
				Volumes: []*volume.Volume{
					builders.Volume(builders.VolumeName("volume-2-foo")),
					builders.Volume(builders.VolumeName("volume-10-foo")),
					builders.Volume(builders.VolumeName("volume-1-foo")),
				},
			}, nil
		},
	})
	cmd := newListCommand(cli)
	assert.Check(t, cmd.Flags().Set("format", "{{ .Name }}"))
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "volume-list-sort.golden")
}

func TestClusterVolumeList(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		volumeListFunc: func(filter filters.Args) (volume.ListResponse, error) {
			return volume.ListResponse{
				Volumes: []*volume.Volume{
					{
						Name:   "volume1",
						Scope:  "global",
						Driver: "driver1",
						ClusterVolume: &volume.ClusterVolume{
							Spec: volume.ClusterVolumeSpec{
								Group: "group1",
								AccessMode: &volume.AccessMode{
									Scope:       volume.ScopeSingleNode,
									Sharing:     volume.SharingOneWriter,
									MountVolume: &volume.TypeMount{},
								},
								Availability: volume.AvailabilityActive,
							},
						},
					},
					{
						Name:   "volume2",
						Scope:  "global",
						Driver: "driver1",
						ClusterVolume: &volume.ClusterVolume{
							Spec: volume.ClusterVolumeSpec{
								Group: "group1",
								AccessMode: &volume.AccessMode{
									Scope:       volume.ScopeSingleNode,
									Sharing:     volume.SharingOneWriter,
									MountVolume: &volume.TypeMount{},
								},
								Availability: volume.AvailabilityPause,
							},
							Info: &volume.Info{
								CapacityBytes: 100000000,
								VolumeID:      "driver1vol2",
							},
						},
					},
					{
						Name:   "volume3",
						Scope:  "global",
						Driver: "driver2",
						ClusterVolume: &volume.ClusterVolume{
							Spec: volume.ClusterVolumeSpec{
								Group: "group2",
								AccessMode: &volume.AccessMode{
									Scope:       volume.ScopeMultiNode,
									Sharing:     volume.SharingAll,
									MountVolume: &volume.TypeMount{},
								},
								Availability: volume.AvailabilityActive,
							},
							PublishStatus: []*volume.PublishStatus{
								{
									NodeID: "nodeid1",
									State:  volume.StatePublished,
								},
							},
							Info: &volume.Info{
								CapacityBytes: 100000000,
								VolumeID:      "driver1vol3",
							},
						},
					},
					{
						Name:   "volume4",
						Scope:  "global",
						Driver: "driver2",
						ClusterVolume: &volume.ClusterVolume{
							Spec: volume.ClusterVolumeSpec{
								Group: "group2",
								AccessMode: &volume.AccessMode{
									Scope:       volume.ScopeMultiNode,
									Sharing:     volume.SharingAll,
									MountVolume: &volume.TypeMount{},
								},
								Availability: volume.AvailabilityActive,
							},
							PublishStatus: []*volume.PublishStatus{
								{
									NodeID: "nodeid1",
									State:  volume.StatePublished,
								}, {
									NodeID: "nodeid2",
									State:  volume.StatePublished,
								},
							},
							Info: &volume.Info{
								CapacityBytes: 100000000,
								VolumeID:      "driver1vol4",
							},
						},
					},
					builders.Volume(builders.VolumeName("volume-local-1")),
				},
			}, nil
		},
	})

	cmd := newListCommand(cli)
	assert.Check(t, cmd.Flags().Set("cluster", "true"))
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "volume-cluster-volume-list.golden")
}
