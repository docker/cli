package engine

import (
	"fmt"
	"testing"

	"github.com/docker/cli/internal/containerizedengine"
	"gotest.tools/assert"
)

func TestInitNoContainerd(t *testing.T) {
	testCli.SetContainerizedEngineClient(
		func(string) (containerizedengine.Client, error) {
			return nil, fmt.Errorf("some error")
		},
	)
	cmd := newInitCommand(testCli)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	assert.ErrorContains(t, err, "unable to access local containerd")
}

func TestInitHappy(t *testing.T) {
	testCli.SetContainerizedEngineClient(
		func(string) (containerizedengine.Client, error) {
			return &fakeContainerizedEngineClient{}, nil
		},
	)
	cmd := newInitCommand(testCli)
	err := cmd.Execute()
	assert.NilError(t, err)
}
