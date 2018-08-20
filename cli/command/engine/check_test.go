package engine

import (
	"context"
	"fmt"
	"testing"

	registryclient "github.com/docker/cli/cli/registry/client"
	"github.com/docker/cli/internal/containerizedengine"
	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/client"
	ver "github.com/hashicorp/go-version"
	"gotest.tools/assert"
	"gotest.tools/golden"
)

var (
	testCli = test.NewFakeCli(&client.Client{})
)

func TestCheckForUpdatesNoContainerd(t *testing.T) {
	testCli.SetContainerizedEngineClient(
		func(string) (containerizedengine.Client, error) {
			return nil, fmt.Errorf("some error")
		},
	)
	cmd := newCheckForUpdatesCommand(testCli)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	assert.ErrorContains(t, err, "unable to access local containerd")
}

func TestCheckForUpdatesNoCurrentVersion(t *testing.T) {
	retErr := fmt.Errorf("some failure")
	getCurrentEngineVersionFunc := func(ctx context.Context) (containerizedengine.EngineInitOptions, error) {
		return containerizedengine.EngineInitOptions{}, retErr
	}
	testCli.SetContainerizedEngineClient(
		func(string) (containerizedengine.Client, error) {
			return &fakeContainerizedEngineClient{
				getCurrentEngineVersionFunc: getCurrentEngineVersionFunc,
			}, nil
		},
	)
	cmd := newCheckForUpdatesCommand(testCli)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	assert.Assert(t, err == retErr)
}

func TestCheckForUpdatesGetEngineVersionsFail(t *testing.T) {
	retErr := fmt.Errorf("some failure")
	getEngineVersionsFunc := func(ctx context.Context,
		registryClient registryclient.RegistryClient,
		currentVersion, imageName string) (containerizedengine.AvailableVersions, error) {
		return containerizedengine.AvailableVersions{}, retErr
	}
	testCli.SetContainerizedEngineClient(
		func(string) (containerizedengine.Client, error) {
			return &fakeContainerizedEngineClient{
				getEngineVersionsFunc: getEngineVersionsFunc,
			}, nil
		},
	)
	cmd := newCheckForUpdatesCommand(testCli)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	assert.Assert(t, err == retErr)
}

func TestCheckForUpdatesGetEngineVersionsHappy(t *testing.T) {
	getCurrentEngineVersionFunc := func(ctx context.Context) (containerizedengine.EngineInitOptions, error) {
		return containerizedengine.EngineInitOptions{
			EngineImage:   "current engine",
			EngineVersion: "1.1.0",
		}, nil
	}
	getEngineVersionsFunc := func(ctx context.Context,
		registryClient registryclient.RegistryClient,
		currentVersion, imageName string) (containerizedengine.AvailableVersions, error) {
		return containerizedengine.AvailableVersions{
			Downgrades: parseVersions(t, "1.0.1", "1.0.2", "1.0.3-beta1"),
			Patches:    parseVersions(t, "1.1.1", "1.1.2", "1.1.3-beta1"),
			Upgrades:   parseVersions(t, "1.2.0", "2.0.0", "2.1.0-beta1"),
		}, nil
	}
	testCli.SetContainerizedEngineClient(
		func(string) (containerizedengine.Client, error) {
			return &fakeContainerizedEngineClient{
				getEngineVersionsFunc:       getEngineVersionsFunc,
				getCurrentEngineVersionFunc: getCurrentEngineVersionFunc,
			}, nil
		},
	)
	cmd := newCheckForUpdatesCommand(testCli)
	cmd.Flags().Set("pre-releases", "true")
	cmd.Flags().Set("downgrades", "true")
	err := cmd.Execute()
	assert.NilError(t, err)
	golden.Assert(t, testCli.OutBuffer().String(), "check-all.golden")

	testCli.OutBuffer().Reset()
	cmd.Flags().Set("pre-releases", "false")
	cmd.Flags().Set("downgrades", "true")
	err = cmd.Execute()
	assert.NilError(t, err)
	fmt.Println(testCli.OutBuffer().String())
	golden.Assert(t, testCli.OutBuffer().String(), "check-no-prerelease.golden")

	testCli.OutBuffer().Reset()
	cmd.Flags().Set("pre-releases", "false")
	cmd.Flags().Set("downgrades", "false")
	err = cmd.Execute()
	assert.NilError(t, err)
	fmt.Println(testCli.OutBuffer().String())
	golden.Assert(t, testCli.OutBuffer().String(), "check-no-downgrades.golden")

	testCli.OutBuffer().Reset()
	cmd.Flags().Set("pre-releases", "false")
	cmd.Flags().Set("downgrades", "false")
	cmd.Flags().Set("upgrades", "false")
	err = cmd.Execute()
	assert.NilError(t, err)
	fmt.Println(testCli.OutBuffer().String())
	golden.Assert(t, testCli.OutBuffer().String(), "check-patches-only.golden")
}

func makeVersion(t *testing.T, tag string) containerizedengine.DockerVersion {
	v, err := ver.NewVersion(tag)
	assert.NilError(t, err)
	return containerizedengine.DockerVersion{Version: *v, Tag: tag}
}

func parseVersions(t *testing.T, tags ...string) []containerizedengine.DockerVersion {
	ret := make([]containerizedengine.DockerVersion, len(tags))
	for i, tag := range tags {
		ret[i] = makeVersion(t, tag)
	}
	return ret
}
