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
	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestVolumeCreateErrors(t *testing.T) {
	testCases := []struct {
		args             []string
		flags            map[string]string
		volumeCreateFunc func(client.VolumeCreateOptions) (client.VolumeCreateResult, error)
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
			volumeCreateFunc: func(client.VolumeCreateOptions) (client.VolumeCreateResult, error) {
				return client.VolumeCreateResult{}, errors.New("error creating volume")
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
		volumeCreateFunc: func(options client.VolumeCreateOptions) (client.VolumeCreateResult, error) {
			if options.Name != name {
				return client.VolumeCreateResult{}, fmt.Errorf("expected name %q, got %q", name, options.Name)
			}
			return client.VolumeCreateResult{
				Volume: volume.Volume{Name: options.Name},
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
		volumeCreateFunc: func(options client.VolumeCreateOptions) (client.VolumeCreateResult, error) {
			if options.Name != "" {
				return client.VolumeCreateResult{}, fmt.Errorf("expected empty name, got %q", options.Name)
			}
			if options.Driver != expectedDriver {
				return client.VolumeCreateResult{}, fmt.Errorf("expected driver %q, got %q", expectedDriver, options.Driver)
			}
			if !reflect.DeepEqual(options.DriverOpts, expectedOpts) {
				return client.VolumeCreateResult{}, fmt.Errorf("expected drivers opts %v, got %v", expectedOpts, options.DriverOpts)
			}
			if !reflect.DeepEqual(options.Labels, expectedLabels) {
				return client.VolumeCreateResult{}, fmt.Errorf("expected labels %v, got %v", expectedLabels, options.Labels)
			}
			return client.VolumeCreateResult{
				Volume: volume.Volume{
					Name: name,
				},
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
		volumeCreateFunc: func(options client.VolumeCreateOptions) (client.VolumeCreateResult, error) {
			if options.Driver == "csi" && options.ClusterVolumeSpec == nil {
				return client.VolumeCreateResult{}, errors.New("expected ClusterVolumeSpec, but none present")
			}
			if options.Driver == "notcsi" && options.ClusterVolumeSpec != nil {
				return client.VolumeCreateResult{}, errors.New("expected no ClusterVolumeSpec, but present")
			}
			return client.VolumeCreateResult{}, nil
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
	expectedOptions := client.VolumeCreateOptions{
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
		volumeCreateFunc: func(options client.VolumeCreateOptions) (client.VolumeCreateResult, error) {
			sort.SliceStable(options.ClusterVolumeSpec.Secrets, func(i, j int) bool {
				return options.ClusterVolumeSpec.Secrets[i].Key < options.ClusterVolumeSpec.Secrets[j].Key
			})
			assert.DeepEqual(t, options, expectedOptions)
			return client.VolumeCreateResult{}, nil
		},
	})

	cmd := newCreateCommand(cli)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"name"})
	assert.Check(t, cmd.Flags().Set("driver", "csi"))
	assert.Check(t, cmd.Flags().Set("group", "gronp"))
	assert.Check(t, cmd.Flags().Set("scope", "multi"))
	assert.Check(t, cmd.Flags().Set("sharing", "onewriter"))
	assert.Check(t, cmd.Flags().Set("type", "mount"))
	assert.Check(t, cmd.Flags().Set("sharing", "onewriter"))
	assert.Check(t, cmd.Flags().Set("required-bytes", "1234"))
	assert.Check(t, cmd.Flags().Set("limit-bytes", "567890"))

	assert.Check(t, cmd.Flags().Set("secret", "key1=secret1"))
	assert.Check(t, cmd.Flags().Set("secret", "key2=secret2"))

	assert.Check(t, cmd.Flags().Set("topology-required", "region=R1,zone=Z1"))
	assert.Check(t, cmd.Flags().Set("topology-required", "region=R1,zone=Z2"))
	assert.Check(t, cmd.Flags().Set("topology-required", "region=R1,zone=Z3"))

	assert.Check(t, cmd.Flags().Set("topology-preferred", "region=R1,zone=Z2"))
	assert.Check(t, cmd.Flags().Set("topology-preferred", "region=R1,zone=Z3"))

	assert.NilError(t, cmd.Execute())
}
