//go:build linux

package standalone

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/containerd/containerd/v2/plugins/snapshots/overlay/overlayutils"
	"github.com/containerd/log"
	"github.com/moby/sys/userns"
)

const (
	// namespace is the containerd metadata namespace used for all state.
	namespace = "docker"
	// shimRuntime is the runtime v2 shim used to supervise containers.
	shimRuntime = "io.containerd.runc.v2"

	snapshotterOverlay = "overlayfs"
	snapshotterNative  = "native"
)

// config holds the resolved configuration for the standalone engine.
type config struct {
	// root holds persistent state: content, snapshots, metadata, container
	// records, logs and volumes.
	root string
	// runDir holds transient state: shim sockets, bundles, FIFOs and
	// network namespaces. It is expected to be wiped on reboot.
	runDir string
	// runtimeBinary is the OCI runtime (crun/runc) binary name or path.
	runtimeBinary string
	// snapshotter is the containerd snapshotter used for image layers and
	// container root filesystems.
	snapshotter string
	// network is the default network backend for containers.
	network string
	// rootless is true when running without real root privileges (inside a
	// user namespace).
	rootless bool
	// systemdCgroup selects the systemd cgroup driver for the runtime.
	systemdCgroup bool
	// executable is the absolute path of this binary; used to spawn the
	// logging helper.
	executable string
}

//nolint:gocyclo // path and tool resolution for both rootful and rootless modes
func loadConfig() (*config, error) {
	cfg := &config{}
	cfg.rootless = os.Geteuid() != 0 || userns.RunningInUserNS()

	var err error
	cfg.executable, err = os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolving executable path: %w", err)
	}
	cfg.executable, err = filepath.EvalSymlinks(cfg.executable)
	if err != nil {
		return nil, fmt.Errorf("resolving executable path: %w", err)
	}

	cfg.root = os.Getenv(EnvRoot)
	cfg.runDir = os.Getenv(EnvRunDir)
	if cfg.rootless {
		if cfg.root == "" {
			dataHome := os.Getenv("XDG_DATA_HOME")
			if dataHome == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return nil, fmt.Errorf("resolving home directory: %w", err)
				}
				dataHome = filepath.Join(home, ".local", "share")
			}
			cfg.root = filepath.Join(dataHome, "docker-standalone")
		}
		if cfg.runDir == "" {
			cfg.runDir = filepath.Join(runtimeDir(), "docker-standalone")
		}
	} else {
		if cfg.root == "" {
			cfg.root = "/var/lib/docker-standalone"
		}
		if cfg.runDir == "" {
			cfg.runDir = "/run/docker-standalone"
		}
	}
	for _, d := range []string{cfg.root, cfg.runDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return nil, err
		}
	}

	cfg.runtimeBinary, err = resolveRuntime(os.Getenv(EnvRuntime))
	if err != nil {
		return nil, err
	}
	// Rootless containers can only get a cgroup through the user's systemd
	// instance; see rootlessCgroupPath.
	cfg.systemdCgroup = cfg.rootless && isCgroup2() && hasSystemdUserSession()

	cfg.snapshotter, err = resolveSnapshotter(cfg, os.Getenv(EnvSnapshotter))
	if err != nil {
		return nil, err
	}

	cfg.network = os.Getenv(EnvNetwork)
	if cfg.network == "" {
		cfg.network = defaultNetworkBackend(cfg)
	}
	return cfg, nil
}

// runtimeDir returns the XDG runtime directory, falling back to a
// user-specific directory under /tmp.
func runtimeDir() string {
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return d
	}
	if d := "/run/user/" + strconv.Itoa(os.Getuid()); isOwnedDir(d) {
		return d
	}
	return filepath.Join(os.TempDir(), "docker-standalone-"+strconv.Itoa(os.Getuid()))
}

func isOwnedDir(path string) bool {
	fi, err := os.Stat(path)
	if err != nil || !fi.IsDir() {
		return false
	}
	return true
}

// resolveRuntime finds the OCI runtime binary: runc, then crun. runc is
// preferred because older crun releases reject the runtime-spec version
// emitted by containerd's spec generator.
func resolveRuntime(requested string) (string, error) {
	candidates := []string{"runc", "crun"}
	if requested != "" {
		candidates = []string{requested}
	}
	for _, c := range candidates {
		if filepath.IsAbs(c) {
			if _, err := os.Stat(c); err == nil {
				return c, nil
			}
			continue
		}
		if p, err := exec.LookPath(c); err == nil {
			return p, nil
		}
	}
	if requested != "" {
		return "", fmt.Errorf("OCI runtime %q not found", requested)
	}
	return "", errors.New("no OCI runtime found: install crun or runc")
}

// resolveSnapshotter picks the snapshotter. overlayfs is used when the
// kernel supports it in the current (user) namespace, otherwise the native
// (copying) snapshotter is used.
//
// The choice is persisted in <root>/snapshotter: switching snapshotters
// would make previously pulled images invisible.
func resolveSnapshotter(cfg *config, requested string) (string, error) {
	switch requested {
	case snapshotterOverlay, snapshotterNative, "":
	default:
		return "", fmt.Errorf("unsupported snapshotter %q (supported: overlayfs, native)", requested)
	}
	marker := filepath.Join(cfg.root, "snapshotter")
	if data, err := os.ReadFile(marker); err == nil {
		current := strings.TrimSpace(string(data))
		if requested != "" && requested != current {
			log.L.Warnf("%s=%s ignored: this storage root uses the %s snapshotter", EnvSnapshotter, requested, current)
		}
		if current == snapshotterOverlay || current == snapshotterNative {
			return current, nil
		}
	}
	chosen := requested
	if chosen == "" {
		chosen = snapshotterOverlay
		snRoot := filepath.Join(cfg.root, "snapshots", snapshotterOverlay)
		if err := os.MkdirAll(snRoot, 0o700); err != nil {
			return "", err
		}
		if err := overlayutils.Supported(snRoot); err != nil {
			log.L.WithError(err).Debug("overlayfs snapshotter not supported, falling back to native")
			chosen = snapshotterNative
		}
	}
	if err := os.WriteFile(marker, []byte(chosen+"\n"), 0o600); err != nil {
		return "", err
	}
	return chosen, nil
}

// paths

func (c *config) containerDir(id string) string {
	return filepath.Join(c.root, "containers", id)
}

func (c *config) containerRunDir(id string) string {
	// Keep this short: unix socket paths are limited to 108 bytes.
	return filepath.Join(c.runDir, "c", shortID(id))
}

func (c *config) volumesDir() string             { return filepath.Join(c.root, "volumes") }
func (c *config) contentDir() string             { return filepath.Join(c.root, "content") }
func (c *config) metadataPath() string           { return filepath.Join(c.root, "metadata.db") }
func (c *config) snapshotterDir(n string) string { return filepath.Join(c.root, "snapshots", n) }
func (c *config) tasksRootDir() string           { return filepath.Join(c.root, "tasks") }
func (c *config) tasksStateDir() string          { return filepath.Join(c.runDir, "tasks") }
func (c *config) shimSocketDir() string          { return filepath.Join(c.runDir, "s") }
func (c *config) runcRoot() string               { return filepath.Join(c.runDir, "runc") }
func (c *config) netnsDir() string               { return filepath.Join(c.runDir, "netns") }

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
