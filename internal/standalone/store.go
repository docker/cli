//go:build linux

package standalone

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/containerd/containerd/v2/core/containers"
	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/core/leases"
	"github.com/containerd/containerd/v2/core/metadata"
	"github.com/containerd/containerd/v2/core/snapshots"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/containerd/v2/plugins/content/local"
	"github.com/containerd/containerd/v2/plugins/snapshots/native"
	"github.com/containerd/containerd/v2/plugins/snapshots/overlay"
	"github.com/containerd/log"
	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

// session is a short-lived handle on the containerd stores. The metadata
// database (bbolt) holds an exclusive file lock while open, so sessions are
// opened per operation and closed promptly to allow concurrent CLI
// invocations (including the per-container logging helpers) to access the
// stores.
type session struct {
	cfg  *config
	bdb  *bolt.DB
	db   *metadata.DB
	snRw snapshots.Snapshotter
}

// nsCtx returns ctx with the containerd namespace used by the engine.
func nsCtx(ctx context.Context) context.Context {
	return namespaces.WithNamespace(ctx, namespace)
}

// configureLogging routes the containerd libraries' log output to a
// dedicated logger. The libraries are chatty at info/warning level for
// conditions that are irrelevant to a CLI (for example missing fsverity
// support), so only errors are shown unless the CLI runs with debug logging.
var configureLogging = sync.OnceFunc(func() {
	l := logrus.New()
	l.SetOutput(os.Stderr)
	l.SetFormatter(&filteredFormatter{Formatter: logrus.StandardLogger().Formatter})
	if logrus.GetLevel() >= logrus.DebugLevel {
		l.SetLevel(logrus.DebugLevel)
	} else {
		l.SetLevel(logrus.ErrorLevel)
	}
	log.L = logrus.NewEntry(l)
})

// contentStores caches the local content store per root: it holds no locks,
// and creating it probes the filesystem (logging a warning when fsverity is
// unavailable) which need not be repeated for every session.
var contentStores sync.Map

func contentStore(root string) (content.Store, error) {
	if cs, ok := contentStores.Load(root); ok {
		return cs.(content.Store), nil
	}
	cs, err := local.NewStore(root)
	if err != nil {
		return nil, err
	}
	actual, _ := contentStores.LoadOrStore(root, cs)
	return actual.(content.Store), nil
}

// filteredFormatter drops log entries that are expected in this setup but
// would only confuse users. The shim's log pipe is read by whichever CLI
// process started or reconnected to it, so it reports a closed pipe when
// that process moves on; that is normal here, unlike in a daemon.
type filteredFormatter struct {
	logrus.Formatter
}

func (f *filteredFormatter) Format(e *logrus.Entry) ([]byte, error) {
	switch e.Message {
	case "copy shim log", "copy shim log after reload":
		return nil, nil
	}
	return f.Formatter.Format(e)
}

func openSession(ctx context.Context, cfg *config) (_ *session, retErr error) {
	configureLogging()
	for _, d := range []string{cfg.contentDir(), cfg.snapshotterDir(cfg.snapshotter)} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return nil, err
		}
	}
	cs, err := contentStore(cfg.contentDir())
	if err != nil {
		return nil, fmt.Errorf("opening content store: %w", err)
	}

	var sn snapshots.Snapshotter
	switch cfg.snapshotter {
	case snapshotterOverlay:
		sn, err = overlay.NewSnapshotter(cfg.snapshotterDir(cfg.snapshotter), overlay.AsynchronousRemove)
	case snapshotterNative:
		sn, err = native.NewSnapshotter(cfg.snapshotterDir(cfg.snapshotter))
	default:
		err = fmt.Errorf("unknown snapshotter %q", cfg.snapshotter)
	}
	if err != nil {
		return nil, fmt.Errorf("opening %s snapshotter: %w", cfg.snapshotter, err)
	}
	defer func() {
		if retErr != nil {
			_ = sn.Close()
		}
	}()

	bdb, err := bolt.Open(cfg.metadataPath(), 0o600, &bolt.Options{Timeout: 30 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("opening metadata database: %w", err)
	}
	defer func() {
		if retErr != nil {
			_ = bdb.Close()
		}
	}()

	db := metadata.NewDB(bdb, cs, map[string]snapshots.Snapshotter{cfg.snapshotter: sn})
	if err := db.Init(nsCtx(ctx)); err != nil {
		return nil, fmt.Errorf("initializing metadata database: %w", err)
	}
	return &session{cfg: cfg, bdb: bdb, db: db, snRw: sn}, nil
}

// Close releases the stores. It is safe to call multiple times.
func (s *session) Close() error {
	if s.db == nil {
		return nil
	}
	var errs []error
	if err := s.db.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := s.snRw.Close(); err != nil {
		errs = append(errs, err)
	}
	s.db, s.bdb, s.snRw = nil, nil, nil
	return errors.Join(errs...)
}

func (s *session) content() content.Store             { return s.db.ContentStore() }
func (s *session) snapshotter() snapshots.Snapshotter { return s.db.Snapshotter(s.cfg.snapshotter) }
func (s *session) images() images.Store               { return metadata.NewImageStore(s.db) }
func (s *session) containers() containers.Store       { return metadata.NewContainerStore(s.db) }
func (s *session) leases() leases.Manager             { return metadata.NewLeaseManager(s.db) }

// gc runs the metadata garbage collector, removing unreferenced content and
// snapshots.
func (s *session) gc(ctx context.Context) error {
	_, err := s.db.GarbageCollect(nsCtx(ctx))
	return err
}

// SnapshotService implements oci.Client so that spec options which need to
// read /etc/passwd and /etc/group from the container's root filesystem can
// mount the snapshot.
func (s *session) SnapshotService(name string) snapshots.Snapshotter {
	return s.db.Snapshotter(name)
}

// withSession runs fn with a session on the stores.
//
// Sessions are reference-counted per engine: the metadata database takes an
// exclusive file lock, which the same process cannot take twice, and some
// containerd components (the shim manager's container store) call back into
// the stores while an operation is already holding a session. Reusing the
// open session keeps those nested accesses deadlock-free, while the session
// is still closed as soon as the outermost operation finishes so that other
// CLI invocations can take the lock.
func (e *engine) withSession(ctx context.Context, fn func(ctx context.Context, s *session) error) error {
	s, err := e.acquireSession(ctx)
	if err != nil {
		return err
	}
	defer e.releaseSession()
	return fn(nsCtx(ctx), s)
}

func (e *engine) acquireSession(ctx context.Context) (*session, error) {
	e.sessMu.Lock()
	defer e.sessMu.Unlock()
	if e.sess != nil {
		e.sessRefs++
		return e.sess, nil
	}
	s, err := openSession(ctx, e.cfg)
	if err != nil {
		return nil, err
	}
	e.sess, e.sessRefs = s, 1
	return s, nil
}

func (e *engine) releaseSession() {
	e.sessMu.Lock()
	defer e.sessMu.Unlock()
	e.sessRefs--
	if e.sessRefs > 0 {
		return
	}
	if e.sess != nil {
		_ = e.sess.Close()
		e.sess = nil
	}
}

// lazyContainerStore implements containers.Store on top of the engine's
// session, opening one only for the duration of each call. It is handed to
// the shim manager, which needs occasional access to container records, so
// that no long-lived database lock is held while containers are supervised.
type lazyContainerStore struct {
	eng *engine
}

func (l lazyContainerStore) with(ctx context.Context, fn func(ctx context.Context, cs containers.Store) error) error {
	return l.eng.withSession(ctx, func(ctx context.Context, s *session) error {
		return fn(ctx, s.containers())
	})
}

func (l lazyContainerStore) Get(ctx context.Context, id string) (c containers.Container, err error) {
	err = l.with(ctx, func(ctx context.Context, cs containers.Store) error {
		c, err = cs.Get(ctx, id)
		return err
	})
	return c, err
}

func (l lazyContainerStore) List(ctx context.Context, filters ...string) (out []containers.Container, err error) {
	err = l.with(ctx, func(ctx context.Context, cs containers.Store) error {
		out, err = cs.List(ctx, filters...)
		return err
	})
	return out, err
}

func (l lazyContainerStore) Create(ctx context.Context, c containers.Container) (out containers.Container, err error) {
	err = l.with(ctx, func(ctx context.Context, cs containers.Store) error {
		out, err = cs.Create(ctx, c)
		return err
	})
	return out, err
}

func (l lazyContainerStore) Update(ctx context.Context, c containers.Container, fieldpaths ...string) (out containers.Container, err error) {
	err = l.with(ctx, func(ctx context.Context, cs containers.Store) error {
		out, err = cs.Update(ctx, c, fieldpaths...)
		return err
	})
	return out, err
}

func (l lazyContainerStore) Delete(ctx context.Context, id string) error {
	return l.with(ctx, func(ctx context.Context, cs containers.Store) error {
		return cs.Delete(ctx, id)
	})
}
