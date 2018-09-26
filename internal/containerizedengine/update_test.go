package containerizedengine

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/errdefs"
	"github.com/docker/cli/cli/command"
	clitypes "github.com/docker/cli/types"
	"github.com/docker/docker/api/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"gotest.tools/assert"
)

func healthfnHappy(ctx context.Context) error {
	return nil
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
	opts := clitypes.EngineInitOptions{
		EngineVersion:  "engineversiongoeshere",
		RegistryPrefix: "registryprefixgoeshere",
		ConfigFile:     "/tmp/configfilegoeshere",
		EngineImage:    clitypes.EnterpriseEngineImage,
	}

	err := client.ActivateEngine(ctx, opts, command.NewOutStream(&bytes.Buffer{}), &types.AuthConfig{}, healthfnHappy)
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
	opts := clitypes.EngineInitOptions{
		EngineVersion:  "engineversiongoeshere",
		RegistryPrefix: "registryprefixgoeshere",
		ConfigFile:     "/tmp/configfilegoeshere",
		EngineImage:    clitypes.EnterpriseEngineImage,
	}

	err := client.ActivateEngine(ctx, opts, command.NewOutStream(&bytes.Buffer{}), &types.AuthConfig{}, healthfnHappy)
	assert.ErrorContains(t, err, "check for image")
	assert.ErrorContains(t, err, "something went wrong")
}

func TestDoUpdateNoVersion(t *testing.T) {
	ctx := context.Background()
	opts := clitypes.EngineInitOptions{
		EngineVersion:  "",
		RegistryPrefix: "registryprefixgoeshere",
		ConfigFile:     "/tmp/configfilegoeshere",
		EngineImage:    clitypes.EnterpriseEngineImage,
	}
	client := baseClient{}
	err := client.DoUpdate(ctx, opts, command.NewOutStream(&bytes.Buffer{}), &types.AuthConfig{}, healthfnHappy)
	assert.ErrorContains(t, err, "pick the version you")
}

func TestDoUpdateImageMiscError(t *testing.T) {
	ctx := context.Background()
	opts := clitypes.EngineInitOptions{
		EngineVersion:  "engineversiongoeshere",
		RegistryPrefix: "registryprefixgoeshere",
		ConfigFile:     "/tmp/configfilegoeshere",
		EngineImage:    "testnamegoeshere",
	}
	client := baseClient{
		cclient: &fakeContainerdClient{
			getImageFunc: func(ctx context.Context, ref string) (containerd.Image, error) {
				return nil, fmt.Errorf("something went wrong")

			},
		},
	}
	err := client.DoUpdate(ctx, opts, command.NewOutStream(&bytes.Buffer{}), &types.AuthConfig{}, healthfnHappy)
	assert.ErrorContains(t, err, "check for image")
	assert.ErrorContains(t, err, "something went wrong")
}

func TestDoUpdatePullFail(t *testing.T) {
	ctx := context.Background()
	opts := clitypes.EngineInitOptions{
		EngineVersion:  "engineversiongoeshere",
		RegistryPrefix: "registryprefixgoeshere",
		ConfigFile:     "/tmp/configfilegoeshere",
		EngineImage:    "testnamegoeshere",
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
	err := client.DoUpdate(ctx, opts, command.NewOutStream(&bytes.Buffer{}), &types.AuthConfig{}, healthfnHappy)
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
	opts := clitypes.EngineInitOptions{
		EngineVersion:  "engineversiongoeshere",
		RegistryPrefix: "registryprefixgoeshere",
		ConfigFile:     "/tmp/configfilegoeshere",
	}

	tmpdir, err := ioutil.TempDir("", "docker-root")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	tmpDockerRoot := defaultDockerRoot
	defaultDockerRoot = tmpdir
	defer func() {
		defaultDockerRoot = tmpDockerRoot
	}()
	metadata := RuntimeMetadata{Platform: "platformgoeshere"}
	err = client.WriteRuntimeMetadata(tmpdir, &metadata)
	assert.NilError(t, err)

	err = client.ActivateEngine(ctx, opts, command.NewOutStream(&bytes.Buffer{}), &types.AuthConfig{}, healthfnHappy)
	assert.ErrorContains(t, err, "check for image")
	assert.ErrorContains(t, err, "something went wrong")
	expectedImage := fmt.Sprintf("%s/%s:%s", opts.RegistryPrefix, "engine-enterprise", opts.EngineVersion)
	assert.Assert(t, requestedImage == expectedImage, "%s != %s", requestedImage, expectedImage)

	// Redo with enterprise set
	metadata = RuntimeMetadata{Platform: "Docker Engine - Enterprise"}
	err = client.WriteRuntimeMetadata(tmpdir, &metadata)
	assert.NilError(t, err)

	err = client.ActivateEngine(ctx, opts, command.NewOutStream(&bytes.Buffer{}), &types.AuthConfig{}, healthfnHappy)
	assert.ErrorContains(t, err, "check for image")
	assert.ErrorContains(t, err, "something went wrong")
	expectedImage = fmt.Sprintf("%s/%s:%s", opts.RegistryPrefix, "engine-enterprise", opts.EngineVersion)
	assert.Assert(t, requestedImage == expectedImage, "%s != %s", requestedImage, expectedImage)
}

func TestGetCurrentRuntimeMetadataNotPresent(t *testing.T) {
	ctx := context.Background()
	tmpdir, err := ioutil.TempDir("", "docker-root")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	client := baseClient{}
	_, err = client.GetCurrentRuntimeMetadata(ctx, tmpdir)
	assert.ErrorType(t, err, os.IsNotExist)
}

func TestGetCurrentRuntimeMetadataBadJson(t *testing.T) {
	ctx := context.Background()
	tmpdir, err := ioutil.TempDir("", "docker-root")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	filename := filepath.Join(tmpdir, runtimeMetadataName+".json")
	err = ioutil.WriteFile(filename, []byte("not json"), 0644)
	assert.NilError(t, err)
	client := baseClient{}
	_, err = client.GetCurrentRuntimeMetadata(ctx, tmpdir)
	assert.ErrorContains(t, err, "malformed runtime metadata file")
}

func TestGetCurrentRuntimeMetadataHappyPath(t *testing.T) {
	ctx := context.Background()
	tmpdir, err := ioutil.TempDir("", "docker-root")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	client := baseClient{}
	metadata := RuntimeMetadata{Platform: "platformgoeshere"}
	err = client.WriteRuntimeMetadata(tmpdir, &metadata)
	assert.NilError(t, err)

	res, err := client.GetCurrentRuntimeMetadata(ctx, tmpdir)
	assert.NilError(t, err)
	assert.Equal(t, res.Platform, "platformgoeshere")
}

func TestGetReleaseNotesURL(t *testing.T) {
	imageName := "bogus image name #$%&@!"
	url := getReleaseNotesURL(imageName)
	assert.Equal(t, url, clitypes.ReleaseNotePrefix+"/")
	imageName = "foo.bar/valid/repowithouttag"
	url = getReleaseNotesURL(imageName)
	assert.Equal(t, url, clitypes.ReleaseNotePrefix+"/")
	imageName = "foo.bar/valid/repowithouttag:tag123"
	url = getReleaseNotesURL(imageName)
	assert.Equal(t, url, clitypes.ReleaseNotePrefix+"/tag123")
}
