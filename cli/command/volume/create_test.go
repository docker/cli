package volume

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestVolumeCreateErrors(t *testing.T) {
	testCases := []struct {
		args             []string
		flags            map[string]string
		volumeCreateFunc func(volumetypes.VolumeCreateBody) (types.Volume, error)
		expectedError    string
	}{
		{
			args: []string{"volumeName"},
			flags: map[string]string{
				"name": "volumeName",
			},
			expectedError: "Conflicting options: either specify --name or provide positional arg, not both",
		},
		{
			args:          []string{"too", "many"},
			expectedError: "requires at most 1 argument",
		},
		{
			volumeCreateFunc: func(createBody volumetypes.VolumeCreateBody) (types.Volume, error) {
				return types.Volume{}, errors.Errorf("error creating volume")
			},
			expectedError: "error creating volume",
		},
	}
	for _, tc := range testCases {
		cmd := newCreateCommand(
			test.NewFakeCli(&fakeClient{
				volumeCreateFunc: tc.volumeCreateFunc,
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

func TestVolumeCreateWithName(t *testing.T) {
	name := "foo"
	cli := test.NewFakeCli(&fakeClient{
		volumeCreateFunc: func(body volumetypes.VolumeCreateBody) (types.Volume, error) {
			if body.Name != name {
				return types.Volume{}, errors.Errorf("expected name %q, got %q", name, body.Name)
			}
			return types.Volume{
				Name: body.Name,
			}, nil
		},
	})

	buf := cli.OutBuffer()

	// Test by flags
	cmd := newCreateCommand(cli)
	cmd.Flags().Set("name", name)
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal(name, strings.TrimSpace(buf.String())))

	// Then by args
	buf.Reset()
	cmd = newCreateCommand(cli)
	cmd.SetArgs([]string{name})
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal(name, strings.TrimSpace(buf.String())))
}

func TestVolumeCreateWithFlags(t *testing.T) {
	expectedDriver := "foo"
	expectedOpts := map[string]string{
		"bar": "1",
		"baz": "baz",
	}
	expectedLabels := map[string]string{
		"lbl1": "v1",
		"lbl2": "v2",
	}
	name := "banana"

	cli := test.NewFakeCli(&fakeClient{
		volumeCreateFunc: func(body volumetypes.VolumeCreateBody) (types.Volume, error) {
			if body.Name != "" {
				return types.Volume{}, errors.Errorf("expected empty name, got %q", body.Name)
			}
			if body.Driver != expectedDriver {
				return types.Volume{}, errors.Errorf("expected driver %q, got %q", expectedDriver, body.Driver)
			}
			if !reflect.DeepEqual(body.DriverOpts, expectedOpts) {
				return types.Volume{}, errors.Errorf("expected drivers opts %v, got %v", expectedOpts, body.DriverOpts)
			}
			if !reflect.DeepEqual(body.Labels, expectedLabels) {
				return types.Volume{}, errors.Errorf("expected labels %v, got %v", expectedLabels, body.Labels)
			}
			return types.Volume{
				Name: name,
			}, nil
		},
	})

	cmd := newCreateCommand(cli)
	cmd.Flags().Set("driver", "foo")
	cmd.Flags().Set("opt", "bar=1")
	cmd.Flags().Set("opt", "baz=baz")
	cmd.Flags().Set("label", "lbl1=v1")
	cmd.Flags().Set("label", "lbl2=v2")
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal(name, strings.TrimSpace(cli.OutBuffer().String())))
}

func TestVolumeCreateCluster(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		volumeCreateFunc: func(body volumetypes.VolumeCreateBody) (types.Volume, error) {
			if body.Driver == "csi" && body.ClusterVolumeSpec == nil {
				return types.Volume{}, fmt.Errorf("expected ClusterVolumeSpec, but none present")
			}
			if body.Driver == "notcsi" && body.ClusterVolumeSpec != nil {
				return types.Volume{}, fmt.Errorf("expected no ClusterVolumeSpec, but present")
			}
			return types.Volume{}, nil
		},
	})

	cmd := newCreateCommand(cli)
	cmd.Flags().Set("type", "block")
	cmd.Flags().Set("group", "gronp")
	cmd.Flags().Set("driver", "csi")
	cmd.SetArgs([]string{"name"})

	assert.NilError(t, cmd.Execute())

	cmd = newCreateCommand(cli)
	cmd.Flags().Set("driver", "notcsi")
	cmd.SetArgs([]string{"name"})

	assert.NilError(t, cmd.Execute())
}

func TestVolumeCreateClusterOpts(t *testing.T) {
	expectedBody := volumetypes.VolumeCreateBody{
		Name:       "name",
		Driver:     "csi",
		DriverOpts: map[string]string{},
		Labels:     map[string]string{},
		ClusterVolumeSpec: &types.ClusterVolumeSpec{
			Group: "gronp",
			AccessMode: &types.VolumeAccessMode{
				Scope:   types.VolumeScopeMultiNode,
				Sharing: types.VolumeSharingOneWriter,
				// TODO(dperny): support mount options
				MountVolume: &types.VolumeTypeMount{},
			},
			// TODO(dperny): topology requirements
			CapacityRange: &types.VolumeCapacityRange{
				RequiredBytes: uint64(1234),
				LimitBytes:    uint64(567890),
			},
			Secrets: []types.VolumeSecret{
				{Key: "key1", Secret: "secret1"},
				{Key: "key2", Secret: "secret2"},
			},
			Availability: types.VolumeAvailabilityActive,
		},
	}

	cli := test.NewFakeCli(&fakeClient{
		volumeCreateFunc: func(body volumetypes.VolumeCreateBody) (types.Volume, error) {
			assert.DeepEqual(t, body, expectedBody)
			return types.Volume{}, nil
		},
	})

	cmd := newCreateCommand(cli)
	cmd.SetArgs([]string{"name"})
	cmd.Flags().Set("driver", "csi")
	cmd.Flags().Set("group", "gronp")
	cmd.Flags().Set("scope", "multi")
	cmd.Flags().Set("sharing", "onewriter")
	cmd.Flags().Set("type", "mount")
	cmd.Flags().Set("sharing", "onewriter")
	cmd.Flags().Set("required-bytes", "1234")
	cmd.Flags().Set("limit-bytes", "567890")
	cmd.Flags().Set("secret", "key1=secret1")
	cmd.Flags().Set("secret", "key2=secret2")

	cmd.Execute()
}
