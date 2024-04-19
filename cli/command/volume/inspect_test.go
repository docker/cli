package volume

import (
	"fmt"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/volume"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestVolumeInspectErrors(t *testing.T) {
	testCases := []struct {
		args              []string
		flags             map[string]string
		volumeInspectFunc func(volumeID string) (volume.Volume, error)
		expectedError     string
	}{
		{
			expectedError: "requires at least 1 argument",
		},
		{
			args: []string{"foo"},
			volumeInspectFunc: func(volumeID string) (volume.Volume, error) {
				return volume.Volume{}, errors.Errorf("error while inspecting the volume")
			},
			expectedError: "error while inspecting the volume",
		},
		{
			args: []string{"foo"},
			flags: map[string]string{
				"format": "{{invalid format}}",
			},
			expectedError: "template parsing error",
		},
		{
			args: []string{"foo", "bar"},
			volumeInspectFunc: func(volumeID string) (volume.Volume, error) {
				if volumeID == "foo" {
					return volume.Volume{
						Name: "foo",
					}, nil
				}
				return volume.Volume{}, errors.Errorf("error while inspecting the volume")
			},
			expectedError: "error while inspecting the volume",
		},
	}
	for _, tc := range testCases {
		cmd := newInspectCommand(
			test.NewFakeCli(&fakeClient{
				volumeInspectFunc: tc.volumeInspectFunc,
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

func TestVolumeInspectWithoutFormat(t *testing.T) {
	testCases := []struct {
		name              string
		args              []string
		volumeInspectFunc func(volumeID string) (volume.Volume, error)
	}{
		{
			name: "single-volume",
			args: []string{"foo"},
			volumeInspectFunc: func(volumeID string) (volume.Volume, error) {
				if volumeID != "foo" {
					return volume.Volume{}, errors.Errorf("Invalid volumeID, expected %s, got %s", "foo", volumeID)
				}
				return *builders.Volume(), nil
			},
		},
		{
			name: "multiple-volume-with-labels",
			args: []string{"foo", "bar"},
			volumeInspectFunc: func(volumeID string) (volume.Volume, error) {
				return *builders.Volume(builders.VolumeName(volumeID), builders.VolumeLabels(map[string]string{
					"foo": "bar",
				})), nil
			},
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			volumeInspectFunc: tc.volumeInspectFunc,
		})
		cmd := newInspectCommand(cli)
		cmd.SetArgs(tc.args)
		assert.NilError(t, cmd.Execute())
		golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("volume-inspect-without-format.%s.golden", tc.name))
	}
}

func TestVolumeInspectWithFormat(t *testing.T) {
	volumeInspectFunc := func(volumeID string) (volume.Volume, error) {
		return *builders.Volume(builders.VolumeLabels(map[string]string{
			"foo": "bar",
		})), nil
	}
	testCases := []struct {
		name              string
		format            string
		args              []string
		volumeInspectFunc func(volumeID string) (volume.Volume, error)
	}{
		{
			name:              "simple-template",
			format:            "{{.Name}}",
			args:              []string{"foo"},
			volumeInspectFunc: volumeInspectFunc,
		},
		{
			name:              "json-template",
			format:            "{{json .Labels}}",
			args:              []string{"foo"},
			volumeInspectFunc: volumeInspectFunc,
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			volumeInspectFunc: tc.volumeInspectFunc,
		})
		cmd := newInspectCommand(cli)
		cmd.SetArgs(tc.args)
		assert.Check(t, cmd.Flags().Set("format", tc.format))
		assert.NilError(t, cmd.Execute())
		golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("volume-inspect-with-format.%s.golden", tc.name))
	}
}

func TestVolumeInspectCluster(t *testing.T) {
	volumeInspectFunc := func(volumeID string) (volume.Volume, error) {
		return volume.Volume{
			Name:   "clustervolume",
			Driver: "clusterdriver1",
			Scope:  "global",
			ClusterVolume: &volume.ClusterVolume{
				ID: "fooid",
				Meta: swarm.Meta{
					Version: swarm.Version{
						Index: uint64(123),
					},
				},
				Spec: volume.ClusterVolumeSpec{
					Group: "group0",
					AccessMode: &volume.AccessMode{
						Scope:       volume.ScopeMultiNode,
						Sharing:     volume.SharingAll,
						BlockVolume: &volume.TypeBlock{},
					},
					AccessibilityRequirements: &volume.TopologyRequirement{
						Requisite: []volume.Topology{
							{
								Segments: map[string]string{
									"region": "R1",
									"zone":   "Z1",
								},
							}, {
								Segments: map[string]string{
									"region": "R1",
									"zone":   "Z2",
								},
							},
						},
						Preferred: []volume.Topology{
							{
								Segments: map[string]string{
									"region": "R1",
									"zone":   "Z1",
								},
							},
						},
					},
					CapacityRange: &volume.CapacityRange{
						RequiredBytes: 1000,
						LimitBytes:    1000000,
					},
					Secrets: []volume.Secret{
						{
							Key:    "secretkey1",
							Secret: "mysecret1",
						}, {
							Key:    "secretkey2",
							Secret: "mysecret2",
						},
					},
					Availability: volume.AvailabilityActive,
				},
				Info: &volume.Info{
					CapacityBytes: 10000,
					VolumeContext: map[string]string{
						"the": "context",
						"has": "entries",
					},
					VolumeID: "clusterdriver1volume1id",
					AccessibleTopology: []volume.Topology{
						{
							Segments: map[string]string{
								"region": "R1",
								"zone":   "Z1",
							},
						},
					},
				},
				PublishStatus: []*volume.PublishStatus{
					{
						NodeID: "node1",
						State:  volume.StatePublished,
						PublishContext: map[string]string{
							"some": "data",
							"yup":  "data",
						},
					}, {
						NodeID: "node2",
						State:  volume.StatePendingNodeUnpublish,
						PublishContext: map[string]string{
							"some":    "more",
							"publish": "context",
						},
					},
				},
			},
		}, nil
	}

	cli := test.NewFakeCli(&fakeClient{
		volumeInspectFunc: volumeInspectFunc,
	})

	cmd := newInspectCommand(cli)
	cmd.SetArgs([]string{"clustervolume"})
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "volume-inspect-cluster.golden")
}
