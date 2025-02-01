package volume

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/volume"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestVolumeCreateErrors(t *testing.T) {
	testCases := []struct {
		args             []string
		flags            map[string]string
		volumeCreateFunc func(volume.CreateOptions) (volume.Volume, error)
		expectedError    string
	}{
		{
			args: []string{"volumeName"},
			flags: map[string]string{
				"name": "volumeName",
			},
			expectedError: "conflicting options: cannot specify a volume-name through both --name and as a positional arg",
		},
		{
			args:          []string{"too", "many"},
			expectedError: "requires at most 1 argument",
		},
		{
			volumeCreateFunc: func(createBody volume.CreateOptions) (volume.Volume, error) {
				return volume.Volume{}, errors.New("error creating volume")
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
			assert.Check(t, cmd.Flags().Set(key, value))
		}
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestVolumeCreateWithName(t *testing.T) {
	const name = "my-volume-name"
	cli := test.NewFakeCli(&fakeClient{
		volumeCreateFunc: func(body volume.CreateOptions) (volume.Volume, error) {
			if body.Name != name {
				return volume.Volume{}, fmt.Errorf("expected name %q, got %q", name, body.Name)
			}
			return volume.Volume{
				Name: body.Name,
			}, nil
		},
	})

	buf := cli.OutBuffer()
	t.Run("using-flags", func(t *testing.T) {
		cmd := newCreateCommand(cli)
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{})
		assert.Check(t, cmd.Flags().Set("name", name))
		assert.NilError(t, cmd.Execute())
		assert.Check(t, is.Equal(strings.TrimSpace(buf.String()), name))
	})

	buf.Reset()
	t.Run("using-args", func(t *testing.T) {
		cmd := newCreateCommand(cli)
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{name})
		assert.NilError(t, cmd.Execute())
		assert.Check(t, is.Equal(strings.TrimSpace(buf.String()), name))
	})

	buf.Reset()
	t.Run("using-both", func(t *testing.T) {
		cmd := newCreateCommand(cli)
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{name})
		assert.Check(t, cmd.Flags().Set("name", name))
		err := cmd.Execute()
		assert.Check(t, is.Error(err, `conflicting options: cannot specify a volume-name through both --name and as a positional arg`))
		assert.Check(t, is.Equal(strings.TrimSpace(buf.String()), ""))
	})
}

func TestVolumeCreateWithFlags(t *testing.T) {
	const name = "random-generated-name"
	const expectedDriver = "foo-volume-driver"
	expectedOpts := map[string]string{
		"bar": "1",
		"baz": "baz",
	}
	expectedLabels := map[string]string{
		"lbl1": "v1",
		"lbl2": "v2",
	}

	cli := test.NewFakeCli(&fakeClient{
		volumeCreateFunc: func(body volume.CreateOptions) (volume.Volume, error) {
			if body.Name != "" {
				return volume.Volume{}, fmt.Errorf("expected empty name, got %q", body.Name)
			}
			if body.Driver != expectedDriver {
				return volume.Volume{}, fmt.Errorf("expected driver %q, got %q", expectedDriver, body.Driver)
			}
			if !reflect.DeepEqual(body.DriverOpts, expectedOpts) {
				return volume.Volume{}, fmt.Errorf("expected drivers opts %v, got %v", expectedOpts, body.DriverOpts)
			}
			if !reflect.DeepEqual(body.Labels, expectedLabels) {
				return volume.Volume{}, fmt.Errorf("expected labels %v, got %v", expectedLabels, body.Labels)
			}
			return volume.Volume{
				Name: name,
			}, nil
		},
	})

	cmd := newCreateCommand(cli)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{})
	assert.Check(t, cmd.Flags().Set("driver", expectedDriver))
	assert.Check(t, cmd.Flags().Set("opt", "bar=1"))
	assert.Check(t, cmd.Flags().Set("opt", "baz=baz"))
	assert.Check(t, cmd.Flags().Set("label", "lbl1=v1"))
	assert.Check(t, cmd.Flags().Set("label", "lbl2=v2"))
	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.Equal(strings.TrimSpace(cli.OutBuffer().String()), name))
}

func TestVolumeCreateCluster(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		volumeCreateFunc: func(body volume.CreateOptions) (volume.Volume, error) {
			if body.Driver == "csi" && body.ClusterVolumeSpec == nil {
				return volume.Volume{}, errors.New("expected ClusterVolumeSpec, but none present")
			}
			if body.Driver == "notcsi" && body.ClusterVolumeSpec != nil {
				return volume.Volume{}, errors.New("expected no ClusterVolumeSpec, but present")
			}
			return volume.Volume{}, nil
		},
	})

	t.Run("csi-volume", func(t *testing.T) {
		cmd := newCreateCommand(cli)
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.Check(t, cmd.Flags().Set("type", "block"))
		assert.Check(t, cmd.Flags().Set("group", "gronp"))
		assert.Check(t, cmd.Flags().Set("driver", "csi"))
		cmd.SetArgs([]string{"my-csi-volume"})

		assert.NilError(t, cmd.Execute())
	})

	t.Run("non-csi-volume", func(t *testing.T) {
		cmd := newCreateCommand(cli)
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.Check(t, cmd.Flags().Set("driver", "notcsi"))
		cmd.SetArgs([]string{"my-non-csi-volume"})

		assert.NilError(t, cmd.Execute())
	})
}

func TestVolumeCreateClusterOpts(t *testing.T) {
	expectedBody := volume.CreateOptions{
		Name:       "name",
		Driver:     "csi",
		DriverOpts: map[string]string{},
		Labels:     map[string]string{},
		ClusterVolumeSpec: &volume.ClusterVolumeSpec{
			Group: "gronp",
			AccessMode: &volume.AccessMode{
				Scope:   volume.ScopeMultiNode,
				Sharing: volume.SharingOneWriter,
				// TODO(dperny): support mount options
				MountVolume: &volume.TypeMount{},
			},
			// TODO(dperny): topology requirements
			CapacityRange: &volume.CapacityRange{
				RequiredBytes: 1234,
				LimitBytes:    567890,
			},
			Secrets: []volume.Secret{
				{Key: "key1", Secret: "secret1"},
				{Key: "key2", Secret: "secret2"},
			},
			Availability: volume.AvailabilityActive,
			AccessibilityRequirements: &volume.TopologyRequirement{
				Requisite: []volume.Topology{
					{Segments: map[string]string{"region": "R1", "zone": "Z1"}},
					{Segments: map[string]string{"region": "R1", "zone": "Z2"}},
					{Segments: map[string]string{"region": "R1", "zone": "Z3"}},
				},
				Preferred: []volume.Topology{
					{Segments: map[string]string{"region": "R1", "zone": "Z2"}},
					{Segments: map[string]string{"region": "R1", "zone": "Z3"}},
				},
			},
		},
	}

	cli := test.NewFakeCli(&fakeClient{
		volumeCreateFunc: func(body volume.CreateOptions) (volume.Volume, error) {
			sort.SliceStable(body.ClusterVolumeSpec.Secrets, func(i, j int) bool {
				return body.ClusterVolumeSpec.Secrets[i].Key < body.ClusterVolumeSpec.Secrets[j].Key
			})
			assert.DeepEqual(t, body, expectedBody)
			return volume.Volume{}, nil
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

	cmd.Flags().Set("topology-required", "region=R1,zone=Z1")
	cmd.Flags().Set("topology-required", "region=R1,zone=Z2")
	cmd.Flags().Set("topology-required", "region=R1,zone=Z3")

	cmd.Flags().Set("topology-preferred", "region=R1,zone=Z2")
	cmd.Flags().Set("topology-preferred", "region=R1,zone=Z3")

	cmd.Execute()
}
