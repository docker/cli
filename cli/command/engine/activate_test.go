package engine

import (
	"fmt"
	"testing"

	"github.com/docker/cli/types"
	"gotest.tools/assert"
)

func TestActivateNoContainerd(t *testing.T) {
	testCli.SetContainerizedEngineClient(
		func(string) (types.ContainerizedClient, error) {
			return nil, fmt.Errorf("some error")
		},
	)
	cmd := newActivateCommand(testCli)
	cmd.Flags().Set("license", "invalidpath")
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	assert.ErrorContains(t, err, "unable to access local containerd")
}

func TestActivateBadLicense(t *testing.T) {
	testCli.SetContainerizedEngineClient(
		func(string) (types.ContainerizedClient, error) {
			return &fakeContainerizedEngineClient{}, nil
		},
	)
	cmd := newActivateCommand(testCli)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.Flags().Set("license", "invalidpath")
	err := cmd.Execute()
	assert.Error(t, err, "open invalidpath: no such file or directory")
}
