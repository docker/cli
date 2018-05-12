package manifest

import (
	"context"
	"io/ioutil"
	"testing"

	manifesttypes "github.com/docker/cli/cli/manifest/types"
	"github.com/docker/cli/internal/test"
	"github.com/docker/distribution/reference"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/pkg/errors"
)

func newFakeRegistryClient() *fakeRegistryClient {
	return &fakeRegistryClient{
		getManifestFunc: func(_ context.Context, _ reference.Named) (manifesttypes.ImageManifest, error) {
			return manifesttypes.ImageManifest{}, errors.New("")
		},
		getManifestListFunc: func(_ context.Context, _ reference.Named) ([]manifesttypes.ImageManifest, error) {
			return nil, errors.Errorf("")
		},
	}
}

func TestManifestPushErrors(t *testing.T) {
	testCases := []struct {
		args          []string
		expectedError string
	}{
		{
			args:          []string{"one-arg", "extra-arg"},
			expectedError: "requires exactly 1 argument",
		},
		{
			args:          []string{"th!si'sa/fa!ke/li$t/-name"},
			expectedError: "invalid reference format",
		},
	}

	for _, tc := range testCases {
		cli := test.NewFakeCli(nil)
		cmd := newPushListCommand(cli)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestManifestPush(t *testing.T) {
	store, sCleanup := newTempManifestStore(t)
	defer sCleanup()

	registry := newFakeRegistryClient()

	cli := test.NewFakeCli(nil)
	cli.SetManifestStore(store)
	cli.SetRegistryClient(registry)

	namedRef := ref(t, "alpine:3.0")
	imageManifest := fullImageManifest(t, namedRef)
	err := store.Save(ref(t, "list:v1"), namedRef, imageManifest)
	assert.NilError(t, err)

	cmd := newPushListCommand(cli)
	cmd.SetArgs([]string{"example.com/list:v1"})
	err = cmd.Execute()
	assert.NilError(t, err)
}
