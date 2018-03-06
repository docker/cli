package container

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	// Import builders to get the builder function as package function
	. "github.com/docker/cli/internal/test/builders"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/golden"
)

func TestContainerListErrors(t *testing.T) {
	testCases := []struct {
		args              []string
		flags             map[string]string
		containerListFunc func(types.ContainerListOptions) ([]types.Container, error)
		expectedError     string
	}{
		{
			flags: map[string]string{
				"format": "{{invalid}}",
			},
			expectedError: `function "invalid" not defined`,
		},
		{
			flags: map[string]string{
				"format": "{{join}}",
			},
			expectedError: `wrong number of args for join`,
		},
		{
			containerListFunc: func(_ types.ContainerListOptions) ([]types.Container, error) {
				return nil, fmt.Errorf("error listing containers")
			},
			expectedError: "error listing containers",
		},
	}
	for _, tc := range testCases {
		cmd := newListCommand(
			test.NewFakeCli(&fakeClient{
				containerListFunc: tc.containerListFunc,
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

func TestContainerListWithoutFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerListFunc: func(_ types.ContainerListOptions) ([]types.Container, error) {
			return []types.Container{
				*Container("c1"),
				*Container("c2", WithName("foo")),
				*Container("c3", WithPort(80, 80, TCP), WithPort(81, 81, TCP), WithPort(82, 82, TCP)),
				*Container("c4", WithPort(81, 81, UDP)),
				*Container("c5", WithPort(82, 82, IP("8.8.8.8"), TCP)),
			}, nil
		},
	})
	cmd := newListCommand(cli)
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "container-list-without-format.golden")
}

func TestContainerListNoTrunc(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerListFunc: func(_ types.ContainerListOptions) ([]types.Container, error) {
			return []types.Container{
				*Container("c1"),
				*Container("c2", WithName("foo/bar")),
			}, nil
		},
	})
	cmd := newListCommand(cli)
	cmd.Flags().Set("no-trunc", "true")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "container-list-without-format-no-trunc.golden")
}

// Test for GitHub issue docker/docker#21772
func TestContainerListNamesMultipleTime(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerListFunc: func(_ types.ContainerListOptions) ([]types.Container, error) {
			return []types.Container{
				*Container("c1"),
				*Container("c2", WithName("foo/bar")),
			}, nil
		},
	})
	cmd := newListCommand(cli)
	cmd.Flags().Set("format", "{{.Names}} {{.Names}}")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "container-list-format-name-name.golden")
}

// Test for GitHub issue docker/docker#30291
func TestContainerListFormatTemplateWithArg(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerListFunc: func(_ types.ContainerListOptions) ([]types.Container, error) {
			return []types.Container{
				*Container("c1", WithLabel("some.label", "value")),
				*Container("c2", WithName("foo/bar"), WithLabel("foo", "bar")),
			}, nil
		},
	})
	cmd := newListCommand(cli)
	cmd.Flags().Set("format", `{{.Names}} {{.Label "some.label"}}`)
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "container-list-format-with-arg.golden")
}

func TestContainerListFormatSizeSetsOption(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerListFunc: func(options types.ContainerListOptions) ([]types.Container, error) {
			assert.Check(t, options.Size)
			return []types.Container{}, nil
		},
	})
	cmd := newListCommand(cli)
	cmd.Flags().Set("format", `{{.Size}}`)
	assert.NilError(t, cmd.Execute())
}

func TestContainerListWithConfigFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerListFunc: func(_ types.ContainerListOptions) ([]types.Container, error) {
			return []types.Container{
				*Container("c1", WithLabel("some.label", "value")),
				*Container("c2", WithName("foo/bar"), WithLabel("foo", "bar")),
			}, nil
		},
	})
	cli.SetConfigFile(&configfile.ConfigFile{
		PsFormat: "{{ .Names }} {{ .Image }} {{ .Labels }}",
	})
	cmd := newListCommand(cli)
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "container-list-with-config-format.golden")
}

func TestContainerListWithFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerListFunc: func(_ types.ContainerListOptions) ([]types.Container, error) {
			return []types.Container{
				*Container("c1", WithLabel("some.label", "value")),
				*Container("c2", WithName("foo/bar"), WithLabel("foo", "bar")),
			}, nil
		},
	})
	cmd := newListCommand(cli)
	cmd.Flags().Set("format", "{{ .Names }} {{ .Image }} {{ .Labels }}")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "container-list-with-format.golden")
}
