package manifest

import (
	"io/ioutil"
	"testing"

	manifesttypes "github.com/docker/cli/cli/manifest/types"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/docker/distribution/reference"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestManifestCreateErrors(t *testing.T) {
	testCases := []struct {
		args          []string
		expectedError string
	}{
		{
			args:          []string{"too-few-arguments"},
			expectedError: "requires at least 2 arguments",
		},
		{
			args:          []string{"th!si'sa/fa!ke/li$t/name", "example.com/alpine:3.0"},
			expectedError: "error parsing name for manifest list",
		},
	}

	for _, tc := range testCases {
		cli := test.NewFakeCli(nil)
		cmd := newCreateListCommand(cli)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

// create a manifest list, then overwrite it, and inspect to see if the old one is still there
func TestManifestCreateAmend(t *testing.T) {
	store, cleanup := newTempManifestStore(t)
	defer cleanup()

	cli := test.NewFakeCli(nil)
	cli.SetManifestStore(store)

	namedRef := ref(t, "alpine:3.0")
	imageManifest := fullImageManifest(t, namedRef)
	err := store.Save(ref(t, "list:v1"), namedRef, imageManifest)
	require.NoError(t, err)
	namedRef = ref(t, "alpine:3.1")
	imageManifest = fullImageManifest(t, namedRef)
	err = store.Save(ref(t, "list:v1"), namedRef, imageManifest)
	require.NoError(t, err)

	cmd := newCreateListCommand(cli)
	cmd.SetArgs([]string{"example.com/list:v1", "example.com/alpine:3.1"})
	cmd.Flags().Set("amend", "true")
	cmd.SetOutput(ioutil.Discard)
	err = cmd.Execute()
	require.NoError(t, err)

	// make a new cli to clear the buffers
	cli = test.NewFakeCli(nil)
	cli.SetManifestStore(store)
	inspectCmd := newInspectCommand(cli)
	inspectCmd.SetArgs([]string{"example.com/list:v1"})
	require.NoError(t, inspectCmd.Execute())
	actual := cli.OutBuffer()
	expected := golden.Get(t, "inspect-manifest-list.golden")
	assert.Equal(t, string(expected), actual.String())
}

// attempt to overwrite a saved manifest and get refused
func TestManifestCreateRefuseAmend(t *testing.T) {
	store, cleanup := newTempManifestStore(t)
	defer cleanup()

	cli := test.NewFakeCli(nil)
	cli.SetManifestStore(store)
	namedRef := ref(t, "alpine:3.0")
	imageManifest := fullImageManifest(t, namedRef)
	err := store.Save(ref(t, "list:v1"), namedRef, imageManifest)
	require.NoError(t, err)

	cmd := newCreateListCommand(cli)
	cmd.SetArgs([]string{"example.com/list:v1", "example.com/alpine:3.0"})
	cmd.SetOutput(ioutil.Discard)
	err = cmd.Execute()
	assert.EqualError(t, err, "refusing to amend an existing manifest list with no --amend flag")
}

// attempt to make a manifest list without valid images
func TestManifestCreateNoManifest(t *testing.T) {
	store, cleanup := newTempManifestStore(t)
	defer cleanup()

	cli := test.NewFakeCli(nil)
	cli.SetManifestStore(store)
	cli.SetRegistryClient(&fakeRegistryClient{
		getManifestFunc: func(_ context.Context, ref reference.Named) (manifesttypes.ImageManifest, error) {
			return manifesttypes.ImageManifest{}, errors.Errorf("No such image: %v", ref)
		},
		getManifestListFunc: func(ctx context.Context, ref reference.Named) ([]manifesttypes.ImageManifest, error) {
			return nil, errors.Errorf("No such manifest: %s", ref)
		},
	})

	cmd := newCreateListCommand(cli)
	cmd.SetArgs([]string{"example.com/list:v1", "example.com/alpine:3.0"})
	cmd.SetOutput(ioutil.Discard)
	err := cmd.Execute()
	assert.EqualError(t, err, "No such image: example.com/alpine:3.0")
}
