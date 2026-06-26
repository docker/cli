package plugin

import (
	"errors"
	"io"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/moby/moby/api/types/plugin"
	"github.com/moby/moby/client"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestListErrors(t *testing.T) {
	testCases := []struct {
		description   string
		args          []string
		flags         map[string]string
		expectedError string
		listFunc      func(client.PluginListOptions) (client.PluginListResult, error)
	}{
		{
			description:   "too many arguments",
			args:          []string{"foo"},
			expectedError: "accepts no arguments",
		},
		{
			description:   "error listing plugins",
			args:          []string{},
			expectedError: "error listing plugins",
			listFunc: func(client.PluginListOptions) (client.PluginListResult, error) {
				return client.PluginListResult{}, errors.New("error listing plugins")
			},
		},
		{
			description: "invalid format",
			args:        []string{},
			flags: map[string]string{
				"format": "{{invalid format}}",
			},
			expectedError: "template parsing error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{pluginListFunc: tc.listFunc})
			cmd := newListCommand(cli)
			cmd.SetArgs(tc.args)
			for key, value := range tc.flags {
				assert.NilError(t, cmd.Flags().Set(key, value))
			}
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
		})
	}
}

func TestList(t *testing.T) {
	singlePluginListFunc := func(client.PluginListOptions) (client.PluginListResult, error) {
		return client.PluginListResult{
			Items: plugin.ListResponse{
				{
					ID:      "id-foo",
					Name:    "name-foo",
					Enabled: true,
					Config: plugin.Config{
						Description: "desc-bar",
					},
				},
			},
		}, nil
	}

	testCases := []struct {
		description string
		args        []string
		flags       map[string]string
		golden      string
		listFunc    func(client.PluginListOptions) (client.PluginListResult, error)
	}{
		{
			description: "list with no additional flags",
			args:        []string{},
			golden:      "plugin-list-without-format.golden",
			listFunc:    singlePluginListFunc,
		},
		{
			description: "list with filters",
			args:        []string{},
			flags: map[string]string{
				"filter": "foo=bar",
			},
			golden: "plugin-list-without-format.golden",
			listFunc: func(opts client.PluginListOptions) (client.PluginListResult, error) {
				assert.Check(t, opts.Filters["foo"]["bar"])
				return singlePluginListFunc(opts)
			},
		},
		{
			description: "list with quiet option",
			args:        []string{},
			flags: map[string]string{
				"quiet": "true",
			},
			golden:   "plugin-list-with-quiet-option.golden",
			listFunc: singlePluginListFunc,
		},
		{
			description: "list with no-trunc option",
			args:        []string{},
			flags: map[string]string{
				"no-trunc": "true",
				"format":   "{{ .ID }}",
			},
			golden: "plugin-list-with-no-trunc-option.golden",
			listFunc: func(opts client.PluginListOptions) (client.PluginListResult, error) {
				return client.PluginListResult{
					Items: []plugin.Plugin{{
						ID:      "xyg4z2hiSLO5yTnBJfg4OYia9gKA6Qjd",
						Name:    "name-foo",
						Enabled: true,
						Config: plugin.Config{
							Description: "desc-bar",
						},
					}},
				}, nil
			},
		},
		{
			description: "list with format",
			args:        []string{},
			flags: map[string]string{
				"format": "{{ .Name }}",
			},
			golden:   "plugin-list-with-format.golden",
			listFunc: singlePluginListFunc,
		},
		{
			description: "list output is sorted based on plugin name",
			args:        []string{},
			flags: map[string]string{
				"format": "{{ .Name }}",
			},
			golden: "plugin-list-sort.golden",
			listFunc: func(client.PluginListOptions) (client.PluginListResult, error) {
				return client.PluginListResult{
					Items: []plugin.Plugin{
						{
							ID:   "id-1",
							Name: "plugin-1-foo",
						},
						{
							ID:   "id-2",
							Name: "plugin-10-foo",
						},
						{
							ID:   "id-3",
							Name: "plugin-2-foo",
						},
					},
				}, nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{pluginListFunc: tc.listFunc})
			cmd := newListCommand(cli)
			cmd.SetArgs(tc.args)
			for key, value := range tc.flags {
				assert.NilError(t, cmd.Flags().Set(key, value))
			}
			assert.NilError(t, cmd.Execute())
			golden.Assert(t, cli.OutBuffer().String(), tc.golden)
		})
	}
}
