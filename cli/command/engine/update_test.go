package engine

import (
	"fmt"
	"testing"

	"github.com/docker/cli/internal/test"
	clitypes "github.com/docker/cli/types"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"gotest.tools/assert"
)

func TestUpdateNoContainerd(t *testing.T) {
	testCli.SetContainerizedEngineClient(
		func(string) (clitypes.ContainerizedClient, error) {
			return nil, fmt.Errorf("some error")
		},
	)
	cmd := newUpdateCommand(testCli)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	assert.ErrorContains(t, err, "unable to access local containerd")
}

func TestUpdateHappy(t *testing.T) {
	c := test.NewFakeCli(&verClient{client.Client{}, types.Version{Version: "1.1.0"}, nil, types.Info{ServerVersion: "1.1.0"}, nil})
	c.SetContainerizedEngineClient(
		func(string) (clitypes.ContainerizedClient, error) {
			return &fakeContainerizedEngineClient{}, nil
		},
	)
	cmd := newUpdateCommand(c)
	cmd.Flags().Set("registry-prefix", clitypes.RegistryPrefix)
	cmd.Flags().Set("version", "someversion")
	cmd.Flags().Set("engine-image", "someimage")
	err := cmd.Execute()
	assert.NilError(t, err)
}
