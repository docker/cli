package container

import (
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
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
		flags             map[string]string
		containerListFunc func(container.ListOptions) ([]types.Container, error)
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
			containerListFunc: func(_ container.ListOptions) ([]types.Container, error) {
				return nil, errors.New("error listing containers")
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
		for key, value := range tc.flags {
			assert.Check(t, cmd.Flags().Set(key, value))
		}
		cmd.SetArgs([]string{})
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestContainerListWithoutFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerListFunc: func(_ container.ListOptions) ([]types.Container, error) {
			return []types.Container{
				*builders.Container("c1"),
				*builders.Container("c2", builders.WithName("foo")),
				*builders.Container("c3", builders.WithPort(80, 80, builders.TCP), builders.WithPort(81, 81, builders.TCP), builders.WithPort(82, 82, builders.TCP)),
				*builders.Container("c4", builders.WithPort(81, 81, builders.UDP)),
				*builders.Container("c5", builders.WithPort(82, 82, builders.IP("8.8.8.8"), builders.TCP)),
			}, nil
		},
	})
	cmd := newListCommand(cli)
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "container-list-without-format.golden")
}

func TestContainerListNoTrunc(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerListFunc: func(_ container.ListOptions) ([]types.Container, error) {
			return []types.Container{
				*builders.Container("c1"),
				*builders.Container("c2", builders.WithName("foo/bar")),
			}, nil
		},
	})
	cmd := newListCommand(cli)
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	assert.Check(t, cmd.Flags().Set("no-trunc", "true"))
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "container-list-without-format-no-trunc.golden")
}

// Test for GitHub issue docker/docker#21772
func TestContainerListNamesMultipleTime(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerListFunc: func(_ container.ListOptions) ([]types.Container, error) {
			return []types.Container{
				*builders.Container("c1"),
				*builders.Container("c2", builders.WithName("foo/bar")),
			}, nil
		},
	})
	cmd := newListCommand(cli)
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	assert.Check(t, cmd.Flags().Set("format", "{{.Names}} {{.Names}}"))
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "container-list-format-name-name.golden")
}

// Test for GitHub issue docker/docker#30291
func TestContainerListFormatTemplateWithArg(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerListFunc: func(_ container.ListOptions) ([]types.Container, error) {
			return []types.Container{
				*builders.Container("c1", builders.WithLabel("some.label", "value")),
				*builders.Container("c2", builders.WithName("foo/bar"), builders.WithLabel("foo", "bar")),
			}, nil
		},
	})
	cmd := newListCommand(cli)
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	assert.Check(t, cmd.Flags().Set("format", `{{.Names}} {{.Label "some.label"}}`))
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "container-list-format-with-arg.golden")
}

func TestContainerListFormatSizeSetsOption(t *testing.T) {
	tests := []struct {
		doc, format, sizeFlag string
		sizeExpected          bool
	}{
		{
			doc:          "detect with all fields",
			format:       `{{json .}}`,
			sizeExpected: true,
		},
		{
			doc:          "detect with explicit field",
			format:       `{{.Size}}`,
			sizeExpected: true,
		},
		{
			doc:          "detect no size",
			format:       `{{.Names}}`,
			sizeExpected: false,
		},
		{
			doc:          "override enable",
			format:       `{{.Names}}`,
			sizeFlag:     "true",
			sizeExpected: true,
		},
		{
			doc:          "override disable",
			format:       `{{.Size}}`,
			sizeFlag:     "false",
			sizeExpected: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.doc, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				containerListFunc: func(options container.ListOptions) ([]types.Container, error) {
					assert.Check(t, is.Equal(options.Size, tc.sizeExpected))
					return []types.Container{}, nil
				},
			})
			cmd := newListCommand(cli)
			cmd.SetArgs([]string{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			assert.Check(t, cmd.Flags().Set("format", tc.format))
			if tc.sizeFlag != "" {
				assert.Check(t, cmd.Flags().Set("size", tc.sizeFlag))
			}
			assert.NilError(t, cmd.Execute())
		})
	}
}

func TestContainerListWithConfigFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerListFunc: func(_ container.ListOptions) ([]types.Container, error) {
			return []types.Container{
				*builders.Container("c1", builders.WithLabel("some.label", "value"), builders.WithSize(10700000)),
				*builders.Container("c2", builders.WithName("foo/bar"), builders.WithLabel("foo", "bar"), builders.WithSize(3200000)),
			}, nil
		},
	})
	cli.SetConfigFile(&configfile.ConfigFile{
		PsFormat: "{{ .Names }} {{ .Image }} {{ .Labels }} {{ .Size}}",
	})
	cmd := newListCommand(cli)
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "container-list-with-config-format.golden")
}

func TestContainerListWithFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		containerListFunc: func(_ container.ListOptions) ([]types.Container, error) {
			return []types.Container{
				*builders.Container("c1", builders.WithLabel("some.label", "value")),
				*builders.Container("c2", builders.WithName("foo/bar"), builders.WithLabel("foo", "bar")),
			}, nil
		},
	})

	t.Run("with format", func(t *testing.T) {
		cli.OutBuffer().Reset()
		cmd := newListCommand(cli)
		cmd.SetArgs([]string{})
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.Check(t, cmd.Flags().Set("format", "{{ .Names }} {{ .Image }} {{ .Labels }}"))
		assert.NilError(t, cmd.Execute())
		golden.Assert(t, cli.OutBuffer().String(), "container-list-with-format.golden")
	})

	t.Run("with format and quiet", func(t *testing.T) {
		cli.OutBuffer().Reset()
		cmd := newListCommand(cli)
		cmd.SetArgs([]string{})
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		assert.Check(t, cmd.Flags().Set("format", "{{ .Names }} {{ .Image }} {{ .Labels }}"))
		assert.Check(t, cmd.Flags().Set("quiet", "true"))
		assert.NilError(t, cmd.Execute())
		assert.Equal(t, cli.ErrBuffer().String(), "WARNING: Ignoring custom format, because both --format and --quiet are set.\n")
		golden.Assert(t, cli.OutBuffer().String(), "container-list-quiet.golden")
	})
}
