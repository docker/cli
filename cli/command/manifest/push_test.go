package manifest

import (
	"context"
	"io/ioutil"
	"testing"

	manifesttypes "github.com/docker/cli/cli/manifest/types"
	"github.com/docker/cli/internal/test"
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
)

func newFakeRegistryClient() *fakeRegistryClient {
	return &fakeRegistryClient{
		getManifestFunc: func(_ context.Context, _ reference.Named) (manifesttypes.ImageManifest, error) {
			return manifesttypes.ImageManifest{}, errors.New("getManifestFunc not implemented")
		},
		getManifestListFunc: func(_ context.Context, _ reference.Named) ([]manifesttypes.ImageManifest, error) {
			return nil, errors.Errorf("getManifestListFunc not implemented")
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

func TestPushFromYaml(t *testing.T) {
	cli := test.NewFakeCli(nil)
	cli.SetRegistryClient(&fakeRegistryClient{
		getManifestFunc: func(_ context.Context, ref reference.Named) (manifesttypes.ImageManifest, error) {
			return fullImageManifest(t, ref), nil
		},
	})

	cmd := newPushListCommand(cli)
	cmd.Flags().Set("file", "testdata/test-push.yaml")
	cmd.SetArgs([]string{"pushtest/pass:latest"})
	assert.NilError(t, cmd.Execute())
}

func TestManifestPushYamlErrors(t *testing.T) {
	testCases := []struct {
		flags         map[string]string
		args          []string
		expectedError string
	}{
		{
			flags:         map[string]string{"file": "testdata/test-push-fail.yaml"},
			args:          []string{"pushtest/fail:latest"},
			expectedError: "manifest entry for image has unsupported os/arch combination: linux/nope",
		},
		{
			flags:         map[string]string{"file": "testdata/test-push-empty.yaml"},
			args:          []string{"pushtest/fail:latest"},
			expectedError: "no manifests specified in file input",
		},
		{
			args:          []string{"testdata/test-push-empty.yaml"},
			expectedError: "No such manifest: docker.io/testdata/test-push-empty.yaml:latest",
		},
	}

	store, sCleanup := newTempManifestStore(t)
	defer sCleanup()
	for _, tc := range testCases {
		cli := test.NewFakeCli(nil)
		cli.SetRegistryClient(&fakeRegistryClient{
			getManifestFunc: func(_ context.Context, ref reference.Named) (manifesttypes.ImageManifest, error) {
				return fullImageManifest(t, ref), nil
			},
		})

		cli.SetManifestStore(store)
		cmd := newPushListCommand(cli)
		for k, v := range tc.flags {
			cmd.Flags().Set(k, v)
		}
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}
