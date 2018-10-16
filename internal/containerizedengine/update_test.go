package containerizedengine

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/errdefs"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/versions"
	clitypes "github.com/docker/cli/types"
	"github.com/docker/docker/api/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"gotest.tools/assert"
)

func TestActivateImagePermutations(t *testing.T) {
	ctx := context.Background()
	lookedup := "not called yet"
	expectedError := fmt.Errorf("expected error")
	client := baseClient{
		cclient: &fakeContainerdClient{
			getImageFunc: func(ctx context.Context, ref string) (containerd.Image, error) {
				lookedup = ref
				return nil, expectedError
			},
		},
	}
	tmpdir, err := ioutil.TempDir("", "enginedir")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	metadata := clitypes.RuntimeMetadata{EngineImage: clitypes.EnterpriseEngineImage}
	err = versions.WriteRuntimeMetadata(tmpdir, &metadata)
	assert.NilError(t, err)

	opts := clitypes.EngineInitOptions{
		EngineVersion:      "engineversiongoeshere",
		RegistryPrefix:     "registryprefixgoeshere",
		ConfigFile:         "/tmp/configfilegoeshere",
		RuntimeMetadataDir: tmpdir,
	}

	err = client.ActivateEngine(ctx, opts, command.NewOutStream(&bytes.Buffer{}), &types.AuthConfig{})
	assert.ErrorContains(t, err, expectedError.Error())
	assert.Equal(t, lookedup, fmt.Sprintf("%s/%s:%s", opts.RegistryPrefix, clitypes.EnterpriseEngineImage, opts.EngineVersion))

	metadata = clitypes.RuntimeMetadata{EngineImage: clitypes.CommunityEngineImage}
	err = versions.WriteRuntimeMetadata(tmpdir, &metadata)
	assert.NilError(t, err)
	err = client.ActivateEngine(ctx, opts, command.NewOutStream(&bytes.Buffer{}), &types.AuthConfig{})
	assert.ErrorContains(t, err, expectedError.Error())
	assert.Equal(t, lookedup, fmt.Sprintf("%s/%s:%s", opts.RegistryPrefix, clitypes.EnterpriseEngineImage, opts.EngineVersion))

	metadata = clitypes.RuntimeMetadata{EngineImage: clitypes.CommunityEngineImage + "-dm"}
	err = versions.WriteRuntimeMetadata(tmpdir, &metadata)
	assert.NilError(t, err)
	err = client.ActivateEngine(ctx, opts, command.NewOutStream(&bytes.Buffer{}), &types.AuthConfig{})
	assert.ErrorContains(t, err, expectedError.Error())
	assert.Equal(t, lookedup, fmt.Sprintf("%s/%s:%s", opts.RegistryPrefix, clitypes.EnterpriseEngineImage+"-dm", opts.EngineVersion))
}

func TestActivateConfigFailure(t *testing.T) {
	ctx := context.Background()
	registryPrefix := "registryprefixgoeshere"
	image := &fakeImage{
		nameFunc: func() string {
			return registryPrefix + "/" + clitypes.EnterpriseEngineImage + ":engineversion"
		},
		configFunc: func(ctx context.Context) (ocispec.Descriptor, error) {
			return ocispec.Descriptor{}, fmt.Errorf("config lookup failure")
		},
	}
	container := &fakeContainer{
		imageFunc: func(context.Context) (containerd.Image, error) {
			return image, nil
		},
		taskFunc: func(context.Context, cio.Attach) (containerd.Task, error) {
			return nil, errdefs.ErrNotFound
		},
		labelsFunc: func(context.Context) (map[string]string, error) {
			return map[string]string{}, nil
		},
	}
	client := baseClient{
		cclient: &fakeContainerdClient{
			containersFunc: func(ctx context.Context, filters ...string) ([]containerd.Container, error) {
				return []containerd.Container{container}, nil
			},
			getImageFunc: func(ctx context.Context, ref string) (containerd.Image, error) {
				return image, nil
			},
		},
	}
	tmpdir, err := ioutil.TempDir("", "engindir")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	metadata := clitypes.RuntimeMetadata{EngineImage: clitypes.CommunityEngineImage}
	err = versions.WriteRuntimeMetadata(tmpdir, &metadata)
	assert.NilError(t, err)
	opts := clitypes.EngineInitOptions{
		EngineVersion:      "engineversiongoeshere",
		RegistryPrefix:     "registryprefixgoeshere",
		ConfigFile:         "/tmp/configfilegoeshere",
		EngineImage:        clitypes.EnterpriseEngineImage,
		RuntimeMetadataDir: tmpdir,
	}

	err = client.ActivateEngine(ctx, opts, command.NewOutStream(&bytes.Buffer{}), &types.AuthConfig{})
	assert.ErrorContains(t, err, "config lookup failure")
}

func TestActivateDoUpdateFail(t *testing.T) {
	ctx := context.Background()
	registryPrefix := "registryprefixgoeshere"
	image := &fakeImage{
		nameFunc: func() string {
			return registryPrefix + "/ce-engine:engineversion"
		},
	}
	container := &fakeContainer{
		imageFunc: func(context.Context) (containerd.Image, error) {
			return image, nil
		},
	}
	client := baseClient{
		cclient: &fakeContainerdClient{
			containersFunc: func(ctx context.Context, filters ...string) ([]containerd.Container, error) {
				return []containerd.Container{container}, nil
			},
			getImageFunc: func(ctx context.Context, ref string) (containerd.Image, error) {
				return nil, fmt.Errorf("something went wrong")

			},
		},
	}
	tmpdir, err := ioutil.TempDir("", "enginedir")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	metadata := clitypes.RuntimeMetadata{EngineImage: clitypes.CommunityEngineImage}
	err = versions.WriteRuntimeMetadata(tmpdir, &metadata)
	assert.NilError(t, err)
	opts := clitypes.EngineInitOptions{
		EngineVersion:      "engineversiongoeshere",
		RegistryPrefix:     "registryprefixgoeshere",
		ConfigFile:         "/tmp/configfilegoeshere",
		EngineImage:        clitypes.EnterpriseEngineImage,
		RuntimeMetadataDir: tmpdir,
	}

	err = client.ActivateEngine(ctx, opts, command.NewOutStream(&bytes.Buffer{}), &types.AuthConfig{})
	assert.ErrorContains(t, err, "check for image")
	assert.ErrorContains(t, err, "something went wrong")
}

func TestDoUpdateNoVersion(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "enginedir")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	metadata := clitypes.RuntimeMetadata{EngineImage: clitypes.EnterpriseEngineImage}
	err = versions.WriteRuntimeMetadata(tmpdir, &metadata)
	assert.NilError(t, err)
	ctx := context.Background()
	opts := clitypes.EngineInitOptions{
		EngineVersion:      "",
		RegistryPrefix:     "registryprefixgoeshere",
		ConfigFile:         "/tmp/configfilegoeshere",
		EngineImage:        clitypes.EnterpriseEngineImage,
		RuntimeMetadataDir: tmpdir,
	}

	client := baseClient{}
	err = client.DoUpdate(ctx, opts, command.NewOutStream(&bytes.Buffer{}), &types.AuthConfig{})
	assert.ErrorContains(t, err, "pick the version you")
}

func TestDoUpdateImageMiscError(t *testing.T) {
	ctx := context.Background()
	tmpdir, err := ioutil.TempDir("", "enginedir")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	metadata := clitypes.RuntimeMetadata{EngineImage: clitypes.EnterpriseEngineImage}
	err = versions.WriteRuntimeMetadata(tmpdir, &metadata)
	assert.NilError(t, err)
	opts := clitypes.EngineInitOptions{
		EngineVersion:      "engineversiongoeshere",
		RegistryPrefix:     "registryprefixgoeshere",
		ConfigFile:         "/tmp/configfilegoeshere",
		EngineImage:        "testnamegoeshere",
		RuntimeMetadataDir: tmpdir,
	}
	client := baseClient{
		cclient: &fakeContainerdClient{
			getImageFunc: func(ctx context.Context, ref string) (containerd.Image, error) {
				return nil, fmt.Errorf("something went wrong")

			},
		},
	}

	err = client.DoUpdate(ctx, opts, command.NewOutStream(&bytes.Buffer{}), &types.AuthConfig{})
	assert.ErrorContains(t, err, "check for image")
	assert.ErrorContains(t, err, "something went wrong")
}

func TestDoUpdatePullFail(t *testing.T) {
	ctx := context.Background()
	tmpdir, err := ioutil.TempDir("", "enginedir")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	metadata := clitypes.RuntimeMetadata{EngineImage: clitypes.EnterpriseEngineImage}
	err = versions.WriteRuntimeMetadata(tmpdir, &metadata)
	assert.NilError(t, err)
	opts := clitypes.EngineInitOptions{
		EngineVersion:      "engineversiongoeshere",
		RegistryPrefix:     "registryprefixgoeshere",
		ConfigFile:         "/tmp/configfilegoeshere",
		EngineImage:        "testnamegoeshere",
		RuntimeMetadataDir: tmpdir,
	}
	client := baseClient{
		cclient: &fakeContainerdClient{
			getImageFunc: func(ctx context.Context, ref string) (containerd.Image, error) {
				return nil, errdefs.ErrNotFound

			},
			pullFunc: func(ctx context.Context, ref string, opts ...containerd.RemoteOpt) (containerd.Image, error) {
				return nil, fmt.Errorf("pull failure")
			},
		},
	}

	err = client.DoUpdate(ctx, opts, command.NewOutStream(&bytes.Buffer{}), &types.AuthConfig{})
	assert.ErrorContains(t, err, "unable to pull")
	assert.ErrorContains(t, err, "pull failure")
}

func TestActivateDoUpdateVerifyImageName(t *testing.T) {
	ctx := context.Background()
	registryPrefix := "registryprefixgoeshere"
	image := &fakeImage{
		nameFunc: func() string {
			return registryPrefix + "/ce-engine:engineversion"
		},
	}
	container := &fakeContainer{
		imageFunc: func(context.Context) (containerd.Image, error) {
			return image, nil
		},
	}
	requestedImage := "unset"
	client := baseClient{
		cclient: &fakeContainerdClient{
			containersFunc: func(ctx context.Context, filters ...string) ([]containerd.Container, error) {
				return []containerd.Container{container}, nil
			},
			getImageFunc: func(ctx context.Context, ref string) (containerd.Image, error) {
				requestedImage = ref
				return nil, fmt.Errorf("something went wrong")

			},
		},
	}
	tmpdir, err := ioutil.TempDir("", "enginedir")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	metadata := clitypes.RuntimeMetadata{EngineImage: clitypes.EnterpriseEngineImage}
	err = versions.WriteRuntimeMetadata(tmpdir, &metadata)
	assert.NilError(t, err)

	opts := clitypes.EngineInitOptions{
		EngineVersion:      "engineversiongoeshere",
		RegistryPrefix:     "registryprefixgoeshere",
		EngineImage:        "testnamegoeshere",
		ConfigFile:         "/tmp/configfilegoeshere",
		RuntimeMetadataDir: tmpdir,
	}

	err = client.ActivateEngine(ctx, opts, command.NewOutStream(&bytes.Buffer{}), &types.AuthConfig{})
	assert.ErrorContains(t, err, "check for image")
	assert.ErrorContains(t, err, "something went wrong")
	expectedImage := fmt.Sprintf("%s/%s:%s", opts.RegistryPrefix, opts.EngineImage, opts.EngineVersion)
	assert.Assert(t, requestedImage == expectedImage, "%s != %s", requestedImage, expectedImage)
}

func TestGetReleaseNotesURL(t *testing.T) {
	imageName := "bogus image name #$%&@!"
	url := getReleaseNotesURL(imageName)
	assert.Equal(t, url, clitypes.ReleaseNotePrefix+"?")
	imageName = "foo.bar/valid/repowithouttag"
	url = getReleaseNotesURL(imageName)
	assert.Equal(t, url, clitypes.ReleaseNotePrefix+"?")
	imageName = "foo.bar/valid/repowithouttag:tag123"
	url = getReleaseNotesURL(imageName)
	assert.Equal(t, url, clitypes.ReleaseNotePrefix+"?tag123")
}
