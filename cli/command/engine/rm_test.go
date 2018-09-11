package engine

import (
	"fmt"
	"testing"

	clitypes "github.com/docker/cli/types"
	"gotest.tools/assert"
)

func TestRmNoContainerd(t *testing.T) {
	testCli.SetContainerizedEngineClient(
		func(string) (clitypes.ContainerizedClient, error) {
			return nil, fmt.Errorf("some error")
		},
	)
	cmd := newRmCommand(testCli)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	assert.ErrorContains(t, err, "unable to access local containerd")
}

func TestRmHappy(t *testing.T) {
	testCli.SetContainerizedEngineClient(
		func(string) (clitypes.ContainerizedClient, error) {
			return &fakeContainerizedEngineClient{}, nil
		},
	)
	cmd := newRmCommand(testCli)
	err := cmd.Execute()
	assert.NilError(t, err)
}
