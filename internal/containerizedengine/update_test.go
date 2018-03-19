package containerizedengine

import (
	"context"
	"fmt"
	"testing"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/errdefs"
	"github.com/docker/docker/api/types"
	"gotest.tools/assert"
)

func TestGetCurrentEngineVersionHappy(t *testing.T) {
	ctx := context.Background()
	image := &fakeImage{
		nameFunc: func() string {
			return "acme.com/dockermirror/" + CommunityEngineImage + ":engineversion"
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
		},
	}

	opts, err := client.GetCurrentEngineVersion(ctx)
	assert.NilError(t, err)
	assert.Equal(t, opts.EngineImage, CommunityEngineImage)
	assert.Equal(t, opts.RegistryPrefix, "acme.com/dockermirror")
	assert.Equal(t, opts.EngineVersion, "engineversion")
}

func TestGetCurrentEngineVersionEnterpriseHappy(t *testing.T) {
	ctx := context.Background()
	image := &fakeImage{
		nameFunc: func() string {
			return "docker.io/docker/" + EnterpriseEngineImage + ":engineversion"
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
		},
	}

	opts, err := client.GetCurrentEngineVersion(ctx)
	assert.NilError(t, err)
	assert.Equal(t, opts.EngineImage, EnterpriseEngineImage)
	assert.Equal(t, opts.EngineVersion, "engineversion")
	assert.Equal(t, opts.RegistryPrefix, "docker.io/docker")
}

func TestGetCurrentEngineVersionNoEngine(t *testing.T) {
	ctx := context.Background()
	client := baseClient{
		cclient: &fakeContainerdClient{
			containersFunc: func(ctx context.Context, filters ...string) ([]containerd.Container, error) {
				return []containerd.Container{}, nil
			},
		},
	}

	_, err := client.GetCurrentEngineVersion(ctx)
	assert.ErrorContains(t, err, "failed to find existing engine")
}

func TestGetCurrentEngineVersionMiscEngineError(t *testing.T) {
	ctx := context.Background()
	expectedError := fmt.Errorf("some container lookup error")
	client := baseClient{
		cclient: &fakeContainerdClient{
			containersFunc: func(ctx context.Context, filters ...string) ([]containerd.Container, error) {
				return nil, expectedError
			},
		},
	}

	_, err := client.GetCurrentEngineVersion(ctx)
	assert.Assert(t, err == expectedError)
}

func TestGetCurrentEngineVersionImageFailure(t *testing.T) {
	ctx := context.Background()
	container := &fakeContainer{
		imageFunc: func(context.Context) (containerd.Image, error) {
			return nil, fmt.Errorf("container image failure")
		},
	}
	client := baseClient{
		cclient: &fakeContainerdClient{
			containersFunc: func(ctx context.Context, filters ...string) ([]containerd.Container, error) {
				return []containerd.Container{container}, nil
			},
		},
	}

	_, err := client.GetCurrentEngineVersion(ctx)
	assert.ErrorContains(t, err, "container image failure")
}

func TestGetCurrentEngineVersionMalformed(t *testing.T) {
	ctx := context.Background()
	image := &fakeImage{
		nameFunc: func() string {
			return "imagename"
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
		},
	}

	_, err := client.GetCurrentEngineVersion(ctx)
	assert.Assert(t, err == ErrEngineImageMissingTag)
}

func TestActivateNoEngine(t *testing.T) {
	ctx := context.Background()
	client := baseClient{
		cclient: &fakeContainerdClient{
			containersFunc: func(ctx context.Context, filters ...string) ([]containerd.Container, error) {
				return []containerd.Container{}, nil
			},
		},
	}
	opts := EngineInitOptions{
		EngineVersion:  "engineversiongoeshere",
		RegistryPrefix: "registryprefixgoeshere",
		ConfigFile:     "/tmp/configfilegoeshere",
		EngineImage:    EnterpriseEngineImage,
	}

	err := client.ActivateEngine(ctx, opts, &testOutStream{}, &types.AuthConfig{}, healthfnHappy)
	assert.ErrorContains(t, err, "unable to find")
}

func TestActivateNoChange(t *testing.T) {
	ctx := context.Background()
	registryPrefix := "registryprefixgoeshere"
	image := &fakeImage{
		nameFunc: func() string {
			return registryPrefix + "/" + EnterpriseEngineImage + ":engineversion"
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
		},
	}
	opts := EngineInitOptions{
		EngineVersion:  "engineversiongoeshere",
		RegistryPrefix: "registryprefixgoeshere",
		ConfigFile:     "/tmp/configfilegoeshere",
		EngineImage:    EnterpriseEngineImage,
	}

	err := client.ActivateEngine(ctx, opts, &testOutStream{}, &types.AuthConfig{}, healthfnHappy)
	assert.NilError(t, err)
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
	opts := EngineInitOptions{
		EngineVersion:  "engineversiongoeshere",
		RegistryPrefix: "registryprefixgoeshere",
		ConfigFile:     "/tmp/configfilegoeshere",
		EngineImage:    EnterpriseEngineImage,
	}

	err := client.ActivateEngine(ctx, opts, &testOutStream{}, &types.AuthConfig{}, healthfnHappy)
	assert.ErrorContains(t, err, "check for image")
	assert.ErrorContains(t, err, "something went wrong")
}

func TestDoUpdateNoVersion(t *testing.T) {
	ctx := context.Background()
	opts := EngineInitOptions{
		EngineVersion:  "",
		RegistryPrefix: "registryprefixgoeshere",
		ConfigFile:     "/tmp/configfilegoeshere",
		EngineImage:    EnterpriseEngineImage,
	}
	client := baseClient{}
	err := client.DoUpdate(ctx, opts, &testOutStream{}, &types.AuthConfig{}, healthfnHappy)
	assert.ErrorContains(t, err, "please pick the version you")
}

func TestDoUpdateImageMiscError(t *testing.T) {
	ctx := context.Background()
	opts := EngineInitOptions{
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
	err := client.DoUpdate(ctx, opts, &testOutStream{}, &types.AuthConfig{}, healthfnHappy)
	assert.ErrorContains(t, err, "check for image")
	assert.ErrorContains(t, err, "something went wrong")
}

func TestDoUpdatePullFail(t *testing.T) {
	ctx := context.Background()
	opts := EngineInitOptions{
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
	err := client.DoUpdate(ctx, opts, &testOutStream{}, &types.AuthConfig{}, healthfnHappy)
	assert.ErrorContains(t, err, "unable to pull")
	assert.ErrorContains(t, err, "pull failure")
}

func TestDoUpdateEngineMissing(t *testing.T) {
	ctx := context.Background()
	opts := EngineInitOptions{
		EngineVersion:  "engineversiongoeshere",
		RegistryPrefix: "registryprefixgoeshere",
		ConfigFile:     "/tmp/configfilegoeshere",
		EngineImage:    "testnamegoeshere",
	}
	image := &fakeImage{
		nameFunc: func() string {
			return "imagenamehere"
		},
	}
	client := baseClient{
		cclient: &fakeContainerdClient{
			getImageFunc: func(ctx context.Context, ref string) (containerd.Image, error) {
				return image, nil

			},
			containersFunc: func(ctx context.Context, filters ...string) ([]containerd.Container, error) {
				return []containerd.Container{}, nil
			},
		},
	}
	err := client.DoUpdate(ctx, opts, &testOutStream{}, &types.AuthConfig{}, healthfnHappy)
	assert.ErrorContains(t, err, "unable to find existing engine")
}
