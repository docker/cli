package volume

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	. "github.com/docker/cli/internal/test/builders" // Import builders to get the builder function as package function
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestVolumeInspectErrors(t *testing.T) {
	testCases := []struct {
		args              []string
		flags             map[string]string
		volumeInspectFunc func(volumeID string) (types.Volume, error)
		expectedError     string
	}{
		{
			expectedError: "requires at least 1 argument",
		},
		{
			args: []string{"foo"},
			volumeInspectFunc: func(volumeID string) (types.Volume, error) {
				return types.Volume{}, errors.Errorf("error while inspecting the volume")
			},
			expectedError: "error while inspecting the volume",
		},
		{
			args: []string{"foo"},
			flags: map[string]string{
				"format": "{{invalid format}}",
			},
			expectedError: "Template parsing error",
		},
		{
			args: []string{"foo", "bar"},
			volumeInspectFunc: func(volumeID string) (types.Volume, error) {
				if volumeID == "foo" {
					return types.Volume{
						Name: "foo",
					}, nil
				}
				return types.Volume{}, errors.Errorf("error while inspecting the volume")
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
			cmd.Flags().Set(key, value)
		}
		cmd.SetOut(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestVolumeInspectWithoutFormat(t *testing.T) {
	testCases := []struct {
		name              string
		args              []string
		volumeInspectFunc func(volumeID string) (types.Volume, error)
	}{
		{
			name: "single-volume",
			args: []string{"foo"},
			volumeInspectFunc: func(volumeID string) (types.Volume, error) {
				if volumeID != "foo" {
					return types.Volume{}, errors.Errorf("Invalid volumeID, expected %s, got %s", "foo", volumeID)
				}
				return *Volume(), nil
			},
		},
		{
			name: "multiple-volume-with-labels",
			args: []string{"foo", "bar"},
			volumeInspectFunc: func(volumeID string) (types.Volume, error) {
				return *Volume(VolumeName(volumeID), VolumeLabels(map[string]string{
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
	volumeInspectFunc := func(volumeID string) (types.Volume, error) {
		return *Volume(VolumeLabels(map[string]string{
			"foo": "bar",
		})), nil
	}
	testCases := []struct {
		name              string
		format            string
		args              []string
		volumeInspectFunc func(volumeID string) (types.Volume, error)
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
		cmd.Flags().Set("format", tc.format)
		assert.NilError(t, cmd.Execute())
		golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("volume-inspect-with-format.%s.golden", tc.name))
	}
}

func TestVolumeInspectCluster(t *testing.T) {
	volumeInspectFunc := func(volumeID string) (types.Volume, error) {
		return types.Volume{
			Name:   "clustervolume",
			Driver: "clusterdriver1",
			Scope:  "global",
			ClusterVolume: &types.ClusterVolume{
				ID: "fooid",
				Meta: swarm.Meta{
					Version: swarm.Version{
						Index: uint64(123),
					},
				},
				Spec: types.ClusterVolumeSpec{
					Group: "group0",
					AccessMode: &types.VolumeAccessMode{
						Scope:       types.VolumeScopeMultiNode,
						Sharing:     types.VolumeSharingAll,
						BlockVolume: &types.VolumeTypeBlock{},
					},
					AccessibilityRequirements: &types.TopologyRequirement{
						Requisite: []types.Topology{
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
						Preferred: []types.Topology{
							{
								Segments: map[string]string{
									"region": "R1",
									"zone":   "Z1",
								},
							},
						},
					},
					CapacityRange: &types.VolumeCapacityRange{
						RequiredBytes: 1000,
						LimitBytes:    1000000,
					},
					Secrets: []types.VolumeSecret{
						{
							Key:    "secretkey1",
							Secret: "mysecret1",
						}, {
							Key:    "secretkey2",
							Secret: "mysecret2",
						},
					},
					Availability: types.VolumeAvailabilityActive,
				},
				Info: &types.VolumeInfo{
					CapacityBytes: 10000,
					VolumeContext: map[string]string{
						"the": "context",
						"has": "entries",
					},
					VolumeID: "clusterdriver1volume1id",
					AccessibleTopology: []types.Topology{
						{
							Segments: map[string]string{
								"region": "R1",
								"zone":   "Z1",
							},
						},
					},
				},
				PublishStatus: []*types.VolumePublishStatus{
					{
						NodeID: "node1",
						State:  types.VolumePublished,
						PublishContext: map[string]string{
							"some": "data",
							"yup":  "data",
						},
					}, {
						NodeID: "node2",
						State:  types.VolumePendingNodeUnpublish,
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
