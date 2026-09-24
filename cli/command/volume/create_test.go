// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.26

package volume

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/google/go-cmp/cmp/cmpopts"
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
			if !maps.Equal(options.DriverOpts, expectedOpts) {
				return client.VolumeCreateResult{}, fmt.Errorf("expected drivers opts %v, got %v", expectedOpts, options.DriverOpts)
			}
			if !maps.Equal(options.Labels, expectedLabels) {
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
			assert.Check(t, is.DeepEqual(options, expectedOptions, cmpopts.SortSlices(func(a, b volume.Secret) bool {
				return a.Key < b.Key
			})))
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

// TestVolumeCreateClusterOptionDetection verifies that each cluster-specific
// flag on its own is enough to send a ClusterVolumeSpec, and that its value
// is propagated.
func TestVolumeCreateClusterOptionDetection(t *testing.T) {
	testCases := []struct {
		flag     string
		value    string
		expected *volume.ClusterVolumeSpec
	}{
		{
			flag:  "secret",
			value: "key1=secret1",
			expected: &volume.ClusterVolumeSpec{
				Secrets: []volume.Secret{{Key: "key1", Secret: "secret1"}},
			},
		},
		{
			flag:  "topology-required",
			value: "region=R1,zone=Z1",
			expected: &volume.ClusterVolumeSpec{
				AccessibilityRequirements: &volume.TopologyRequirement{
					Requisite: []volume.Topology{
						{Segments: map[string]string{"region": "R1", "zone": "Z1"}},
					},
					Preferred: []volume.Topology{},
				},
			},
		},
		{
			flag:  "topology-preferred",
			value: "region=R1,zone=Z2",
			expected: &volume.ClusterVolumeSpec{
				AccessibilityRequirements: &volume.TopologyRequirement{
					Requisite: []volume.Topology{},
					Preferred: []volume.Topology{
						{Segments: map[string]string{"region": "R1", "zone": "Z2"}},
					},
				},
			},
		},
		{
			flag:  "limit-bytes",
			value: "567890",
			expected: &volume.ClusterVolumeSpec{
				CapacityRange: &volume.CapacityRange{LimitBytes: 567890},
			},
		},
		{
			flag:  "required-bytes",
			value: "1234",
			expected: &volume.ClusterVolumeSpec{
				CapacityRange: &volume.CapacityRange{RequiredBytes: 1234},
			},
		},
		{
			flag:  "scope",
			value: "multi",
			expected: &volume.ClusterVolumeSpec{
				AccessMode: &volume.AccessMode{
					Scope:       volume.ScopeMultiNode,
					Sharing:     volume.SharingNone,
					BlockVolume: &volume.TypeBlock{},
				},
			},
		},
		{
			flag:  "sharing",
			value: "all",
			expected: &volume.ClusterVolumeSpec{
				AccessMode: &volume.AccessMode{
					Scope:       volume.ScopeSingleNode,
					Sharing:     volume.SharingAll,
					BlockVolume: &volume.TypeBlock{},
				},
			},
		},
		{
			flag:  "type",
			value: "mount",
			expected: &volume.ClusterVolumeSpec{
				AccessMode: &volume.AccessMode{
					Scope:       volume.ScopeSingleNode,
					Sharing:     volume.SharingNone,
					MountVolume: &volume.TypeMount{},
				},
			},
		},
		{
			flag:  "group",
			value: "gronp",
			expected: &volume.ClusterVolumeSpec{
				Group: "gronp",
			},
		},
		{
			flag:  "availability",
			value: "drain",
			expected: &volume.ClusterVolumeSpec{
				Availability: volume.AvailabilityDrain,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.flag, func(t *testing.T) {
			var actual *volume.ClusterVolumeSpec
			cli := test.NewFakeCli(&fakeClient{
				volumeCreateFunc: func(options client.VolumeCreateOptions) (client.VolumeCreateResult, error) {
					actual = options.ClusterVolumeSpec
					return client.VolumeCreateResult{}, nil
				},
			})

			cmd := newCreateCommand(cli)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{"my-csi-volume"})
			assert.Check(t, cmd.Flags().Set("driver", "csi"))
			assert.Check(t, cmd.Flags().Set(tc.flag, tc.value))
			assert.NilError(t, cmd.Execute())

			assert.Assert(t, actual != nil, "expected a ClusterVolumeSpec when only --%s is set", tc.flag)
			if tc.expected.Secrets != nil {
				assert.Check(t, is.DeepEqual(actual.Secrets, tc.expected.Secrets))
			}
			if tc.expected.AccessMode != nil {
				assert.Check(t, is.DeepEqual(actual.AccessMode, tc.expected.AccessMode))
			}
			if tc.expected.AccessibilityRequirements != nil {
				assert.Check(t, is.DeepEqual(actual.AccessibilityRequirements, tc.expected.AccessibilityRequirements))
			}
			if tc.expected.CapacityRange != nil {
				assert.Check(t, is.DeepEqual(actual.CapacityRange, tc.expected.CapacityRange))
			}
			if tc.expected.Group != "" {
				assert.Check(t, is.Equal(actual.Group, tc.expected.Group))
			}
			if tc.expected.Availability != "" {
				assert.Check(t, is.Equal(actual.Availability, tc.expected.Availability))
			}
		})
	}
}
