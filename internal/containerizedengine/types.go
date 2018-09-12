package containerizedengine

import (
	"context"
	"errors"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/content"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const (
	containerdSockPath  = "/run/containerd/containerd.sock"
	engineContainerName = "dockerd"
	engineNamespace     = "docker"

	// Used to signal the containerd-proxy if it should manage
	proxyLabel = "com.docker/containerd-proxy.scope"
)

var (
	// ErrEngineAlreadyPresent returned when engine already present and should not be
	ErrEngineAlreadyPresent = errors.New("engine already present, use the update command to change versions")

	// ErrEngineNotPresent returned when the engine is not present and should be
	ErrEngineNotPresent = errors.New("engine not present")

	// ErrMalformedConfigFileParam returned if the engine config file parameter is malformed
	ErrMalformedConfigFileParam = errors.New("malformed --config-file param on engine")

	// ErrEngineConfigLookupFailure returned if unable to lookup existing engine configuration
	ErrEngineConfigLookupFailure = errors.New("unable to lookup existing engine configuration")

	// ErrEngineShutdownTimeout returned if the engine failed to shutdown in time
	ErrEngineShutdownTimeout = errors.New("timeout waiting for engine to exit")

	// ErrEngineImageMissingTag returned if the engine image is missing the version tag
	ErrEngineImageMissingTag = errors.New("malformed engine image missing tag")

	engineSpec = specs.Spec{
		Root: &specs.Root{
			Path: "rootfs",
		},
		Process: &specs.Process{
			Cwd: "/",
			Args: []string{
				// In general, configuration should be driven by the config file, not these flags
				// TODO - consider moving more of these to the config file, and make sure the defaults are set if not present.
				"/sbin/dockerd",
				"-s",
				"overlay2",
				"--containerd",
				"/run/containerd/containerd.sock",
				"--default-runtime",
				"containerd",
				"--add-runtime",
				"containerd=runc",
			},
			User: specs.User{
				UID: 0,
				GID: 0,
			},
			Env: []string{
				"PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin",
			},
			NoNewPrivileges: false,
		},
	}
)

type baseClient struct {
	cclient containerdClient
}

// containerdClient abstracts the containerd client to aid in testability
type containerdClient interface {
	Containers(ctx context.Context, filters ...string) ([]containerd.Container, error)
	NewContainer(ctx context.Context, id string, opts ...containerd.NewContainerOpts) (containerd.Container, error)
	Pull(ctx context.Context, ref string, opts ...containerd.RemoteOpt) (containerd.Image, error)
	GetImage(ctx context.Context, ref string) (containerd.Image, error)
	Close() error
	ContentStore() content.Store
	ContainerService() containers.Store
}
