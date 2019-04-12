package service

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"

	cliopts "github.com/docker/cli/opts"
)

// fakeConfigAPIClientList is used to let us pass a closure as a
// ConfigAPIClient, to use as ConfigList. for all the other methods in the
// interface, it does nothing, not even return an error, so don't use them
type fakeConfigAPIClientList func(context.Context, types.ConfigListOptions) ([]swarm.Config, error)

func (f fakeConfigAPIClientList) ConfigList(ctx context.Context, opts types.ConfigListOptions) ([]swarm.Config, error) {
	return f(ctx, opts)
}

func (f fakeConfigAPIClientList) ConfigCreate(_ context.Context, _ swarm.ConfigSpec) (types.ConfigCreateResponse, error) {
	return types.ConfigCreateResponse{}, nil
}

func (f fakeConfigAPIClientList) ConfigRemove(_ context.Context, _ string) error {
	return nil
}

func (f fakeConfigAPIClientList) ConfigInspectWithRaw(_ context.Context, _ string) (swarm.Config, []byte, error) {
	return swarm.Config{}, nil, nil
}

func (f fakeConfigAPIClientList) ConfigUpdate(_ context.Context, _ string, _ swarm.Version, _ swarm.ConfigSpec) error {
	return nil
}

// TestSetConfigsWithCredSpecAndConfigs tests that the setConfigs function for
// create correctly looks up the right configs, and correctly handles the
// credentialSpec
func TestSetConfigsWithCredSpecAndConfigs(t *testing.T) {
	// we can't directly access the internal fields of the ConfigOpt struct, so
	// we need to let it do the parsing
	configOpt := &cliopts.ConfigOpt{}
	configOpt.Set("bar")
	opts := &serviceOptions{
		credentialSpec: credentialSpecOpt{
			value: &swarm.CredentialSpec{
				Config: "foo",
			},
			source: "config://foo",
		},
		configs: *configOpt,
	}

	// create a service spec. we need to be sure to fill in the nullable
	// fields, like the code expects
	service := &swarm.ServiceSpec{
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Privileges: &swarm.Privileges{
					CredentialSpec: opts.credentialSpec.value,
				},
			},
		},
	}

	// set up a function to use as the list function
	var fakeClient fakeConfigAPIClientList = func(_ context.Context, opts types.ConfigListOptions) ([]swarm.Config, error) {
		f := opts.Filters

		// we're expecting the filter to have names "foo" and "bar"
		names := f.Get("name")
		assert.Equal(t, len(names), 2)
		assert.Assert(t, is.Contains(names, "foo"))
		assert.Assert(t, is.Contains(names, "bar"))

		return []swarm.Config{
			{
				ID: "fooID",
				Spec: swarm.ConfigSpec{
					Annotations: swarm.Annotations{
						Name: "foo",
					},
				},
			}, {
				ID: "barID",
				Spec: swarm.ConfigSpec{
					Annotations: swarm.Annotations{
						Name: "bar",
					},
				},
			},
		}, nil
	}

	// now call setConfigs
	err := setConfigs(fakeClient, service, opts)
	// verify no error is returned
	assert.NilError(t, err)

	credSpecConfigValue := service.TaskTemplate.ContainerSpec.Privileges.CredentialSpec.Config
	assert.Equal(t, credSpecConfigValue, "fooID")

	configRefs := service.TaskTemplate.ContainerSpec.Configs
	assert.Assert(t, is.Contains(configRefs, &swarm.ConfigReference{
		ConfigID:   "fooID",
		ConfigName: "foo",
		Runtime:    &swarm.ConfigReferenceRuntimeTarget{},
	}), "expected configRefs to contain foo config")
	assert.Assert(t, is.Contains(configRefs, &swarm.ConfigReference{
		ConfigID:   "barID",
		ConfigName: "bar",
		File: &swarm.ConfigReferenceFileTarget{
			Name: "bar",
			// these are the default field values
			UID:  "0",
			GID:  "0",
			Mode: 0444,
		},
	}), "expected configRefs to contain bar config")
}

// TestSetConfigsOnlyCredSpec tests that even if a CredentialSpec is the only
// config needed, setConfigs still works
func TestSetConfigsOnlyCredSpec(t *testing.T) {
	opts := &serviceOptions{
		credentialSpec: credentialSpecOpt{
			value: &swarm.CredentialSpec{
				Config: "foo",
			},
			source: "config://foo",
		},
	}

	service := &swarm.ServiceSpec{
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Privileges: &swarm.Privileges{
					CredentialSpec: opts.credentialSpec.value,
				},
			},
		},
	}

	// set up a function to use as the list function
	var fakeClient fakeConfigAPIClientList = func(_ context.Context, opts types.ConfigListOptions) ([]swarm.Config, error) {
		f := opts.Filters

		names := f.Get("name")
		assert.Equal(t, len(names), 1)
		assert.Assert(t, is.Contains(names, "foo"))

		return []swarm.Config{
			{
				ID: "fooID",
				Spec: swarm.ConfigSpec{
					Annotations: swarm.Annotations{
						Name: "foo",
					},
				},
			},
		}, nil
	}

	// now call setConfigs
	err := setConfigs(fakeClient, service, opts)
	// verify no error is returned
	assert.NilError(t, err)

	credSpecConfigValue := service.TaskTemplate.ContainerSpec.Privileges.CredentialSpec.Config
	assert.Equal(t, credSpecConfigValue, "fooID")

	configRefs := service.TaskTemplate.ContainerSpec.Configs
	assert.Assert(t, is.Contains(configRefs, &swarm.ConfigReference{
		ConfigID:   "fooID",
		ConfigName: "foo",
		Runtime:    &swarm.ConfigReferenceRuntimeTarget{},
	}))
}

// TestSetConfigsOnlyConfigs verifies setConfigs when only configs (and not a
// CredentialSpec) is needed.
func TestSetConfigsOnlyConfigs(t *testing.T) {
	configOpt := &cliopts.ConfigOpt{}
	configOpt.Set("bar")
	opts := &serviceOptions{
		configs: *configOpt,
	}

	service := &swarm.ServiceSpec{
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{},
		},
	}

	var fakeClient fakeConfigAPIClientList = func(_ context.Context, opts types.ConfigListOptions) ([]swarm.Config, error) {
		f := opts.Filters

		names := f.Get("name")
		assert.Equal(t, len(names), 1)
		assert.Assert(t, is.Contains(names, "bar"))

		return []swarm.Config{
			{
				ID: "barID",
				Spec: swarm.ConfigSpec{
					Annotations: swarm.Annotations{
						Name: "bar",
					},
				},
			},
		}, nil
	}

	// now call setConfigs
	err := setConfigs(fakeClient, service, opts)
	// verify no error is returned
	assert.NilError(t, err)

	configRefs := service.TaskTemplate.ContainerSpec.Configs
	assert.Assert(t, is.Contains(configRefs, &swarm.ConfigReference{
		ConfigID:   "barID",
		ConfigName: "bar",
		File: &swarm.ConfigReferenceFileTarget{
			Name: "bar",
			// these are the default field values
			UID:  "0",
			GID:  "0",
			Mode: 0444,
		},
	}))
}

// TestSetConfigsNoConfigs checks that setConfigs works when there are no
// configs of any kind needed
func TestSetConfigsNoConfigs(t *testing.T) {
	// add a credentialSpec that isn't a config
	opts := &serviceOptions{
		credentialSpec: credentialSpecOpt{
			value: &swarm.CredentialSpec{
				File: "foo",
			},
			source: "file://foo",
		},
	}
	service := &swarm.ServiceSpec{
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Privileges: &swarm.Privileges{
					CredentialSpec: opts.credentialSpec.value,
				},
			},
		},
	}

	var fakeClient fakeConfigAPIClientList = func(_ context.Context, opts types.ConfigListOptions) ([]swarm.Config, error) {
		// assert false -- we should never call this function
		assert.Assert(t, false, "we should not be listing configs")
		return nil, nil
	}

	err := setConfigs(fakeClient, service, opts)
	assert.NilError(t, err)

	// ensure that the value of the credentialspec has not changed
	assert.Equal(t, service.TaskTemplate.ContainerSpec.Privileges.CredentialSpec.File, "foo")
	assert.Equal(t, service.TaskTemplate.ContainerSpec.Privileges.CredentialSpec.Config, "")
}
