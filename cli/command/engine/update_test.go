package engine

import (
	"fmt"
	"testing"

	clitypes "github.com/docker/cli/types"
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
	testCli.SetContainerizedEngineClient(
		func(string) (clitypes.ContainerizedClient, error) {
			return &fakeContainerizedEngineClient{}, nil
		},
	)
	cmd := newUpdateCommand(testCli)
	cmd.Flags().Set("registry-prefix", "docker.io/store/docker")
	cmd.Flags().Set("version", "someversion")
	err := cmd.Execute()
	assert.NilError(t, err)
}
