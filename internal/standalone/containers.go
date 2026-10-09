//go:build linux

package standalone

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/containerd/containerd/v2/core/containers"
	"github.com/containerd/containerd/v2/core/diff/apply"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/core/mount"
	"github.com/containerd/containerd/v2/core/runtime"
	"github.com/containerd/containerd/v2/core/unpack"
	cerrdefs "github.com/containerd/errdefs"
	"github.com/containerd/log"
	"github.com/containerd/platforms"
	"github.com/containerd/typeurl/v2"
	"github.com/docker/go-units"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"github.com/opencontainers/image-spec/identity"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

const (
	statusCreated = "created"
	statusRunning = "running"
	statusPaused  = "paused"
	statusExited  = "exited"
)

// ---- create

//nolint:gocyclo // validation and defaulting of the full container configuration
func (c *apiClient) ContainerCreate(ctx context.Context, options client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
	if options.Config == nil {
		return client.ContainerCreateResult{}, fmt.Errorf("container config is required: %w", cerrdefs.ErrInvalidArgument)
	}
	cfg := options.Config
	hc := options.HostConfig
	if hc == nil {
		hc = &container.HostConfig{}
	}
	if options.Image != "" && cfg.Image == "" {
		cfg.Image = options.Image
	}
	if cfg.Image == "" {
		return client.ContainerCreateResult{}, fmt.Errorf("image is required: %w", cerrdefs.ErrInvalidArgument)
	}
	if hc.Runtime != "" && hc.Runtime != "runc" && hc.Runtime != "crun" && hc.Runtime != filepath.Base(c.eng.cfg.runtimeBinary) {
		return client.ContainerCreateResult{}, fmt.Errorf("runtime %q is not available; the configured runtime is %s: %w", hc.Runtime, c.eng.cfg.runtimeBinary, cerrdefs.ErrInvalidArgument)
	}
	if !hc.RestartPolicy.IsNone() && hc.RestartPolicy.Name != "" {
		log.G(ctx).Warn("restart policies are not enforced in standalone mode (no daemon is running to restart containers)")
	}
	netBackend, err := c.eng.resolveNetworkBackend(hc)
	if err != nil {
		return client.ContainerCreateResult{}, err
	}

	var platform platforms.MatchComparer
	platformStr := ""
	if options.Platform != nil {
		platform = platforms.Only(*options.Platform)
		platformStr = platforms.Format(*options.Platform)
	}

	id := newID()
	var res client.ContainerCreateResult
	err = c.eng.withSession(ctx, func(ctx context.Context, s *session) (retErr error) {
		ri, err := resolveImage(ctx, s, cfg.Image, platform)
		if err != nil {
			return err
		}
		store := s.containers()
		name := strings.TrimPrefix(options.Name, "/")
		if name == "" {
			name, err = randomName(ctx, store)
			if err != nil {
				return err
			}
		} else if err := validateName(name); err != nil {
			return err
		}
		if existing, _, err := loadContainer(ctx, store, name); err == nil {
			return fmt.Errorf("the container name %q is already in use by container %q: %w", name, shortID(existing.ID), cerrdefs.ErrConflict)
		}

		if cfg.Hostname == "" {
			cfg.Hostname = shortID(id)
		}
		if cfg.Labels == nil {
			cfg.Labels = map[string]string{}
		}
		if hc.NetworkMode == "" {
			hc.NetworkMode = networkBridge
		}
		if hc.LogConfig.Type == "" {
			hc.LogConfig.Type = "json-file"
		}
		ociMounts, mountPoints, anonVolumes, err := c.eng.resolveMounts(cfg, hc, ri.config.Config.Volumes)
		if err != nil {
			return err
		}
		defer func() {
			if retErr != nil {
				for _, v := range anonVolumes {
					_ = c.eng.removeVolume(v)
				}
			}
		}()

		ctrDir := c.eng.cfg.containerDir(id)
		m := &containerMeta{
			Name:             name,
			Created:          time.Now().UTC(),
			Config:           cfg,
			HostConfig:       hc,
			NetworkingConfig: options.NetworkingConfig,
			Platform:         platformStr,
			ImageID:          ri.ID(),
			ImageRef:         ri.img.Name,
			Mounts:           mountPoints,
			AnonymousVolumes: anonVolumes,
			LogPath:          filepath.Join(ctrDir, id+"-json.log"),
			HostnamePath:     filepath.Join(ctrDir, "hostname"),
			HostsPath:        filepath.Join(ctrDir, "hosts"),
			ResolvConfPath:   filepath.Join(ctrDir, "resolv.conf"),
			Runtime:          filepath.Base(c.eng.cfg.runtimeBinary),
			State:            containerState{Status: statusCreated},
			Network:          networkState{Backend: netBackend},
		}
		if err := c.eng.writeEtcFiles(ctrDir, m); err != nil {
			return err
		}
		defer func() {
			if retErr != nil {
				_ = os.RemoveAll(ctrDir)
			}
		}()

		// Root filesystem snapshot.
		if err := ensureUnpacked(ctx, s, c.eng.cfg.snapshotter, ri); err != nil {
			return err
		}
		diffIDs, err := ri.img.RootFS(ctx, s.content(), ri.platform)
		if err != nil {
			return err
		}
		parent := identity.ChainID(diffIDs).String()
		if _, err := s.snapshotter().Prepare(ctx, id, parent); err != nil {
			return fmt.Errorf("preparing root filesystem: %w", err)
		}
		defer func() {
			if retErr != nil {
				_ = s.snapshotter().Remove(ctx, id)
			}
		}()

		ctr := &containers.Container{
			ID:          id,
			Image:       ri.img.Name,
			Labels:      map[string]string{labelName: name, labelImage: ri.img.Name},
			Runtime:     containers.RuntimeInfo{Name: shimRuntime},
			Snapshotter: c.eng.cfg.snapshotter,
			SnapshotKey: id,
		}
		spec, err := c.eng.buildSpec(ctx, s, ctr, specInput{
			id:      id,
			meta:    m,
			image:   ri,
			mounts:  ociMounts,
			etcDir:  ctrDir,
			netMode: netBackend,
		})
		if err != nil {
			return err
		}
		ctr.Spec, err = typeurl.MarshalAny(spec)
		if err != nil {
			return err
		}
		ctr.Extensions = map[string]typeurl.Any{}
		ctr.Extensions[metaExtension], err = marshalMeta(m)
		if err != nil {
			return err
		}
		if _, err := store.Create(ctx, *ctr); err != nil {
			return err
		}
		res.ID = id
		return nil
	})
	return res, err
}

// ensureUnpacked makes sure the image's layers are unpacked into the
// snapshotter (they may not be when an image was loaded from an archive).
func ensureUnpacked(ctx context.Context, s *session, snapshotter string, ri *resolvedImage) error {
	diffIDs, err := ri.img.RootFS(ctx, s.content(), ri.platform)
	if err != nil {
		return err
	}
	if len(diffIDs) == 0 {
		return nil
	}
	chainID := identity.ChainID(diffIDs).String()
	if _, err := s.snapshotter().Stat(ctx, chainID); err == nil {
		return nil
	}
	u, err := unpack.NewUnpacker(ctx, s.content(), unpack.WithUnpackPlatform(unpack.Platform{
		Platform:       ri.platform,
		SnapshotterKey: snapshotter,
		Snapshotter:    s.snapshotter(),
		Applier:        apply.NewFileSystemApplier(s.content()),
	}))
	if err != nil {
		return err
	}
	h := images.FilterPlatforms(images.ChildrenHandler(s.content()), ri.platform)
	if err := images.Dispatch(ctx, u.Unpack(h), nil, ri.img.Target); err != nil {
		_, _ = u.Wait()
		return fmt.Errorf("unpacking image: %w", err)
	}
	if _, err := u.Wait(); err != nil {
		return fmt.Errorf("unpacking image: %w", err)
	}
	return nil
}

func validateName(name string) error {
	if len(name) < 2 {
		return fmt.Errorf("invalid container name %q: must be at least 2 characters: %w", name, cerrdefs.ErrInvalidArgument)
	}
	for i, r := range name {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || (i > 0 && (r == '_' || r == '.' || r == '-'))
		if !ok {
			return fmt.Errorf("invalid container name %q: only [a-zA-Z0-9][a-zA-Z0-9_.-] are allowed: %w", name, cerrdefs.ErrInvalidArgument)
		}
	}
	return nil
}

// ---- start

func (c *apiClient) ContainerStart(ctx context.Context, ref string, _ client.ContainerStartOptions) (client.ContainerStartResult, error) {
	err := c.eng.startContainer(ctx, ref)
	if err != nil && !cerrdefs.IsNotFound(err) {
		// Like the daemon, remove --rm containers that failed to start so
		// that "docker run --rm" (which waits for removal) can finish.
		if _, m, lerr := c.eng.loadSynced(ctx, ref); lerr == nil && m.HostConfig.AutoRemove {
			_ = c.eng.removeContainer(context.WithoutCancel(ctx), ref, true, true)
		}
	}
	return client.ContainerStartResult{}, err
}

//nolint:gocyclo // container startup wires storage, networking, I/O and the task
func (e *engine) startContainer(ctx context.Context, ref string) (retErr error) {
	var (
		ctr    containers.Container
		m      *containerMeta
		mounts []mount.Mount
	)
	// Phase 1: load the record and resolve the snapshot mounts.
	err := e.withSession(ctx, func(ctx context.Context, s *session) error {
		var err error
		ctr, m, err = loadContainer(ctx, s.containers(), ref)
		if err != nil {
			return err
		}
		mm, err := s.snapshotter().Mounts(ctx, ctr.SnapshotKey)
		if err != nil {
			return fmt.Errorf("resolving root filesystem: %w", err)
		}
		mounts = mm
		return nil
	})
	if err != nil {
		return err
	}
	id := ctr.ID

	tm, err := e.newTaskManager(ctx, e.cfg.eventsSocket(id))
	if err != nil {
		return err
	}
	if task, err := tm.get(ctx, id); err == nil {
		st, serr := task.State(nsCtx(ctx))
		if serr == nil && st.Status != runtime.StoppedStatus {
			// Already running: Docker treats this as a no-op.
			return nil
		}
		if err := e.reap(ctx, tm, id, m); err != nil {
			return err
		}
	}

	runDir := e.cfg.containerRunDir(id)
	_ = os.RemoveAll(runDir)
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return err
	}
	defer func() {
		if retErr != nil {
			_ = os.RemoveAll(runDir)
		}
	}()

	// Networking.
	ns, err := e.setupNetwork(ctx, id, m, m.Network.Backend)
	if err != nil {
		return fmt.Errorf("setting up network: %w", err)
	}
	m.Network = ns
	defer func() {
		if retErr != nil {
			e.teardownNetwork(ctx, id, &ns)
		}
	}()
	if err := e.writeEtcFiles(e.cfg.containerDir(id), m); err != nil {
		return err
	}

	// Patch the spec with the network namespace path.
	spec, err := specFromAny(ctr.Spec)
	if err != nil {
		return err
	}
	setNetworkNamespace(spec, ns.NetnsPath, ns.Backend == networkHost)
	specAny, err := typeurl.MarshalAny(spec)
	if err != nil {
		return err
	}

	// I/O: the shim writes the container's output to FIFOs which the
	// logging helper reads; stdin is written to a FIFO by attaching
	// clients. The helper is started first and outlives this process.
	var stdinPath string
	if m.Config.OpenStdin {
		stdinPath = filepath.Join(runDir, "stdin")
		if err := unix.Mkfifo(stdinPath, 0o600); err != nil {
			return fmt.Errorf("creating stdin fifo: %w", err)
		}
	}
	stdoutPath := filepath.Join(runDir, "stdout")
	if err := unix.Mkfifo(stdoutPath, 0o600); err != nil {
		return fmt.Errorf("creating stdout fifo: %w", err)
	}
	// With a terminal there is a single output stream.
	var stderrPath string
	if !m.Config.Tty {
		stderrPath = filepath.Join(runDir, "stderr")
		if err := unix.Mkfifo(stderrPath, 0o600); err != nil {
			return fmt.Errorf("creating stderr fifo: %w", err)
		}
	}
	lc := &loggerConfig{
		ID:           id,
		Root:         e.cfg.root,
		RunDir:       e.cfg.runDir,
		Snapshotter:  e.cfg.snapshotter,
		LogPath:      m.LogPath,
		TTY:          m.Config.Tty,
		OpenStdin:    m.Config.OpenStdin,
		StdinOnce:    m.Config.StdinOnce,
		StdinFifo:    stdinPath,
		StdoutFifo:   stdoutPath,
		StderrFifo:   stderrPath,
		AttachSocket: filepath.Join(runDir, "attach.sock"),
		EventsSocket: e.cfg.eventsSocket(id),
		BundlePath:   filepath.Join(e.cfg.tasksStateDir(), namespace, id),
	}
	if err := e.startLogger(ctx, lc, runDir); err != nil {
		return err
	}

	rtOpts, err := e.runtimeOptions()
	if err != nil {
		return err
	}
	var task runtime.Task
	err = e.withTaskLock(func() error {
		return e.withShimLogRedirect(func() error {
			var err error
			task, err = tm.tm.Create(nsCtx(ctx), id, runtime.CreateOpts{
				Spec:           specAny,
				Rootfs:         mounts,
				IO:             runtime.IO{Stdin: stdinPath, Stdout: stdoutPath, Stderr: stderrPath, Terminal: m.Config.Tty},
				RuntimeOptions: rtOpts,
				Runtime:        shimRuntime,
			})
			return err
		})
	})
	if err != nil {
		return fmt.Errorf("creating container task: %w", translateRuntimeError(err))
	}
	// The shared manager predates this shim; make sure it is rediscovered.
	e.resetTasks()
	defer func() {
		if retErr != nil {
			_, _ = tm.deleteTask(ctx, id)
		}
	}()
	// Connect any client that attached before the container was started
	// (as "docker run" and "docker start --attach" do), so that no output
	// is missed.
	if err := e.dialPendingAttach(id); err != nil {
		return err
	}
	if err := task.Start(nsCtx(ctx)); err != nil {
		return fmt.Errorf("starting container: %w", translateRuntimeError(err))
	}
	e.setStartedTask(id, tm)
	pid, _ := task.PID(nsCtx(ctx))

	m.State = containerState{
		Status:       statusRunning,
		Pid:          int(pid),
		StartedAt:    time.Now().UTC(),
		RestartCount: m.State.RestartCount,
	}
	return e.withSession(ctx, func(ctx context.Context, s *session) error {
		cur, err := s.containers().Get(ctx, id)
		if err != nil {
			return err
		}
		return updateMeta(ctx, s.containers(), cur, m)
	})
}

// translateRuntimeError makes common shim/runtime failures readable.
func translateRuntimeError(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "executable file not found") || strings.Contains(msg, "no such file or directory") && strings.Contains(msg, "exec"):
		return fmt.Errorf("%w: %w", err, cerrdefs.ErrNotFound)
	case strings.Contains(msg, "containerd-shim-runc-v2"):
		return fmt.Errorf("%w (is containerd-shim-runc-v2 installed and in PATH?)", err)
	case strings.Contains(msg, "unknown version specified"):
		return fmt.Errorf("%w (the OCI runtime does not support this runtime-spec version; upgrade it or select another with %s=runc)", err, EnvRuntime)
	}
	return err
}

func specFromAny(a typeurl.Any) (*specs.Spec, error) {
	if a == nil {
		return nil, errors.New("container has no OCI spec")
	}
	var spec specs.Spec
	if err := json.Unmarshal(a.GetValue(), &spec); err != nil {
		return nil, fmt.Errorf("decoding OCI spec: %w", err)
	}
	return &spec, nil
}

// setNetworkNamespace points the spec's network namespace at the persisted
// path (or removes it for host networking).
func setNetworkNamespace(spec *specs.Spec, path string, host bool) {
	if spec.Linux == nil {
		spec.Linux = &specs.Linux{}
	}
	var nss []specs.LinuxNamespace
	for _, n := range spec.Linux.Namespaces {
		if n.Type != specs.NetworkNamespace {
			nss = append(nss, n)
		}
	}
	if !host {
		nss = append(nss, specs.LinuxNamespace{Type: specs.NetworkNamespace, Path: path})
	}
	spec.Linux.Namespaces = nss
}

// ---- state reconciliation

// syncState reconciles the recorded state with the live task, reaping tasks
// that have stopped. It returns the up-to-date metadata.
//
//nolint:gocyclo // one branch per task state
func (e *engine) syncState(ctx context.Context, tm *taskManager, s *session, ctr containers.Container, m *containerMeta) (*containerMeta, error) {
	task, err := tm.get(ctx, ctr.ID)
	if err != nil {
		if !cerrdefs.IsNotFound(err) {
			return m, err
		}
		if m.State.Status == statusRunning || m.State.Status == statusPaused {
			// The shim is gone (reboot, crash) and nobody recorded the exit.
			m.State.Status = statusExited
			m.State.Pid = 0
			if m.State.FinishedAt.IsZero() || m.State.FinishedAt.Before(m.State.StartedAt) {
				m.State.ExitCode = 255
				m.State.FinishedAt = time.Now().UTC()
				m.State.Error = "container exited without status (shim not found)"
			}
			e.teardownNetwork(ctx, ctr.ID, &m.Network)
			_ = os.RemoveAll(e.cfg.containerRunDir(ctr.ID))
			if err := updateMeta(ctx, s.containers(), ctr, m); err != nil {
				return m, err
			}
		}
		return m, nil
	}
	st, err := task.State(nsCtx(ctx))
	if err != nil {
		return m, nil
	}
	switch st.Status {
	case runtime.RunningStatus, runtime.CreatedStatus:
		if m.State.Status != statusRunning {
			m.State.Status = statusRunning
			m.State.Pid = int(st.Pid)
			if err := updateMeta(ctx, s.containers(), ctr, m); err != nil {
				return m, err
			}
		}
	case runtime.PausedStatus, runtime.PausingStatus:
		if m.State.Status != statusPaused {
			m.State.Status = statusPaused
			if err := updateMeta(ctx, s.containers(), ctr, m); err != nil {
				return m, err
			}
		}
	case runtime.DeletedStatus, runtime.StoppedStatus:
		// Reaping needs the session closed (the logger may hold the lock
		// briefly); do the bookkeeping directly here instead.
		exit, err := tm.deleteTask(ctx, ctr.ID)
		if err != nil && !cerrdefs.IsNotFound(err) {
			return m, err
		}
		m.State.Status = statusExited
		m.State.Pid = 0
		if exit != nil {
			m.State.ExitCode = int(exit.status)
			m.State.FinishedAt = exit.exitedAt
		} else {
			m.State.ExitCode = int(st.ExitStatus)
			m.State.FinishedAt = st.ExitedAt
		}
		e.teardownNetwork(ctx, ctr.ID, &m.Network)
		_ = os.RemoveAll(e.cfg.containerRunDir(ctr.ID))
		if err := updateMeta(ctx, s.containers(), ctr, m); err != nil {
			return m, err
		}
	}
	return m, nil
}

// reap deletes the task of a stopped container, records its exit status and
// releases its transient resources.
func (e *engine) reap(ctx context.Context, tm *taskManager, id string, m *containerMeta) error {
	exit, err := tm.deleteTask(ctx, id)
	if err != nil && !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("deleting task: %w", err)
	}
	e.teardownNetwork(ctx, id, &m.Network)
	_ = os.RemoveAll(e.cfg.containerRunDir(id))
	m.State.Status = statusExited
	m.State.Pid = 0
	if exit != nil {
		m.State.ExitCode = int(exit.status)
		m.State.FinishedAt = exit.exitedAt
	} else if m.State.FinishedAt.IsZero() {
		m.State.FinishedAt = time.Now().UTC()
	}
	return e.withSession(ctx, func(ctx context.Context, s *session) error {
		cur, err := s.containers().Get(ctx, id)
		if err != nil {
			return err
		}
		return updateMeta(ctx, s.containers(), cur, m)
	})
}

// loadSynced loads a container and reconciles its state.
func (e *engine) loadSynced(ctx context.Context, ref string) (containers.Container, *containerMeta, error) {
	tm, err := e.tasks(ctx)
	if err != nil {
		return containers.Container{}, nil, err
	}
	var (
		ctr containers.Container
		m   *containerMeta
	)
	err = e.withSession(ctx, func(ctx context.Context, s *session) error {
		var err error
		ctr, m, err = loadContainer(ctx, s.containers(), ref)
		if err != nil {
			return err
		}
		m, err = e.syncState(ctx, tm, s, ctr, m)
		return err
	})
	return ctr, m, err
}

// ---- stop / kill / pause

func (c *apiClient) ContainerStop(ctx context.Context, ref string, options client.ContainerStopOptions) (client.ContainerStopResult, error) {
	return client.ContainerStopResult{}, c.eng.stopContainer(ctx, ref, options.Signal, options.Timeout)
}

//nolint:gocyclo // signal handling with timeouts and escalation
func (e *engine) stopContainer(ctx context.Context, ref, signal string, timeout *int) error {
	ctr, m, err := e.loadSynced(ctx, ref)
	if err != nil {
		return err
	}
	if m.State.Status != statusRunning && m.State.Status != statusPaused {
		return nil
	}
	tm, err := e.taskManagerFor(ctx, ctr.ID)
	if err != nil {
		return err
	}
	task, err := tm.get(ctx, ctr.ID)
	if err != nil {
		return nil
	}
	if signal == "" {
		signal = stopSignal(m)
	}
	sig, err := parseSignal(signal)
	if err != nil {
		return err
	}
	if m.State.Status == statusPaused {
		_ = task.Resume(nsCtx(ctx))
	}
	waitCh := make(chan *taskExit, 1)
	go func() {
		exit, err := waitTask(context.WithoutCancel(ctx), task)
		if err != nil {
			exit = nil
		}
		waitCh <- exit
	}()
	if err := killTask(ctx, task, sig, false); err != nil && !cerrdefs.IsNotFound(err) && !errors.Is(err, cerrdefs.ErrConflict) {
		return err
	}
	secs := stopTimeout(m, timeout)
	if secs >= 0 {
		select {
		case <-waitCh:
		case <-time.After(time.Duration(secs) * time.Second):
			_ = killTask(ctx, task, 9, true)
			select {
			case <-waitCh:
			case <-time.After(10 * time.Second):
				return errors.New("container did not exit after SIGKILL")
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	} else {
		select {
		case <-waitCh:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return e.reap(ctx, tm, ctr.ID, m)
}

func (c *apiClient) ContainerKill(ctx context.Context, ref string, options client.ContainerKillOptions) (client.ContainerKillResult, error) {
	ctr, m, err := c.eng.loadSynced(ctx, ref)
	if err != nil {
		return client.ContainerKillResult{}, err
	}
	if m.State.Status != statusRunning && m.State.Status != statusPaused {
		return client.ContainerKillResult{}, fmt.Errorf("cannot kill container %s: container is not running: %w", shortID(ctr.ID), cerrdefs.ErrConflict)
	}
	sig, err := parseSignal(options.Signal)
	if err != nil {
		return client.ContainerKillResult{}, err
	}
	if options.Signal == "" {
		sig = 9
	}
	tm, err := c.eng.taskManagerFor(ctx, ctr.ID)
	if err != nil {
		return client.ContainerKillResult{}, err
	}
	task, err := tm.get(ctx, ctr.ID)
	if err != nil {
		return client.ContainerKillResult{}, fmt.Errorf("container is not running: %w", cerrdefs.ErrConflict)
	}
	return client.ContainerKillResult{}, killTask(ctx, task, sig, false)
}

func (c *apiClient) ContainerRestart(ctx context.Context, ref string, options client.ContainerRestartOptions) (client.ContainerRestartResult, error) {
	if err := c.eng.stopContainer(ctx, ref, options.Signal, options.Timeout); err != nil {
		return client.ContainerRestartResult{}, err
	}
	err := c.eng.withSession(ctx, func(ctx context.Context, s *session) error {
		ctr, m, err := loadContainer(ctx, s.containers(), ref)
		if err != nil {
			return err
		}
		m.State.RestartCount++
		return updateMeta(ctx, s.containers(), ctr, m)
	})
	if err != nil {
		return client.ContainerRestartResult{}, err
	}
	return client.ContainerRestartResult{}, c.eng.startContainer(ctx, ref)
}

func (c *apiClient) ContainerPause(ctx context.Context, ref string, _ client.ContainerPauseOptions) (client.ContainerPauseResult, error) {
	return client.ContainerPauseResult{}, c.eng.pauseContainer(ctx, ref, true)
}

func (c *apiClient) ContainerUnpause(ctx context.Context, ref string, _ client.ContainerUnpauseOptions) (client.ContainerUnpauseResult, error) {
	return client.ContainerUnpauseResult{}, c.eng.pauseContainer(ctx, ref, false)
}

func (e *engine) pauseContainer(ctx context.Context, ref string, pause bool) error {
	ctr, m, err := e.loadSynced(ctx, ref)
	if err != nil {
		return err
	}
	tm, err := e.taskManagerFor(ctx, ctr.ID)
	if err != nil {
		return err
	}
	task, err := tm.get(ctx, ctr.ID)
	if err != nil {
		return fmt.Errorf("container %s is not running: %w", shortID(ctr.ID), cerrdefs.ErrConflict)
	}
	if pause {
		if m.State.Status == statusPaused {
			return fmt.Errorf("container %s is already paused: %w", shortID(ctr.ID), cerrdefs.ErrConflict)
		}
		if err := task.Pause(nsCtx(ctx)); err != nil {
			return err
		}
		m.State.Status = statusPaused
	} else {
		if m.State.Status != statusPaused {
			return fmt.Errorf("container %s is not paused: %w", shortID(ctr.ID), cerrdefs.ErrConflict)
		}
		if err := task.Resume(nsCtx(ctx)); err != nil {
			return err
		}
		m.State.Status = statusRunning
	}
	return e.withSession(ctx, func(ctx context.Context, s *session) error {
		cur, err := s.containers().Get(ctx, ctr.ID)
		if err != nil {
			return err
		}
		return updateMeta(ctx, s.containers(), cur, m)
	})
}

// ---- wait

func (c *apiClient) ContainerWait(ctx context.Context, ref string, options client.ContainerWaitOptions) client.ContainerWaitResult {
	resultC := make(chan container.WaitResponse, 1)
	errC := make(chan error, 1)
	go func() {
		code, err := c.eng.waitContainer(ctx, ref, options.Condition)
		if err != nil {
			errC <- err
			return
		}
		resultC <- container.WaitResponse{StatusCode: code}
	}()
	return client.ContainerWaitResult{Result: resultC, Error: errC}
}

//nolint:gocyclo // waits for a task that may not exist yet or already be gone
func (e *engine) waitContainer(ctx context.Context, ref string, cond container.WaitCondition) (int64, error) {
	ctr, m, err := e.loadSynced(ctx, ref)
	if err != nil {
		return 0, err
	}
	id := ctr.ID
	stopped := m.State.Status != statusRunning && m.State.Status != statusPaused
	if cond == container.WaitConditionNotRunning && stopped {
		return int64(m.State.ExitCode), nil
	}
	autoRemove := m.HostConfig.AutoRemove || cond == container.WaitConditionRemoved

	// The container may not have been started yet ("docker run" waits before
	// it starts the container), and the task may also be gone already (it
	// exited and was reaped, in which case the recorded state holds the exit
	// status). Poll for whichever happens first.
	var (
		code       int64
		lastReload = time.Now()
	)
	for {
		tm, err := e.taskManagerFor(ctx, id)
		if err != nil {
			return 0, err
		}
		if task, err := tm.get(ctx, id); err == nil {
			exit, werr := waitTask(ctx, task)
			if werr == nil {
				code = int64(exit.status)
				break
			}
			if !cerrdefs.IsNotFound(werr) {
				return 0, werr
			}
			// The task disappeared while waiting: fall back to the
			// recorded state below.
		}
		_, m, err = e.loadSynced(ctx, ref)
		if err != nil {
			if cerrdefs.IsNotFound(err) {
				// Removed by another process (or by us, on auto-remove).
				return code, nil
			}
			return 0, err
		}
		if m.State.Status == statusExited && !m.State.FinishedAt.IsZero() {
			code = int64(m.State.ExitCode)
			break
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
		if time.Since(lastReload) > time.Second {
			// Pick up shims started by other processes.
			e.resetTasks()
			lastReload = time.Now()
		}
	}
	e.forgetStartedTask(id)

	if autoRemove {
		if err := e.removeContainer(context.WithoutCancel(ctx), id, true, true); err != nil && !cerrdefs.IsNotFound(err) {
			log.G(ctx).WithError(err).Warn("removing container")
		}
		return code, nil
	}
	if tm, err := e.tasks(ctx); err == nil {
		if err := e.reap(context.WithoutCancel(ctx), tm, id, m); err != nil {
			log.G(ctx).WithError(err).Debug("reaping container")
		}
	}
	return code, nil
}

// ---- remove

func (c *apiClient) ContainerRemove(ctx context.Context, ref string, options client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
	return client.ContainerRemoveResult{}, c.eng.removeContainer(ctx, ref, options.Force, options.RemoveVolumes)
}

//nolint:gocyclo // removal has to clean up every resource of a container
func (e *engine) removeContainer(ctx context.Context, ref string, force, removeVolumes bool) error {
	ctr, m, err := e.loadSynced(ctx, ref)
	if err != nil {
		return err
	}
	tm, err := e.taskManagerFor(ctx, ctr.ID)
	if err != nil {
		return err
	}
	if m.State.Status == statusRunning || m.State.Status == statusPaused {
		if !force {
			return fmt.Errorf("cannot remove container %q: container is running: stop the container before removing or force remove: %w", m.Name, cerrdefs.ErrConflict)
		}
		if task, err := tm.get(ctx, ctr.ID); err == nil {
			waitCh := make(chan struct{})
			go func() {
				_, _ = waitTask(context.WithoutCancel(ctx), task)
				close(waitCh)
			}()
			_ = killTask(ctx, task, 9, true)
			select {
			case <-waitCh:
			case <-time.After(10 * time.Second):
			}
		}
	}
	if _, err := tm.get(ctx, ctr.ID); err == nil {
		if _, err := tm.deleteTask(ctx, ctr.ID); err != nil && !cerrdefs.IsNotFound(err) {
			return fmt.Errorf("deleting task: %w", err)
		}
	}
	e.teardownNetwork(ctx, ctr.ID, &m.Network)
	_ = os.RemoveAll(e.cfg.containerRunDir(ctr.ID))

	err = e.withSession(ctx, func(ctx context.Context, s *session) error {
		if err := s.snapshotter().Remove(ctx, ctr.SnapshotKey); err != nil && !cerrdefs.IsNotFound(err) {
			return fmt.Errorf("removing root filesystem: %w", err)
		}
		if err := s.containers().Delete(ctx, ctr.ID); err != nil && !cerrdefs.IsNotFound(err) {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	e.forgetStartedTask(ctr.ID)
	_ = os.RemoveAll(e.cfg.containerDir(ctr.ID))
	if removeVolumes {
		for _, v := range m.AnonymousVolumes {
			_ = e.removeVolume(v)
		}
	}
	return nil
}

func (c *apiClient) ContainerPrune(ctx context.Context, opts client.ContainerPruneOptions) (client.ContainerPruneResult, error) {
	var res client.ContainerPruneResult
	list, err := c.ContainerList(ctx, client.ContainerListOptions{All: true, Filters: opts.Filters})
	if err != nil {
		return res, err
	}
	for _, sum := range list.Items {
		if sum.State == container.StateRunning || sum.State == container.StatePaused {
			continue
		}
		if err := c.eng.removeContainer(ctx, sum.ID, false, false); err != nil {
			return res, err
		}
		res.Report.ContainersDeleted = append(res.Report.ContainersDeleted, sum.ID)
	}
	return res, nil
}

// ---- rename

func (c *apiClient) ContainerRename(ctx context.Context, ref string, options client.ContainerRenameOptions) (client.ContainerRenameResult, error) {
	newName := strings.TrimPrefix(options.NewName, "/")
	if err := validateName(newName); err != nil {
		return client.ContainerRenameResult{}, err
	}
	err := c.eng.withSession(ctx, func(ctx context.Context, s *session) error {
		ctr, m, err := loadContainer(ctx, s.containers(), ref)
		if err != nil {
			return err
		}
		if other, _, err := loadContainer(ctx, s.containers(), newName); err == nil && other.ID != ctr.ID {
			return fmt.Errorf("the container name %q is already in use by container %q: %w", newName, shortID(other.ID), cerrdefs.ErrConflict)
		}
		m.Name = newName
		return updateMeta(ctx, s.containers(), ctr, m)
	})
	return client.ContainerRenameResult{}, err
}

// ---- list / inspect

func (c *apiClient) ContainerList(ctx context.Context, options client.ContainerListOptions) (client.ContainerListResult, error) {
	tm, err := c.eng.tasks(ctx)
	if err != nil {
		return client.ContainerListResult{}, err
	}
	res := client.ContainerListResult{Items: []container.Summary{}}
	err = c.eng.withSession(ctx, func(ctx context.Context, s *session) error {
		all, err := s.containers().List(ctx)
		if err != nil {
			return err
		}
		type entry struct {
			ctr containers.Container
			m   *containerMeta
		}
		var entries []entry
		for _, ctr := range all {
			m, err := unmarshalMeta(ctr)
			if err != nil {
				continue
			}
			m, err = c.eng.syncState(ctx, tm, s, ctr, m)
			if err != nil {
				return err
			}
			entries = append(entries, entry{ctr, m})
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].m.Created.After(entries[j].m.Created) })
		for _, en := range entries {
			running := en.m.State.Status == statusRunning || en.m.State.Status == statusPaused
			if !options.All && !running {
				continue
			}
			if !matchContainerFilters(options.Filters, en.ctr, en.m) {
				continue
			}
			res.Items = append(res.Items, c.eng.summary(en.ctr, en.m))
			if options.Limit > 0 && len(res.Items) >= options.Limit {
				break
			}
		}
		return nil
	})
	return res, err
}

//nolint:gocyclo // one branch per supported filter
func matchContainerFilters(f client.Filters, ctr containers.Container, m *containerMeta) bool {
	for key, values := range f {
		switch key {
		case "status":
			if _, ok := values[m.State.Status]; !ok {
				return false
			}
		case "name":
			ok := false
			for v := range values {
				if strings.Contains(m.Name, strings.TrimPrefix(v, "/")) {
					ok = true
				}
			}
			if !ok {
				return false
			}
		case "id":
			ok := false
			for v := range values {
				if strings.HasPrefix(ctr.ID, v) {
					ok = true
				}
			}
			if !ok {
				return false
			}
		case "label":
			for v := range values {
				k, val, hasVal := strings.Cut(v, "=")
				lv, ok := m.Config.Labels[k]
				if !ok || (hasVal && lv != val) {
					return false
				}
			}
		case "ancestor":
			ok := false
			for v := range values {
				if v == m.Config.Image || v == m.ImageID || familiarizeImageRef(m.ImageRef) == v || strings.HasPrefix(strings.TrimPrefix(m.ImageID, "sha256:"), strings.TrimPrefix(v, "sha256:")) {
					ok = true
				}
			}
			if !ok {
				return false
			}
		case "exited":
			if m.State.Status != statusExited {
				return false
			}
			if _, ok := values[strconv.Itoa(m.State.ExitCode)]; !ok {
				return false
			}
		case "network":
			if _, ok := values[networkModeName(m.HostConfig)]; !ok {
				return false
			}
		case "volume":
			ok := false
			for _, mp := range m.Mounts {
				if _, has := values[mp.Name]; has {
					ok = true
				}
				if _, has := values[mp.Destination]; has {
					ok = true
				}
			}
			if !ok {
				return false
			}
		}
	}
	return true
}

func (*engine) statusString(m *containerMeta) string {
	switch m.State.Status {
	case statusRunning:
		return "Up " + units.HumanDuration(time.Since(m.State.StartedAt))
	case statusPaused:
		return "Up " + units.HumanDuration(time.Since(m.State.StartedAt)) + " (Paused)"
	case statusExited:
		if m.State.FinishedAt.IsZero() {
			return fmt.Sprintf("Exited (%d)", m.State.ExitCode)
		}
		return fmt.Sprintf("Exited (%d) %s ago", m.State.ExitCode, units.HumanDuration(time.Since(m.State.FinishedAt)))
	default:
		return "Created"
	}
}

func (e *engine) summary(ctr containers.Container, m *containerMeta) container.Summary {
	cmd := append([]string{m.Path}, m.Args...)
	sum := container.Summary{
		ID:      ctr.ID,
		Names:   []string{"/" + m.Name},
		Image:   m.Config.Image,
		ImageID: m.ImageID,
		Command: strings.Join(cmd, " "),
		Created: m.Created.Unix(),
		Labels:  m.Config.Labels,
		State:   container.ContainerState(m.State.Status),
		Status:  e.statusString(m),
		Mounts:  m.Mounts,
		NetworkSettings: &container.NetworkSettingsSummary{
			Networks: map[string]*network.EndpointSettings{networkModeName(m.HostConfig): e.endpointSettings(m)},
		},
	}
	sum.HostConfig.NetworkMode = networkModeName(m.HostConfig)
	sum.HostConfig.Annotations = m.HostConfig.Annotations
	sum.Ports = e.portSummaries(m)
	return sum
}

func (*engine) portSummaries(m *containerMeta) []container.PortSummary {
	var out []container.PortSummary
	for p := range m.Config.ExposedPorts {
		bindings := m.Network.PortMap[p]
		if len(bindings) == 0 {
			out = append(out, container.PortSummary{PrivatePort: p.Num(), Type: string(p.Proto())})
			continue
		}
		for _, b := range bindings {
			ps := container.PortSummary{PrivatePort: p.Num(), Type: string(p.Proto()), IP: b.HostIP}
			if !ps.IP.IsValid() {
				ps.IP = netip.IPv4Unspecified()
			}
			if n, err := strconv.ParseUint(b.HostPort, 10, 16); err == nil {
				ps.PublicPort = uint16(n)
			}
			out = append(out, ps)
		}
	}
	for p, bindings := range m.Network.PortMap {
		if _, exposed := m.Config.ExposedPorts[p]; exposed {
			continue
		}
		for _, b := range bindings {
			ps := container.PortSummary{PrivatePort: p.Num(), Type: string(p.Proto()), IP: b.HostIP}
			if !ps.IP.IsValid() {
				ps.IP = netip.IPv4Unspecified()
			}
			if n, err := strconv.ParseUint(b.HostPort, 10, 16); err == nil {
				ps.PublicPort = uint16(n)
			}
			out = append(out, ps)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PrivatePort != out[j].PrivatePort {
			return out[i].PrivatePort < out[j].PrivatePort
		}
		return out[i].Type < out[j].Type
	})
	return out
}

func (*engine) endpointSettings(m *containerMeta) *network.EndpointSettings {
	ep := &network.EndpointSettings{
		NetworkID:  "standalone-" + networkModeName(m.HostConfig),
		EndpointID: "",
	}
	if m.State.Status != statusRunning && m.State.Status != statusPaused {
		return ep
	}
	if ip, err := netip.ParseAddr(m.Network.IPAddress); err == nil {
		ep.IPAddress = ip
		ep.IPPrefixLen = m.Network.IPPrefix
	}
	if gw, err := netip.ParseAddr(m.Network.Gateway); err == nil {
		ep.Gateway = gw
	}
	if m.Network.MacAddress != "" {
		if mac, err := net.ParseMAC(m.Network.MacAddress); err == nil {
			ep.MacAddress = network.HardwareAddr(mac)
		}
	}
	return ep
}

func (c *apiClient) ContainerInspect(ctx context.Context, ref string, _ client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	ctr, m, err := c.eng.loadSynced(ctx, ref)
	if err != nil {
		return client.ContainerInspectResult{}, err
	}
	insp := c.eng.inspect(ctr, m)
	raw, err := json.Marshal(insp)
	if err != nil {
		return client.ContainerInspectResult{}, err
	}
	return client.ContainerInspectResult{Container: insp, Raw: raw}, nil
}

func (e *engine) inspect(ctr containers.Container, m *containerMeta) container.InspectResponse {
	st := &container.State{
		Status:     container.ContainerState(m.State.Status),
		Running:    m.State.Status == statusRunning || m.State.Status == statusPaused,
		Paused:     m.State.Status == statusPaused,
		OOMKilled:  m.State.OOMKilled,
		Pid:        m.State.Pid,
		ExitCode:   m.State.ExitCode,
		Error:      m.State.Error,
		StartedAt:  formatTime(m.State.StartedAt),
		FinishedAt: formatTime(m.State.FinishedAt),
	}
	netMode := networkModeName(m.HostConfig)
	ns := &container.NetworkSettings{
		Networks: map[string]*network.EndpointSettings{netMode: e.endpointSettings(m)},
	}
	ns.Ports = network.PortMap{}
	for p := range m.Config.ExposedPorts {
		ns.Ports[p] = nil
	}
	for p, b := range m.Network.PortMap {
		ns.Ports[p] = b
	}
	ns.SandboxKey = m.Network.NetnsPath
	hc := *m.HostConfig
	return container.InspectResponse{
		ID:              ctr.ID,
		Created:         formatTime(m.Created),
		Path:            m.Path,
		Args:            m.Args,
		State:           st,
		Image:           m.ImageID,
		ResolvConfPath:  m.ResolvConfPath,
		HostnamePath:    m.HostnamePath,
		HostsPath:       m.HostsPath,
		LogPath:         m.LogPath,
		Name:            "/" + m.Name,
		RestartCount:    m.State.RestartCount,
		Driver:          e.cfg.snapshotter,
		Platform:        "linux",
		HostConfig:      &hc,
		Mounts:          m.Mounts,
		Config:          m.Config,
		NetworkSettings: ns,
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "0001-01-01T00:00:00Z"
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// ---- misc

func (c *apiClient) ContainerResize(ctx context.Context, ref string, options client.ContainerResizeOptions) (client.ContainerResizeResult, error) {
	ctr, _, err := c.eng.loadSynced(ctx, ref)
	if err != nil {
		return client.ContainerResizeResult{}, err
	}
	tm, err := c.eng.tasks(ctx)
	if err != nil {
		return client.ContainerResizeResult{}, err
	}
	task, err := tm.get(ctx, ctr.ID)
	if err != nil {
		return client.ContainerResizeResult{}, fmt.Errorf("container is not running: %w", cerrdefs.ErrConflict)
	}
	return client.ContainerResizeResult{}, task.ResizePty(nsCtx(ctx), runtime.ConsoleSize{Width: uint32(options.Width), Height: uint32(options.Height)})
}

// pidsInNamespaceOf returns the host PIDs that share the PID namespace of
// the given process, including itself.
func pidsInNamespaceOf(pid int) ([]int, error) {
	if pid <= 0 {
		return nil, errors.New("container has no main process")
	}
	want, err := os.Readlink(fmt.Sprintf("/proc/%d/ns/pid", pid))
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var out []int
	for _, e := range entries {
		p, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		if ns, err := os.Readlink(fmt.Sprintf("/proc/%d/ns/pid", p)); err == nil && ns == want {
			out = append(out, p)
		}
	}
	sort.Ints(out)
	return out, nil
}

//nolint:gocyclo // several fallbacks for listing processes
func (c *apiClient) ContainerTop(ctx context.Context, ref string, options client.ContainerTopOptions) (client.ContainerTopResult, error) {
	ctr, m, err := c.eng.loadSynced(ctx, ref)
	if err != nil {
		return client.ContainerTopResult{}, err
	}
	if m.State.Status != statusRunning && m.State.Status != statusPaused {
		return client.ContainerTopResult{}, fmt.Errorf("container %s is not running: %w", shortID(ctr.ID), cerrdefs.ErrConflict)
	}
	tm, err := c.eng.tasks(ctx)
	if err != nil {
		return client.ContainerTopResult{}, err
	}
	task, err := tm.get(ctx, ctr.ID)
	if err != nil {
		return client.ContainerTopResult{}, fmt.Errorf("container is not running: %w", cerrdefs.ErrConflict)
	}
	var pids []string
	procs, err := task.Pids(nsCtx(ctx))
	if err == nil {
		for _, p := range procs {
			pids = append(pids, strconv.Itoa(int(p.Pid)))
		}
	} else {
		// Listing a container's processes uses its cgroup, which does not
		// exist when running rootless without cgroup delegation. Fall back
		// to the processes sharing the container's PID namespace, and to
		// the main process alone if even that is not permitted (each CLI
		// invocation runs in its own user namespace, which may not be
		// allowed to inspect the container's).
		nsPids, nsErr := pidsInNamespaceOf(m.State.Pid)
		if nsErr != nil {
			if m.State.Pid <= 0 {
				return client.ContainerTopResult{}, err
			}
			nsPids = []int{m.State.Pid}
		}
		for _, p := range nsPids {
			pids = append(pids, strconv.Itoa(p))
		}
	}
	if len(pids) == 0 {
		return client.ContainerTopResult{Titles: []string{"PID"}}, nil
	}
	psArgs := options.Arguments
	if len(psArgs) == 0 {
		psArgs = []string{"-o", "pid,ppid,user,etime,cmd"}
	}
	args := append(append([]string{}, psArgs...), "-p", strings.Join(pids, ","))
	out, err := exec.CommandContext(ctx, "ps", args...).Output()
	if err != nil {
		return client.ContainerTopResult{}, fmt.Errorf("running ps: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return client.ContainerTopResult{}, nil
	}
	res := client.ContainerTopResult{Titles: strings.Fields(lines[0])}
	n := len(res.Titles)
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) > n {
			fields = append(fields[:n-1], strings.Join(fields[n-1:], " "))
		}
		res.Processes = append(res.Processes, fields)
	}
	return res, nil
}
