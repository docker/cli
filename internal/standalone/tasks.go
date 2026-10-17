//go:build linux

package standalone

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	runcoptions "github.com/containerd/containerd/api/types/runc/options"
	"github.com/containerd/containerd/v2/core/events/exchange"
	"github.com/containerd/containerd/v2/core/metadata"
	"github.com/containerd/containerd/v2/core/runtime"
	v2 "github.com/containerd/containerd/v2/core/runtime/v2"
	"github.com/containerd/containerd/v2/core/sandbox"
	cerrdefs "github.com/containerd/errdefs"
	"github.com/containerd/typeurl/v2"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// taskManager wraps containerd's runtime v2 task manager. Constructing it
// discovers and reconnects to the shims of all running containers, so it is
// created lazily and once per CLI invocation.
type taskManager struct {
	tm *v2.TaskManager
}

// containerdAddress is the (non-existent) gRPC address handed to shims. It
// is only used to derive shim socket names and by legacy publishing paths.
func (c *config) containerdAddress() string {
	return filepath.Join(c.runDir, "containerd.sock")
}

// eventsSocket is the per-container ttrpc socket on which the logging helper
// receives the shim's events (TaskCreate, TaskStart, TaskExit, ...).
func (c *config) eventsSocket(id string) string {
	return filepath.Join(c.containerRunDir(id), "events.sock")
}

// withTaskLock serializes shim discovery and creation across CLI processes.
// containerd's shim loader assumes a single daemon owns the state directory:
// it deletes bundles it cannot connect to, which would destroy a bundle that
// another process is in the middle of creating.
func (e *engine) withTaskLock(fn func() error) error {
	if err := os.MkdirAll(e.cfg.runDir, 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(e.cfg.runDir, "tasks.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		return fmt.Errorf("locking task state: %w", err)
	}
	defer func() { _ = unix.Flock(int(f.Fd()), unix.LOCK_UN) }()
	return fn()
}

// withShimLogRedirect runs fn with os.Stderr pointing at the engine's shim
// log file.
//
// containerd copies each shim's log pipe to os.Stderr, which is right for a
// daemon but not for a CLI: shims report conditions that are expected here
// (for example that cgroups are unavailable when running rootless without
// delegation) and would be printed in the middle of a command's output. The
// copy goroutines capture os.Stderr when they start, so redirecting it
// around the calls that start or reconnect to shims is enough. The CLI's own
// streams are unaffected: they were captured before this point.
//
// With debug logging enabled the shim logs are left on stderr.
func (e *engine) withShimLogRedirect(fn func() error) error {
	if logrus.GetLevel() >= logrus.DebugLevel {
		return fn()
	}
	f, err := os.OpenFile(filepath.Join(e.cfg.runDir, "shim.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fn()
	}
	defer f.Close()
	saved := os.Stderr
	os.Stderr = f
	defer func() { os.Stderr = saved }()
	return fn()
}

// newTaskManager builds a shim manager whose shims publish events to the
// given ttrpc address, and loads the existing shims.
func (e *engine) newTaskManager(ctx context.Context, ttrpcAddress string) (tm *taskManager, err error) {
	err = e.withTaskLock(func() error {
		return e.withShimLogRedirect(func() error {
			tm, err = e.newTaskManagerLocked(ctx, ttrpcAddress)
			return err
		})
	})
	return tm, err
}

func (e *engine) newTaskManagerLocked(ctx context.Context, ttrpcAddress string) (*taskManager, error) {
	configureLogging()
	for _, d := range []string{e.cfg.shimSocketDir(), e.cfg.tasksRootDir(), e.cfg.tasksStateDir()} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return nil, err
		}
	}
	store := lazyContainerStore{eng: e}
	sm, err := v2.NewShimManager(&v2.ManagerConfig{
		Store:        store,
		Events:       exchange.NewExchange(),
		Address:      e.cfg.containerdAddress(),
		TTRPCAddress: ttrpcAddress,
		SocketDir:    e.cfg.shimSocketDir(),
		SandboxStore: lazySandboxStore{eng: e},
	})
	if err != nil {
		return nil, err
	}
	tm, err := v2.NewTaskManager(nsCtx(ctx), e.cfg.tasksRootDir(), e.cfg.tasksStateDir(), sm)
	if err != nil {
		return nil, fmt.Errorf("loading container tasks: %w", err)
	}
	return &taskManager{tm: tm}, nil
}

// tasks returns the shared task manager used for operations on existing
// containers. Shims are discovered when the manager is first created; call
// resetTasks after starting a container (or when waiting for one started by
// another process) so that new shims are picked up.
func (e *engine) tasks(ctx context.Context) (*taskManager, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.taskMgr != nil {
		return e.taskMgr, nil
	}
	tm, err := e.newTaskManager(ctx, e.cfg.containerdAddress()+".ttrpc")
	if err != nil {
		return nil, err
	}
	e.taskMgr = tm
	return tm, nil
}

// resetTasks discards the shared task manager so that the next call to
// tasks rediscovers the running shims.
func (e *engine) resetTasks() {
	e.mu.Lock()
	e.taskMgr = nil
	e.mu.Unlock()
}

// setStartedTask remembers the manager that started a container. Its shim is
// registered with that manager, so operations on the container (waiting for
// it, stopping it) can use it directly instead of rediscovering the shim.
// Rediscovery goes through containerd's shim loader, which reaps shims of
// containers that have already exited and discards their exit status.
func (e *engine) setStartedTask(id string, tm *taskManager) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.startedTasks == nil {
		e.startedTasks = map[string]*taskManager{}
	}
	e.startedTasks[id] = tm
}

func (e *engine) forgetStartedTask(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.startedTasks, id)
}

// taskManagerFor returns the manager to use for a container: the one that
// started it in this process when available, else the shared one.
func (e *engine) taskManagerFor(ctx context.Context, id string) (*taskManager, error) {
	e.mu.Lock()
	tm := e.startedTasks[id]
	e.mu.Unlock()
	if tm != nil {
		return tm, nil
	}
	return e.tasks(ctx)
}

// get returns the task for the container, or ErrNotFound when the container
// has no running shim.
func (t *taskManager) get(ctx context.Context, id string) (runtime.Task, error) {
	return t.tm.Get(nsCtx(ctx), id)
}

// runtimeOptions returns the runc shim options selecting the OCI runtime
// binary and its state root.
func (e *engine) runtimeOptions() (typeurl.Any, error) {
	if err := os.MkdirAll(e.cfg.runcRoot(), 0o700); err != nil {
		return nil, err
	}
	return typeurl.MarshalAny(&runcoptions.Options{
		BinaryName:    e.cfg.runtimeBinary,
		Root:          e.cfg.runcRoot(),
		SystemdCgroup: e.cfg.systemdCgroup,
	})
}

// taskExit is the recorded result of a task that has stopped.
type taskExit struct {
	status   uint32
	exitedAt time.Time
}

// deleteTask deletes the task (stopping the shim and unmounting the rootfs)
// and returns its exit status.
func (t *taskManager) deleteTask(ctx context.Context, id string) (*taskExit, error) {
	exit, err := t.tm.Delete(nsCtx(ctx), id)
	if err != nil {
		return nil, err
	}
	return &taskExit{status: exit.Status, exitedAt: exit.Timestamp}, nil
}

// waitTask blocks until the task exits.
func waitTask(ctx context.Context, task runtime.Task) (*taskExit, error) {
	exit, err := task.Wait(ctx)
	if err != nil {
		return nil, err
	}
	return &taskExit{status: exit.Status, exitedAt: exit.Timestamp}, nil
}

// killTask sends a signal to the container's init process (or all processes).
func killTask(ctx context.Context, task runtime.Task, sig int, all bool) error {
	err := task.Kill(nsCtx(ctx), uint32(sig), all)
	if err != nil && cerrdefs.IsNotFound(err) {
		return fmt.Errorf("container is not running: %w", cerrdefs.ErrConflict)
	}
	return err
}

// recordExit stores the exit status of a container in its metadata. It is
// used by whichever process observes the exit first.
func recordExit(ctx context.Context, cfg *config, id string, status uint32, exitedAt time.Time, oom bool) error {
	s, err := openSession(ctx, cfg)
	if err != nil {
		return err
	}
	defer s.Close()
	ctx = nsCtx(ctx)
	store := s.containers()
	c, err := store.Get(ctx, id)
	if err != nil {
		return err
	}
	m, err := unmarshalMeta(c)
	if err != nil {
		return err
	}
	if m.State.Status == statusExited && !m.State.FinishedAt.IsZero() && m.State.FinishedAt.After(m.State.StartedAt) {
		// Already recorded by another observer.
		return nil
	}
	m.State.Status = statusExited
	m.State.ExitCode = int(status)
	m.State.FinishedAt = exitedAt
	m.State.Pid = 0
	if oom {
		m.State.OOMKilled = true
	}
	return updateMeta(ctx, store, c, m)
}

// lazySandboxStore satisfies the shim manager's sandbox store requirement.
// Sandboxes are not used by this backend, so lookups always fail with
// ErrNotFound (which is what the shim loader expects for task shims).
type lazySandboxStore struct {
	eng *engine
}

func (l lazySandboxStore) with(ctx context.Context, fn func(ctx context.Context, s *session) error) error {
	return l.eng.withSession(ctx, fn)
}

func (l lazySandboxStore) Get(ctx context.Context, id string) (out sandbox.Sandbox, err error) {
	err = l.with(ctx, func(ctx context.Context, s *session) error {
		out, err = metadata.NewSandboxStore(s.db).Get(ctx, id)
		return err
	})
	return out, err
}

func (l lazySandboxStore) List(ctx context.Context, filters ...string) (out []sandbox.Sandbox, err error) {
	err = l.with(ctx, func(ctx context.Context, s *session) error {
		out, err = metadata.NewSandboxStore(s.db).List(ctx, filters...)
		return err
	})
	return out, err
}

func (l lazySandboxStore) Create(ctx context.Context, sb sandbox.Sandbox) (out sandbox.Sandbox, err error) {
	err = l.with(ctx, func(ctx context.Context, s *session) error {
		out, err = metadata.NewSandboxStore(s.db).Create(ctx, sb)
		return err
	})
	return out, err
}

func (l lazySandboxStore) Update(ctx context.Context, sb sandbox.Sandbox, fieldpaths ...string) (out sandbox.Sandbox, err error) {
	err = l.with(ctx, func(ctx context.Context, s *session) error {
		out, err = metadata.NewSandboxStore(s.db).Update(ctx, sb, fieldpaths...)
		return err
	})
	return out, err
}

func (l lazySandboxStore) Delete(ctx context.Context, id string) error {
	return l.with(ctx, func(ctx context.Context, s *session) error {
		return metadata.NewSandboxStore(s.db).Delete(ctx, id)
	})
}
