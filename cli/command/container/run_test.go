package container

import (
	"fmt"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/notary"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestRunLabel(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		createContainerFunc: func(_ *container.Config, _ *container.HostConfig, _ *network.NetworkingConfig, _ string) (container.ContainerCreateCreatedBody, error) {
			return container.ContainerCreateCreatedBody{
				ID: "id",
			}, nil
		},
		Version: "1.36",
	})
	cmd := NewRunCommand(cli)
	cmd.Flags().Set("detach", "true")
	cmd.SetArgs([]string{"--label", "foo", "busybox"})
	assert.NilError(t, cmd.Execute())
}

func TestRunCommandWithContentTrustErrors(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		expectedError string
		notaryFunc    test.NotaryClientFuncType
	}{
		{
			name:          "offline-notary-server",
			notaryFunc:    notary.GetOfflineNotaryRepository,
			expectedError: "client is offline",
			args:          []string{"image:tag"},
		},
		{
			name:          "uninitialized-notary-server",
			notaryFunc:    notary.GetUninitializedNotaryRepository,
			expectedError: "remote trust data does not exist",
			args:          []string{"image:tag"},
		},
		{
			name:          "empty-notary-server",
			notaryFunc:    notary.GetEmptyTargetsNotaryRepository,
			expectedError: "No valid trust data for tag",
			args:          []string{"image:tag"},
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			createContainerFunc: func(config *container.Config,
				hostConfig *container.HostConfig,
				networkingConfig *network.NetworkingConfig,
				containerName string,
			) (container.ContainerCreateCreatedBody, error) {
				return container.ContainerCreateCreatedBody{}, fmt.Errorf("shouldn't try to pull image")
			},
		}, test.EnableContentTrust)
		cli.SetNotaryClient(tc.notaryFunc)
		cmd := NewRunCommand(cli)
		cmd.SetArgs(tc.args)
		err := cmd.Execute()
		assert.Assert(t, err != nil)
		assert.Assert(t, is.Contains(cli.ErrBuffer().String(), tc.expectedError))
	}
}
