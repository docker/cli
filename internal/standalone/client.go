//go:build linux

package standalone

import (
	"context"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/docker/cli/cli/version"
	"github.com/moby/moby/api/types/build"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
	"golang.org/x/sys/unix"
)

// NewAPIClient returns a [client.APIClient] backed by the embedded containerd
// libraries. It is used by the CLI in place of the HTTP client when
// DOCKER_STANDALONE is set.
func NewAPIClient(context.Context) (client.APIClient, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	return &apiClient{eng: newEngine(cfg)}, nil
}

// engine ties the configuration, the stores and the task manager together.
type engine struct {
	cfg *config

	mu           sync.Mutex
	execs        map[string]*execProcess
	attaches     map[string]*attachConn
	taskMgr      *taskManager
	startedTasks map[string]*taskManager

	// sessMu guards the reference-counted store session; see withSession.
	sessMu   sync.Mutex
	sess     *session
	sessRefs int
}

func newEngine(cfg *config) *engine {
	return &engine{cfg: cfg, execs: map[string]*execProcess{}}
}

// apiClient implements client.APIClient on top of the engine. Methods that
// are not applicable to a daemonless, single-host setup (swarm, plugins,
// build, ...) fall through to the embedded unsupported implementation.
type apiClient struct {
	unsupported
	eng *engine
}

var _ client.APIClient = (*apiClient)(nil)

func (*apiClient) ClientVersion() string { return client.MaxAPIVersion }

func (c *apiClient) DaemonHost() string { return "standalone://" + c.eng.cfg.root }

func (*apiClient) Close() error { return nil }

func (*apiClient) Dialer() func(context.Context) (net.Conn, error) {
	return func(context.Context) (net.Conn, error) {
		return nil, errNotSupported("Dialer")
	}
}

func (*apiClient) DialHijack(context.Context, string, string, map[string][]string) (net.Conn, error) {
	return nil, errNotSupported("DialHijack")
}

func (*apiClient) Ping(context.Context, client.PingOptions) (client.PingResult, error) {
	return client.PingResult{
		APIVersion:     client.MaxAPIVersion,
		OSType:         "linux",
		Experimental:   false,
		BuilderVersion: build.BuilderV1,
	}, nil
}

func (c *apiClient) ServerVersion(context.Context, client.ServerVersionOptions) (client.ServerVersionResult, error) {
	var uts unix.Utsname
	_ = unix.Uname(&uts)
	return client.ServerVersionResult{
		Platform:      client.PlatformInfo{Name: "Docker CLI standalone (containerd library)"},
		Version:       version.Version,
		APIVersion:    client.MaxAPIVersion,
		MinAPIVersion: client.MaxAPIVersion,
		Os:            "linux",
		Arch:          runtime.GOARCH,
		Components: []system.ComponentVersion{
			{Name: "Engine", Version: version.Version, Details: map[string]string{
				"ApiVersion":    client.MaxAPIVersion,
				"MinAPIVersion": client.MaxAPIVersion,
				"GitCommit":     version.GitCommit,
				"GoVersion":     runtime.Version(),
				"Os":            "linux",
				"Arch":          runtime.GOARCH,
				"KernelVersion": unix.ByteSliceToString(uts.Release[:]),
				"BuildTime":     version.BuildTime,
				"Mode":          c.eng.modeString(),
			}},
			{Name: "containerd (library)", Version: containerdVersion()},
			{Name: "runtime", Version: c.eng.cfg.runtimeBinary},
		},
	}, nil
}

func (c *apiClient) Info(ctx context.Context, _ client.InfoOptions) (client.SystemInfoResult, error) {
	var uts unix.Utsname
	_ = unix.Uname(&uts)
	hostname, _ := os.Hostname()
	info := system.Info{
		ID:              "standalone",
		Driver:          c.eng.cfg.snapshotter,
		LoggingDriver:   "json-file",
		CgroupDriver:    "cgroupfs",
		CgroupVersion:   cgroupVersion(),
		KernelVersion:   unix.ByteSliceToString(uts.Release[:]),
		OperatingSystem: osPrettyName(),
		OSType:          "linux",
		Architecture:    runtime.GOARCH,
		NCPU:            runtime.NumCPU(),
		MemTotal:        totalMemory(),
		DockerRootDir:   c.eng.cfg.root,
		Name:            hostname,
		ServerVersion:   version.Version,
		DefaultRuntime:  c.eng.cfg.runtimeBinary,
		Runtimes: map[string]system.RuntimeWithStatus{
			c.eng.cfg.runtimeBinary: {Runtime: system.Runtime{Path: c.eng.cfg.runtimeBinary}},
		},
		SystemTime:      time.Now().Format(time.RFC3339Nano),
		SecurityOptions: securityOptions(c.eng.cfg),
		Labels: []string{
			"standalone.mode=" + c.eng.modeString(),
			"standalone.network=" + c.eng.cfg.network,
			"standalone.rundir=" + c.eng.cfg.runDir,
		},
	}
	if c.eng.cfg.rootless {
		info.SecurityOptions = append(info.SecurityOptions, "name=rootless")
	}
	err := c.eng.withSession(ctx, func(ctx context.Context, s *session) error {
		imgs, err := s.images().List(ctx)
		if err != nil {
			return err
		}
		info.Images = len(imgs)
		ctrs, err := s.containers().List(ctx)
		if err != nil {
			return err
		}
		info.Containers = len(ctrs)
		for _, ctr := range ctrs {
			m, err := unmarshalMeta(ctr)
			if err != nil {
				continue
			}
			switch m.State.Status {
			case statusRunning:
				info.ContainersRunning++
			case statusPaused:
				info.ContainersPaused++
			default:
				info.ContainersStopped++
			}
		}
		return nil
	})
	if err != nil {
		info.Warnings = append(info.Warnings, "WARNING: "+err.Error())
	}
	return client.SystemInfoResult{Info: info}, nil
}

func (*apiClient) Events(ctx context.Context, _ client.EventsListOptions) client.EventsResult {
	msgs := make(chan events.Message)
	errs := make(chan error, 1)
	go func() {
		<-ctx.Done()
		errs <- ctx.Err()
	}()
	return client.EventsResult{Messages: msgs, Err: errs}
}

func (e *engine) modeString() string {
	if e.cfg.rootless {
		return "rootless"
	}
	return "rootful"
}

func securityOptions(cfg *config) []string {
	opts := []string{"name=seccomp,profile=builtin"}
	if cfg.rootless {
		opts = append(opts, "name=userns")
	}
	return opts
}
