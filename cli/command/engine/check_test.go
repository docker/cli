package engine

import (
	"context"
	"fmt"
	"testing"

	manifesttypes "github.com/docker/cli/cli/manifest/types"
	"github.com/docker/cli/internal/test"
	"github.com/docker/distribution"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/opencontainers/go-digest"
	"gotest.tools/assert"
	"gotest.tools/golden"
)

var (
	testCli = test.NewFakeCli(&client.Client{})
)

type verClient struct {
	client.Client
	ver    types.Version
	verErr error
}

func (c *verClient) ServerVersion(ctx context.Context) (types.Version, error) {
	return c.ver, c.verErr
}

type testRegistryClient struct {
	tags []string
}

func (c testRegistryClient) GetManifest(ctx context.Context, ref reference.Named) (manifesttypes.ImageManifest, error) {
	return manifesttypes.ImageManifest{}, nil
}
func (c testRegistryClient) GetManifestList(ctx context.Context, ref reference.Named) ([]manifesttypes.ImageManifest, error) {
	return nil, nil
}
func (c testRegistryClient) MountBlob(ctx context.Context, source reference.Canonical, target reference.Named) error {
	return nil
}

func (c testRegistryClient) PutManifest(ctx context.Context, ref reference.Named, manifest distribution.Manifest) (digest.Digest, error) {
	return "", nil
}
func (c testRegistryClient) GetTags(ctx context.Context, ref reference.Named) ([]string, error) {
	return c.tags, nil
}

func TestCheckForUpdatesNoCurrentVersion(t *testing.T) {
	isRoot = func() bool { return true }
	c := test.NewFakeCli(&verClient{client.Client{}, types.Version{}, nil})
	c.SetRegistryClient(testRegistryClient{})
	cmd := newCheckForUpdatesCommand(c)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	assert.ErrorContains(t, err, "alformed version")
}

func TestCheckForUpdatesGetEngineVersionsHappy(t *testing.T) {
	c := test.NewFakeCli(&verClient{client.Client{}, types.Version{Version: "1.1.0"}, nil})
	c.SetRegistryClient(testRegistryClient{[]string{
		"1.0.1", "1.0.2", "1.0.3-beta1",
		"1.1.1", "1.1.2", "1.1.3-beta1",
		"1.2.0", "2.0.0", "2.1.0-beta1",
	}})
	isRoot = func() bool { return true }
	cmd := newCheckForUpdatesCommand(c)
	cmd.Flags().Set("pre-releases", "true")
	cmd.Flags().Set("downgrades", "true")
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	assert.NilError(t, err)
	golden.Assert(t, c.OutBuffer().String(), "check-all.golden")

	c.OutBuffer().Reset()
	cmd.Flags().Set("pre-releases", "false")
	cmd.Flags().Set("downgrades", "true")
	err = cmd.Execute()
	assert.NilError(t, err)
	fmt.Println(c.OutBuffer().String())
	golden.Assert(t, c.OutBuffer().String(), "check-no-prerelease.golden")

	c.OutBuffer().Reset()
	cmd.Flags().Set("pre-releases", "false")
	cmd.Flags().Set("downgrades", "false")
	err = cmd.Execute()
	assert.NilError(t, err)
	fmt.Println(c.OutBuffer().String())
	golden.Assert(t, c.OutBuffer().String(), "check-no-downgrades.golden")

	c.OutBuffer().Reset()
	cmd.Flags().Set("pre-releases", "false")
	cmd.Flags().Set("downgrades", "false")
	cmd.Flags().Set("upgrades", "false")
	err = cmd.Execute()
	assert.NilError(t, err)
	fmt.Println(c.OutBuffer().String())
	golden.Assert(t, c.OutBuffer().String(), "check-patches-only.golden")
}
