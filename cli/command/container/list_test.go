package container

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	. "github.com/docker/cli/internal/test/builders" // Import builders to get the builder function as package function
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

func TestContainerListBuildContainerListOptions(t *testing.T) {
	filters := opts.NewFilterOpt()
	assert.NilError(t, filters.Set("foo=bar"))
	assert.NilError(t, filters.Set("baz=foo"))

	contexts := []struct {
		psOpts          *psOptions
		expectedAll     bool
		expectedSize    bool
		expectedLimit   int
		expectedFilters map[string]string
	}{
		{
			psOpts: &psOptions{
				all:    true,
				size:   true,
				last:   5,
				filter: filters,
			},
			expectedAll:   true,
			expectedSize:  true,
			expectedLimit: 5,
			expectedFilters: map[string]string{
				"foo": "bar",
				"baz": "foo",
			},
		},
		{
			psOpts: &psOptions{
				all:     true,
				size:    true,
				last:    -1,
				nLatest: true,
			},
			expectedAll:     true,
			expectedSize:    true,
			expectedLimit:   1,
			expectedFilters: make(map[string]string),
		},
		{
			psOpts: &psOptions{
				all:    true,
				size:   false,
				last:   5,
				filter: filters,
				// With .Size, size should be true
				format: "{{.Size}}",
			},
			expectedAll:   true,
			expectedSize:  true,
			expectedLimit: 5,
			expectedFilters: map[string]string{
				"foo": "bar",
				"baz": "foo",
			},
		},
		{
			psOpts: &psOptions{
				all:    true,
				size:   false,
				last:   5,
				filter: filters,
				// With .Size, size should be true
				format: "{{.Size}} {{.CreatedAt}} {{upper .Networks}}",
			},
			expectedAll:   true,
			expectedSize:  true,
			expectedLimit: 5,
			expectedFilters: map[string]string{
				"foo": "bar",
				"baz": "foo",
			},
		},
		{
			psOpts: &psOptions{
				all:    true,
				size:   false,
				last:   5,
				filter: filters,
				// Without .Size, size should be false
				format: "{{.CreatedAt}} {{.Networks}}",
			},
			expectedAll:   true,
			expectedSize:  false,
			expectedLimit: 5,
			expectedFilters: map[string]string{
				"foo": "bar",
				"baz": "foo",
			},
		},
	}

	for _, c := range contexts {
		options, err := buildContainerListOptions(c.psOpts)
		assert.NilError(t, err)

		assert.Check(t, is.Equal(c.expectedAll, options.All))
		assert.Check(t, is.Equal(c.expectedSize, options.Size))
		assert.Check(t, is.Equal(c.expectedLimit, options.Limit))
		assert.Check(t, is.Equal(len(c.expectedFilters), options.Filters.Len()))

		for k, v := range c.expectedFilters {
			f := options.Filters
			if !f.ExactMatch(k, v) {
				t.Fatalf("Expected filter with key %s to be %s but got %s", k, v, f.Get(k))
			}
		}
	}
}

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
		cmd.SetOut(ioutil.Discard)
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
