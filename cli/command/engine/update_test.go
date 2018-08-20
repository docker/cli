package engine

import (
	"fmt"
	"testing"

	"github.com/docker/cli/internal/containerizedengine"
	"gotest.tools/assert"
)

func TestUpdateNoContainerd(t *testing.T) {
	testCli.SetContainerizedEngineClient(
		func(string) (containerizedengine.Client, error) {
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
		func(string) (containerizedengine.Client, error) {
			return &fakeContainerizedEngineClient{}, nil
		},
	)
	cmd := newUpdateCommand(testCli)
	cmd.Flags().Set("registry-prefix", "docker.io/docker")
	cmd.Flags().Set("version", "someversion")
	err := cmd.Execute()
	assert.NilError(t, err)
}
