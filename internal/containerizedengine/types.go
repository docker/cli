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
	engineNamespace     = "com.docker"

	// runtimeMetadataName is the name of the runtime metadata file
	// When stored as a label on the container it is prefixed by "com.docker."
	runtimeMetadataName = "distribution_based_engine"
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
	Install(context.Context, containerd.Image, ...containerd.InstallOpts) error
	Version(ctx context.Context) (containerd.Version, error)
}

// RuntimeMetadata holds platform information about the daemon
type RuntimeMetadata struct {
	Platform             string `json:"platform"`
	ContainerdMinVersion string `json:"containerd_min_version"`
	Runtime              string `json:"runtime"`
}
