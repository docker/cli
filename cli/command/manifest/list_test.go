package manifest

import (
	"io"
	"testing"

	"github.com/docker/cli/cli/manifest/store"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestListErrors(t *testing.T) {
	manifestStore := store.NewStore(t.TempDir())

	testCases := []struct {
		description   string
		args          []string
		flags         map[string]string
		expectedError string
	}{
		{
			description:   "too many arguments",
			args:          []string{"foo"},
			expectedError: "accepts no arguments",
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
		cli := test.NewFakeCli(nil)
		cli.SetManifestStore(manifestStore)
		cmd := newListCommand(cli)
		cmd.SetArgs(tc.args)
		for key, value := range tc.flags {
			cmd.Flags().Set(key, value)
		}
		cmd.SetOut(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestList(t *testing.T) {
	manifestStore := store.NewStore(t.TempDir())

	list1 := ref(t, "first:1")
	namedRef := ref(t, "alpine:3.0")
	err := manifestStore.Save(list1, namedRef, fullImageManifest(t, namedRef))
	assert.NilError(t, err)
	namedRef = ref(t, "alpine:3.1")
	err = manifestStore.Save(list1, namedRef, fullImageManifest(t, namedRef))
	assert.NilError(t, err)

	list2 := ref(t, "second:2")
	namedRef = ref(t, "alpine:3.2")
	err = manifestStore.Save(list2, namedRef, fullImageManifest(t, namedRef))
	assert.NilError(t, err)

	testCases := []struct {
		description string
		args        []string
		flags       map[string]string
		golden      string
		listFunc    func(filter filters.Args) (types.PluginsListResponse, error)
	}{
		{
			description: "list with no additional flags",
			args:        []string{},
			golden:      "manifest-list.golden",
		},
		{
			description: "list with quiet option",
			args:        []string{},
			flags: map[string]string{
				"quiet": "true",
			},
			golden: "manifest-list-with-quiet-option.golden",
		},
	}

	for _, tc := range testCases {
		cli := test.NewFakeCli(nil)
		cli.SetManifestStore(manifestStore)
		cmd := newListCommand(cli)
		cmd.SetArgs(tc.args)
		for key, value := range tc.flags {
			cmd.Flags().Set(key, value)
		}
		assert.NilError(t, cmd.Execute())
		golden.Assert(t, cli.OutBuffer().String(), tc.golden)
	}
}
